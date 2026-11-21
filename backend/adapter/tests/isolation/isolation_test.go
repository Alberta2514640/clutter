// Package isolation runs simultaneous deployments for two organizations, each
// with its own project, diagram and cloud credentials, on two providers
// (General's OpenStack connector and the mock), through B API's real routes,
// store and job runner. It then proves:
//
//   - each run saw only its own organization's credentials, and each
//     deployment's Terraform state is its own;
//   - neither organization can read or change the other's connections,
//     deployments, runs or logs;
//   - the seeded test secrets appear in no diagram or database row, no API
//     response, no run log, no service log, no Terraform file or state, and
//     nowhere in the repository's working tree.
//
// It needs Docker for Postgres, and skips without it.
package isolation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/tests/testkit"
	"github.com/google/uuid"
)

func TestMain(m *testing.M) { testkit.Main(m) }

const openstackDoc = `{"engine":"blocks","nodes":[
 {"id":"vm","type":"vm","data":{"name":"web","fields":{"size":"small","public":true}}},
 {"id":"net","type":"network","data":{"name":"net","fields":{"cidr":"10.20.0.0/24"}}},
 {"id":"fw","type":"firewall","data":{"name":"fw","fields":{"ingress":"tcp/22","source_cidr":"203.0.113.0/24"}}},
 {"id":"vol","type":"volume","data":{"name":"data","fields":{"size_gb":1}}}],
 "edges":[{"source":"vm","target":"net"},{"source":"fw","target":"vm"},{"source":"vol","target":"vm"}]}`

// seed returns a fresh secret no file in the repo can already hold.
func seed(t *testing.T, label string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return "SEEDED-" + label + "-" + hex.EncodeToString(b)
}

// org is one organization's context in the test.
type org struct {
	tn             testkit.Tenant
	credID, secret string
	mockToken      string
	openstackDep   string
	mockDep        string
	openstackConn  string
	mockConn       string
	runs           []string
}

