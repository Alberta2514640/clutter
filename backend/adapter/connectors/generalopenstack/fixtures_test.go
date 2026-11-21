package generalopenstack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
)

const (
	testRegion      = "RegionOne"
	goodCredID      = "4c1f0e1b2a3d4e5f8a9b0c1d2e3f4a5b"
	goodSecret      = "app-cred-secret-good-0123456789"
	badSecret       = "app-cred-secret-bad-9876543210"
	missingID       = "0000000000000000000000000000dead"
	fixedDeployment = "3f2a9c1e-7b4d-4e2a-9c1e-000000000001"
)

// sampleGraph is one diagram with every block this connector builds and
// every edge it wires: a public VM on a network, with a firewall and a volume.
func sampleGraph() connectors.Graph {
	return connectors.Graph{
		Blocks: map[string]connectors.Block{
			"n1": {ID: "n1", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"size": "small", "public": true}},
			"n2": {ID: "n2", Kind: connectors.KindNetwork, Name: "net", Fields: map[string]any{"cidr": "10.20.0.0/24"}},
			"n3": {ID: "n3", Kind: connectors.KindFirewall, Name: "fw", Fields: map[string]any{"ingress": "tcp/22, tcp/80, udp/1000-2000, icmp", "source_cidr": "0.0.0.0/0"}},
			"n4": {ID: "n4", Kind: connectors.KindVolume, Name: "data", Fields: map[string]any{"size_gb": float64(1)}},
		},
		Edges: []connectors.Edge{
			{Source: "n1", Target: "n2"},
			{Source: "n3", Target: "n1"},
			{Source: "n4", Target: "n1"},
		},
	}
}

func sampleSettings() map[string]any {
	return map[string]any{"image": "cirros", "flavor_small": "m1.tiny"}
}

func testDeployment(id string) connectors.Deployment {
	return connectors.Deployment{
		ID:         id,
		Prefix:     connectors.PrefixFor(id),
		Connection: connectors.Connection{ID: "conn", ConnectorID: ID, Region: testRegion},
		Settings:   sampleSettings(),
	}
}

// keystone is a fake Identity v3 service. It accepts goodCredID with
// goodSecret, answers 404 for missingID, 401 for any other pair, and, when
// hang is set, holds the request until the client gives up.
type keystone struct {
	*httptest.Server
	mu   sync.Mutex
	hang bool
}

func newKeystone(t *testing.T) *keystone {
	k := &keystone{}
	k.Server = httptest.NewServer(http.HandlerFunc(k.serve))
	t.Cleanup(k.Close)
	return k
}

func (k *keystone) authURL() string { return k.URL + "/identity/v3" }

func (k *keystone) setHang(v bool) {
	k.mu.Lock()
	k.hang = v
	k.mu.Unlock()
}

func (k *keystone) serve(w http.ResponseWriter, r *http.Request) {
	k.mu.Lock()
	hang := k.hang
	k.mu.Unlock()
	if hang {
		// Reading the body lets the server notice when the client leaves.
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
		return
	}
	if r.Method != http.MethodPost || r.URL.Path != "/identity/v3/auth/tokens" {
		http.NotFound(w, r)
		return
	}
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Auth.Identity.Methods) != 1 ||
		req.Auth.Identity.Methods[0] != "application_credential" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	ac := req.Auth.Identity.ApplicationCredential
	switch {
	case ac.ID == missingID:
		http.Error(w, `{"error": {"code": 404}}`, http.StatusNotFound)
		return
	case ac.ID != goodCredID || ac.Secret != goodSecret:
		http.Error(w, `{"error": {"code": 401}}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("X-Subject-Token", "gAAAAA-fake-token")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"token": {"methods": ["application_credential"], "catalog": [
	  {"type": "compute", "name": "nova", "endpoints": [{"interface": "public", "region": %[1]q, "region_id": %[1]q, "url": "http://x/compute"}]},
	  {"type": "identity", "name": "keystone", "endpoints": [{"interface": "public", "region": %[1]q, "region_id": %[1]q, "url": "http://x/identity"},
	    {"interface": "public", "region": "RegionTwo", "region_id": "RegionTwo", "url": "http://y/identity"},
	    {"interface": "public", "region": "bad region\"", "region_id": "bad region\"", "url": "http://z/identity"}]}
	]}}`, testRegion)
}

func (k *keystone) secrets(secret string) connectors.Secrets {
	return connectors.Secrets{credAuthURL: k.authURL(), credID: goodCredID, credSecret: secret}
}

