package aws

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
)

// TestRenderedTerraformValidates runs terraform init and validate on one
// diagram rendered for two deployments. It needs the terraform binary and the
// Terraform registry, so it runs only with CLUTTER_TF_VALIDATE=1.
func TestRenderedTerraformValidates(t *testing.T) {
	if os.Getenv("CLUTTER_TF_VALIDATE") != "1" {
		t.Skip("set CLUTTER_TF_VALIDATE=1 to run terraform validate")
	}
	bin, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("CLUTTER_TF_VALIDATE=1 but terraform is not installed")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	cache = filepath.Join(cache, "clutter-tf-plugins")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}

	c, _, _ := newTestConnector(t)
	hostile := fullGraph()
	hostile.Blocks["api"].Fields["description"] = "x\" }\n${file(\"/etc/passwd\")} %{ if true } \\"
	zipBody := testZip(t)
	cases := map[string]struct {
		g connectors.Graph
		d connectors.Deployment
	}{
		"first deployment":   {fullGraph(), testDeployment("ca-central-1")},
		"second deployment":  {fullGraph(), testDeployment("ca-central-1")},
		"other region":       {fullGraph(), testDeployment("us-west-2")},
		"hostile text":       {hostile, testDeployment("ca-central-1")},
		"default VPC no API": {defaultVPCGraph(), testDeployment("ca-central-1")},
		// The run stages the zip beside the files, as the Terraform runner does.
		"uploaded code and startup script": {withCodeAndScript(uploaded(t, nil, zipBody)), testDeployment("ca-central-1")},
		// A split deployment's AWS parts: scoped keys and an address for
		// other clouds, and values from another cloud on a VM.
		"exports to other clouds":  {exportGraph(), exportDeployment()},
		"values from other clouds": {importGraph(), importDeployment(true)},
		"VM edges on one cloud":    {vmEdgesGraph(), testDeployment("ca-central-1")},
		// The portable firewall grammar: icmp, and an IPv6 source.
		"firewall icmp from IPv4":         {firewallGraph("tcp/22, icmp", "203.0.113.0/24"), testDeployment("ca-central-1")},
		"firewall tcp and icmp from IPv6": {firewallGraph("tcp/443, udp/1000-2000, icmp", "2001:db8::/32"), testDeployment("ca-central-1")},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for n, body := range renderFor(t, c, tc.g, tc.d) {
				if err := os.WriteFile(filepath.Join(dir, n), body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			for _, a := range newModel(tc.g, tc.d).archives() {
				if err := os.WriteFile(filepath.Join(dir, terraform.ArchiveName(a.ID)), zipBody, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			env := []string{"PATH=/usr/bin:/bin", "HOME=" + dir, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1", "TF_PLUGIN_CACHE_DIR=" + cache}
			for _, args := range [][]string{{"init", "-backend=false", "-input=false", "-no-color"}, {"validate", "-no-color"}} {
				cmd := exec.Command(bin, args...)
				cmd.Dir, cmd.Env = dir, env
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("terraform %s: %v\n%s", args[0], err, out)
				}
				if args[0] == "validate" {
					if !strings.Contains(string(out), "Success! The configuration is valid.") {
						t.Fatalf("validate: %s", out)
					}
					t.Logf("%s", strings.TrimSpace(string(out)))
				}
			}
		})
	}
}

// defaultVPCGraph is a VM and firewall with no network block (the account's
// default VPC) and a lone function.
func defaultVPCGraph() connectors.Graph {
	return connectors.Graph{
		Blocks: map[string]connectors.Block{
			"vm": {ID: "vm", Kind: connectors.KindVM, Name: "box", Fields: map[string]any{"public": true}},
			"fw": {ID: "fw", Kind: connectors.KindFirewall, Name: "box-fw", Fields: map[string]any{"ingress": ""}},
			"fn": {ID: "fn", Kind: connectors.KindFunction, Name: "job", Fields: map[string]any{}},
		},
		Edges: []connectors.Edge{{Source: "fw", Target: "vm"}},
	}
}
