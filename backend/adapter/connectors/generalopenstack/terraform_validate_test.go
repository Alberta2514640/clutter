package generalopenstack

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

// TestRenderedTerraformValidates runs terraform fmt -check, init
// -backend=false and validate on the rendered sample and on a variant with
// exclusions. It downloads the OpenStack provider, so it runs only with
// CLUTTER_TF_VALIDATE=1. It needs no cloud and no credentials.
func TestRenderedTerraformValidates(t *testing.T) {
	if os.Getenv("CLUTTER_TF_VALIDATE") != "1" {
		t.Skip("set CLUTTER_TF_VALIDATE=1 to run terraform validate on the rendered files")
	}
	bin, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("terraform is not installed")
	}
	cache := t.TempDir()

	excluded := testDeployment(fixedDeployment)
	excluded.Exclusions = []string{"n3", "n4"}
	private := sampleGraph()
	private.Blocks["n1"].Fields["public"] = false
	script := sampleGraph()
	script.Blocks["n1"].Fields[connectors.FieldStartupScript] = hostileScript
	peers := twoTierGraph()
	peers.Blocks["web"].Fields[connectors.FieldSSHKey] = testSSHKey
	unrouted := twoTierGraph()
	unrouted.Blocks["back"].Fields["private_internet"] = false
	unrouted.Edges = unrouted.Edges[:4]

	cases := map[string]struct {
		g connectors.Graph
		d connectors.Deployment
	}{
		"sample":                  {sampleGraph(), testDeployment(fixedDeployment)},
		"firewall and volume out": {sampleGraph(), excluded},
		"no public address":       {private, testDeployment(fixedDeployment)},
		"startup script":          {script, testDeployment(fixedDeployment)},
		// Two networks on one router, a peer group and a key pair; then a
		// network with internet access off.
		"peers and key":      {peers, testDeployment(fixedDeployment)},
		"network off router": {unrouted, testDeployment(fixedDeployment)},
		// A split deployment's OpenStack parts.
		"values from other clouds": {importGraph(), importDeployment(true)},
		"address for other clouds": {exportGraph(), exportDeployment()},
		// The portable firewall grammar: icmp, and an IPv6 source.
		"firewall icmp from IPv4":         {firewallGraph("tcp/22, icmp", "203.0.113.0/24"), testDeployment(fixedDeployment)},
		"firewall tcp and icmp from IPv6": {firewallGraph("tcp/443, udp/1000-2000, icmp", "2001:db8::/32"), testDeployment(fixedDeployment)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			files, err := newConnector(nil).Render(tc.g, tc.d)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			for n, body := range files {
				if err := os.WriteFile(filepath.Join(dir, n), body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			for _, args := range [][]string{
				{"fmt", "-check", "-diff"},
				{"init", "-backend=false", "-input=false", "-no-color"},
				{"validate", "-no-color"},
			} {
				cmd := exec.Command(bin, args...)
				cmd.Dir = dir
				// Built from nothing, as the shared runner does; no credential is needed.
				cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + dir, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1", "TF_PLUGIN_CACHE_DIR=" + cache}
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("terraform %s: %v\n%s", args[0], err, out)
				}
				if args[0] == "validate" {
					t.Logf("%s", out)
				}
			}
		})
	}
}