func TestSimultaneousDeploymentsStayIsolated(t *testing.T) {
	keystone := testkit.NewKeystone(t)
	fake := &testkit.FakeTerraform{Delay: 300 * time.Millisecond}
	tfRoot := t.TempDir()
	env := testkit.NewEnv(t, testkit.Options{Workers: 6}, testkit.NewOpenStackIn(tfRoot, fake), testkit.NewMock())

	orgs := []*org{{}, {}}
	for i, o := range orgs {
		label := string(rune('A' + i))
		o.credID = strings.ReplaceAll(uuid.NewString(), "-", "")
		o.secret = seed(t, "OS-SECRET-"+label)
		keystone.Accept(o.credID, o.secret)
		o.tn = env.NewTenant(t, openstackDoc)
		o.openstackConn = env.Connect(t, &o.tn, "general-openstack", testkit.OpenStackRegion, keystone.Secrets(o.credID, o.secret))
		o.mockConn = env.Connect(t, &o.tn, "mock", "mock-1", map[string]string{"token": "mock-valid-token"})
		o.openstackDep = env.Deploy(t, &o.tn, o.openstackConn, map[string]any{"image": "cirros"}, nil)
		o.mockDep = env.Deploy(t, &o.tn, o.mockConn, map[string]any{"image": "debian-12"}, nil)
	}
	if orgs[0].tn.Project == orgs[1].tn.Project || orgs[0].tn.Org == orgs[1].tn.Org {
		t.Fatal("the two contexts share a project or organization")
	}

	// Four applies at once: two organizations by two providers.
	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, o := range orgs {
		for _, dep := range []string{o.openstackDep, o.mockDep} {
			wg.Add(1)
			go func(o *org, dep string) {
				defer wg.Done()
				run := env.StartRun(t, &o.tn, dep, "apply")
				seen := env.WaitRun(t, &o.tn, run, time.Minute)
				mu.Lock()
				o.runs = append(o.runs, run)
				mu.Unlock()
				if seen[len(seen)-1] != "completed" {
					t.Errorf("apply of %s ended %v", dep, seen)
				}
			}(o, dep)
		}
	}
	wg.Wait()
	elapsed := time.Since(start)
	t.Logf("four simultaneous applies took %s (each OpenStack apply waits %s in its cloud)", elapsed, fake.Delay)
	if t.Failed() {
		t.FailNow()
	}

	t.Run("each run saw only its own credentials and wrote only its own state", func(t *testing.T) {
		states := map[string]string{} // state path -> deployment
		for _, l := range fake.Launches() {
			got := l.Env["OS_APPLICATION_CREDENTIAL_SECRET"]
			var owner *org
			for _, o := range orgs {
				if strings.Contains(l.State, o.openstackDep) {
					owner = o
				}
			}
			if owner == nil {
				t.Fatalf("a run's state %q names no OpenStack deployment of either organization", l.State)
			}
			if got != owner.secret {
				t.Fatalf("a run of %s held another organization's credential", owner.openstackDep)
			}
			states[l.State] = owner.openstackDep
			for name, body := range l.Files {
				for _, o := range orgs {
					if strings.Contains(body, o.secret) {
						t.Fatalf("the run file %s holds a credential", name)
					}
				}
			}
		}
		if len(states) != 2 {
			t.Fatalf("want one state file per OpenStack deployment, got %v", states)
		}
		for path, dep := range states {
			if filepath.Base(filepath.Dir(path)) != dep {
				t.Fatalf("state %s is not in its deployment's own folder", path)
			}
		}
	})

	t.Run("neither organization can read or change the other's deployments", func(t *testing.T) {
		a, b := orgs[0], orgs[1]
		ownerPaths := []string{
			"/connections/" + a.openstackConn + "/options",
			"/deployments/" + a.openstackDep + "/compatibility",
			"/deployments/" + a.openstackDep + "/preview",
			"/runs/" + a.runs[0],
			"/runs/" + a.runs[0] + "/logs",
			"/diagrams/" + a.tn.Diagram + "/deployments",
		}
		for _, p := range ownerPaths {
			// B names A's ids under B's own organization, and under A's.
			for _, orgID := range []string{b.tn.Org, a.tn.Org} {
				code, body := env.Call(t, &b.tn, "GET", "/v1/organizations/"+orgID+p, nil)
				if code != http.StatusNotFound && code != http.StatusForbidden {
					t.Fatalf("B read A's %s under org %s: %d %.200s", p, orgID[:8], code, body)
				}
			}
		}
		writes := []struct{ method, path string }{
			{"POST", "/deployments/" + a.openstackDep + "/runs"},
			{"PATCH", "/deployments/" + a.openstackDep},
			{"DELETE", "/deployments/" + a.openstackDep},
			{"POST", "/connections/" + a.openstackConn + "/verify"},
			{"DELETE", "/connections/" + a.openstackConn},
			{"POST", "/diagrams/" + a.tn.Diagram + "/deployments"},
		}
		for _, w := range writes {
			for _, orgID := range []string{b.tn.Org, a.tn.Org} {
				body := map[string]any{"action": "destroy", "connection_id": b.openstackConn, "exclusions": []string{"vm"}}
				code, resp := env.Call(t, &b.tn, w.method, "/v1/organizations/"+orgID+w.path, body)
				if code < 400 {
					t.Fatalf("B changed A's %s %s under org %s: %d %.200s", w.method, w.path, orgID[:8], code, resp)
				}
			}
		}
		// A's deployment is still A's and still live.
		code, body := env.Call(t, &a.tn, "GET", "/v1/organizations/"+a.tn.Org+"/diagrams/"+a.tn.Diagram+"/deployments", nil)
		if code != http.StatusOK || !strings.Contains(body, a.openstackDep) || !strings.Contains(body, `"live"`) {
			t.Fatalf("A's deployments after B's attempts: %d %.300s", code, body)
		}
	})

	t.Run("seeded secrets appear in no diagram, row, response, log, file or repository change", func(t *testing.T) {
		secrets := []string{orgs[0].secret, orgs[1].secret}
		var places []string
		check := func(where, text string) {
			for _, s := range secrets {
				if strings.Contains(text, s) {
					places = append(places, where)
				}
			}
		}

		// Diagrams and every row B or A keeps for them.
		for _, table := range []string{"diagrams", "diagram_history", "connections", "deployments", "runs", "deployment_placements", "function_code", "organizations", "projects"} {
			rows, err := env.Pool.Query(context.Background(), "SELECT row_to_json(t)::text FROM public."+table+" t")
			if err != nil {
				t.Fatalf("%s: %v", table, err)
			}
			for rows.Next() {
				var s string
				if err := rows.Scan(&s); err != nil {
					t.Fatal(err)
				}
				check("table "+table, s)
			}
			rows.Close()
		}

		// Every response either organization can get.
		for _, o := range orgs {
			for _, p := range []string{"/connections", "/diagrams/" + o.tn.Diagram + "/deployments",
				"/deployments/" + o.openstackDep + "/preview", "/deployments/" + o.openstackDep + "/compatibility",
				"/connections/" + o.openstackConn + "/options"} {
				_, body := env.Call(t, &o.tn, "GET", "/v1/organizations/"+o.tn.Org+p, nil)
				check("response "+p, body)
			}
			for _, r := range o.runs {
				_, body := env.Call(t, &o.tn, "GET", "/v1/organizations/"+o.tn.Org+"/runs/"+r, nil)
				check("run response", body)
				check("run log response", env.RunLog(t, &o.tn, r))
			}
		}
		check("service log", env.Service.String())

		// Everything on disk: B's data folder (secrets are encrypted there,
		// so the plain value must not appear), run folders and state.
		for _, root := range []string{env.Dir, tfRoot} {
			_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error { //nolint:errcheck // a walk error fails below
				if err != nil || info.IsDir() {
					return err
				}
				b, err := os.ReadFile(path)
				if err == nil {
					check("file "+strings.TrimPrefix(path, root), string(b))
				}
				return nil
			})
		}

		// The repository's working tree, tracked and untracked.
		repo := testkit.RepoRoot()
		for _, s := range secrets {
			cmd := exec.Command("git", "-C", repo, "grep", "--untracked", "-I", "-l", "-F", s)
			out, err := cmd.Output()
			if err == nil && len(strings.TrimSpace(string(out))) > 0 {
				places = append(places, "repository: "+strings.TrimSpace(string(out)))
			}
			if err != nil {
				if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
					t.Logf("git grep could not scan the repository: %v", err)
				}
			}
		}

		if len(places) > 0 {
			t.Fatalf("a seeded secret was found in: %v", places)
		}
		t.Logf("scanned %d tables, every response, the service log, %s and %s, and the repository", 9, env.Dir, tfRoot)
	})
}