// fakeTerraform stands in for the terraform binary behind the shared runner.
// It refuses a run whose files hold a credential value or whose environment
// lacks one, and prints what Terraform prints for each step.
type fakeTerraform struct {
	mu    sync.Mutex
	tasks []runtask.Task
	files []map[string]string
}

func (f *fakeTerraform) Launch(_ context.Context, t runtask.Task) error {
	files := map[string]string{}
	entries, err := os.ReadDir(t.Dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(t.Dir, e.Name()))
		if err != nil {
			return err
		}
		files[e.Name()] = string(b)
		if secret := t.Env["OS_APPLICATION_CREDENTIAL_SECRET"]; secret != "" && strings.Contains(string(b), secret) {
			return fmt.Errorf("%s holds the credential secret", e.Name())
		}
	}
	f.mu.Lock()
	f.tasks = append(f.tasks, t)
	f.files = append(f.files, files)
	f.mu.Unlock()

	switch t.Args[0] {
	case "init":
		fmt.Fprintln(t.Output, "Terraform has been successfully initialized!")
	case "plan":
		fmt.Fprintln(t.Output, "  # openstack_compute_instance_v2.b_n1 will be created")
		fmt.Fprint(t.Output, "Plan: 11 to add, 0 to ")
		fmt.Fprintln(t.Output, "change, 0 to destroy.")
	case "apply":
		// n1 is built and n2 changed; n3 is left as it was, so only its
		// output is printed.
		fmt.Fprintln(t.Output, "openstack_compute_instance_v2.b_n1: Creation complete after 12s [id=9b1d5a8e-0000-4000-8000-0000000000a1]")
		fmt.Fprintln(t.Output, "openstack_networking_network_v2.b_n2: Modifications complete after 1s [id=9b1d5a8e-0000-4000-8000-0000000000a2]")
		fmt.Fprintln(t.Output, "openstack_networking_router_interface_v2.b_n2: Creation complete after 3s [id=9b1d5a8e-0000-4000-8000-0000000000b2]")
		fmt.Fprintln(t.Output, "Apply complete! Resources: 2 added, 1 changed, 0 destroyed.")
		fmt.Fprint(t.Output, "\nOutputs:\n\n")
		fmt.Fprintln(t.Output, `address_n1 = "172.24.4.10"`)
		fmt.Fprintln(t.Output, `id_n1 = "9b1d5a8e-0000-4000-8000-0000000000a1"`)
		fmt.Fprintln(t.Output, `id_n2 = "9b1d5a8e-0000-4000-8000-0000000000a2"`)
		fmt.Fprintln(t.Output, `id_n3 = "9b1d5a8e-0000-4000-8000-0000000000a3"`)
	case "destroy":
		fmt.Fprintln(t.Output, "Destroy complete! Resources: 11 destroyed.")
	case runtask.OutputStep:
		// Every cross_ output the run's files declare, as a public address.
		var parts []string
		for _, body := range files {
			for _, line := range strings.Split(body, "\n") {
				if name, ok := strings.CutPrefix(line, `output "cross_`); ok {
					name = "cross_" + strings.TrimSuffix(name, `" {`)
					parts = append(parts, fmt.Sprintf(`%q: {"sensitive": false, "type": "string", "value": "172.24.4.77"}`, name))
				}
				// Every result_ output, as a private or a public address.
				if name, ok := strings.CutPrefix(line, `output "result_`); ok {
					name = "result_" + strings.TrimSuffix(name, `" {`)
					value := "10.0.0.12"
					if strings.HasPrefix(name, "result_public_address_") {
						value = "172.24.4.10"
					}
					parts = append(parts, fmt.Sprintf(`%q: {"sensitive": false, "type": "string", "value": %q}`, name, value))
				}
			}
		}
		fmt.Fprintf(t.Output, "{%s}\n", strings.Join(parts, ", "))
	}
	return nil
}

func (f *fakeTerraform) last() (runtask.Task, map[string]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tasks[len(f.tasks)-1], f.files[len(f.files)-1]
}

// newTestConnector returns a connector whose Terraform runs go to fake. It
// allows private targets, because the fake Keystone listens on 127.0.0.1
// over plain http.
func newTestConnector(t *testing.T, fake runtask.Launcher) *Connector {
	root := t.TempDir()
	return newConnector(&terraform.Runner{
		Bin:            "/fake/terraform",
		WorkDir:        filepath.Join(root, "work"),
		StateDir:       filepath.Join(root, "state"),
		PluginCacheDir: filepath.Join(root, "plugins"),
		Launcher:       fake,
	}, AllowPrivateTargets())
}