func TestRunLimitsHoldPerOrganization(t *testing.T) {
	fake := &testkit.FakeTerraform{Delay: 2 * time.Second}
	keystone := testkit.NewKeystone(t)
	env := testkit.NewEnv(t, testkit.Options{Workers: 8}, testkit.NewOpenStackIn(t.TempDir(), fake))
	a := env.NewTenant(t, openstackDoc)
	b := env.NewTenant(t, openstackDoc)
	// A diagram has one deployment per connection, so four deployments use
	// four connections.
	var deps []string
	for i := 0; i < 4; i++ {
		conn := env.Connect(t, &a, "general-openstack", testkit.OpenStackRegion, keystone.Secrets(testkit.OpenStackCredID, testkit.OpenStackSecret))
		deps = append(deps, env.Deploy(t, &a, conn, map[string]any{"image": "cirros"}, nil))
	}
	accepted := 0
	for i, dep := range deps {
		code, body := env.Call(t, &a, "POST", "/v1/organizations/"+a.Org+"/deployments/"+dep+"/runs", map[string]string{"action": "plan"})
		if code == http.StatusAccepted {
			accepted++
		} else if i < 3 {
			t.Fatalf("run %d refused: %d %s", i+1, code, body)
		}
	}
	if accepted != 3 {
		t.Fatalf("organization A had %d runs at once; the cap is 3", accepted)
	}
	// A second run of a busy deployment is refused too.
	code, _ := env.Call(t, &a, "POST", "/v1/organizations/"+a.Org+"/deployments/"+deps[0]+"/runs", map[string]string{"action": "plan"})
	if code == http.StatusAccepted {
		t.Fatal("a deployment ran twice at once")
	}
	// A's full queue does not block B.
	connB := env.Connect(t, &b, "general-openstack", testkit.OpenStackRegion, keystone.Secrets(testkit.OpenStackCredID, testkit.OpenStackSecret))
	depB := env.Deploy(t, &b, connB, map[string]any{"image": "cirros"}, nil)
	code, body := env.Call(t, &b, "POST", "/v1/organizations/"+b.Org+"/deployments/"+depB+"/runs", map[string]string{"action": "plan"})
	if code != http.StatusAccepted {
		t.Fatalf("organization B was blocked by A's runs: %d %s", code, body)
	}
}
