// Package testkit holds the fakes and builders the test suites under
// backend/adapter/tests share: a stand-in for the terraform binary, a fake
// Keystone, a fake STS, the three connectors wired to them, and an in-process
// B API on a throwaway Postgres.
//
// It is a test-only helper, like internal/pgtest: no package under cmd/
// imports it, and nothing in it reaches a real cloud.
package testkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/aws"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/generalopenstack"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
)

// SecretEnv are the environment variables that carry a run's credentials.
// FakeTerraform refuses a run whose files hold any of their values.
var SecretEnv = []string{
	"OS_APPLICATION_CREDENTIAL_SECRET",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
}

var (
	resourceLine = regexp.MustCompile(`(?m)^resource "([^"]+)" "([^"]+)"`)
	crossOutput  = regexp.MustCompile(`(?m)^output "(cross_x[0-9a-f]{10}_([a-z_]+))"`)
	resultOutput = regexp.MustCompile(`(?m)^output "(result_(private_address|public_address|url|bucket|table)_[A-Za-z0-9_-]+)"`)
	statePath    = regexp.MustCompile(`path\s*=\s*"([^"]+)"`)
)

// crossValues are what a fake apply hands another cloud, by value name.
var crossValues = map[string]string{
	connectors.ValueAddress:         "198.51.100.7",
	connectors.ValueBucket:          "c1-uploads-0a1b2c3d",
	connectors.ValueTable:           "c1-orders",
	connectors.ValueRegion:          "ca-central-1",
	connectors.ValueAccessKeyID:     "AKIAFAKESCOPED000001",
	connectors.ValueSecretAccessKey: "fakeScopedSecret/0123456789abcdefABCDEF",
}

// resultValues are what a fake apply returns for each block result.
var resultValues = map[string]string{
	connectors.ResultPrivateAddress: "10.1.0.10",
	connectors.ResultPublicAddress:  "198.51.100.8",
	connectors.ResultURL:            "https://fake.example.test/v1/fn",
	connectors.ResultBucket:         "c1-uploads-0a1b2c3d",
	connectors.ResultTable:          "c1-orders",
}

// Launch is one step FakeTerraform ran.
type Launch struct {
	Args  []string
	Env   map[string]string
	Files map[string]string
	// State is the state file the run's backend names, or "".
	State string
}

// FakeTerraform stands in for the terraform binary behind B's shared
// Terraform runner. It prints what Terraform prints, writes a small state
// file on apply (holding the deployment's own marker), and refuses a run
// whose files hold a credential value.
type FakeTerraform struct {
	// Delay is added to every plan, apply and destroy step, so performance
	// tests can model a slow cloud. Zero means none.
	Delay time.Duration

	mu       sync.Mutex
	launches []Launch
}

// Launch implements runtask.Launcher.
func (f *FakeTerraform) Launch(ctx context.Context, t runtask.Task) error {
	files := map[string]string{}
	entries, err := os.ReadDir(t.Dir)
	if err != nil {
		return err
	}
	var addrs []string
	state := ""
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(t.Dir, e.Name()))
		if err != nil {
			return err
		}
		body := string(b)
		files[e.Name()] = body
		for _, k := range SecretEnv {
			if v := t.Env[k]; v != "" && strings.Contains(body, v) {
				return fmt.Errorf("%s holds the value of %s", e.Name(), k)
			}
		}
		for _, m := range resourceLine.FindAllStringSubmatch(body, -1) {
			addrs = append(addrs, m[1]+"."+m[2])
		}
		if strings.Contains(body, `backend "local"`) {
			if m := statePath.FindStringSubmatch(body); m != nil {
				state = strings.ReplaceAll(m[1], `\\`, `\`)
			}
		}
	}
	sort.Strings(addrs)
	f.mu.Lock()
	f.launches = append(f.launches, Launch{Args: append([]string(nil), t.Args...), Env: copyEnv(t.Env), Files: files, State: state})
	f.mu.Unlock()

	if f.Delay > 0 && t.Args[0] != "init" && t.Args[0] != runtask.OutputStep {
		select {
		case <-time.After(f.Delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	out := t.Output
	switch t.Args[0] {
	case "init":
		fmt.Fprintln(out, "Terraform has been successfully initialized!")
	case "plan":
		fmt.Fprintf(out, "Plan: %d to add, 0 to change, 0 to destroy.\n", len(addrs))
	case "apply":
		for _, a := range addrs {
			fmt.Fprintf(out, "%s: Creation complete after 1s [id=fake-%s]\n", a, strings.ReplaceAll(a, ".", "-"))
		}
		fmt.Fprintf(out, "Apply complete! Resources: %d added, 0 changed, 0 destroyed.\n", len(addrs))
		if state != "" {
			if err := os.MkdirAll(filepath.Dir(state), 0o700); err != nil {
				return err
			}
			body, _ := json.Marshal(map[string]any{"version": 4, "resources": addrs, "run_dir": t.Dir})
			if err := os.WriteFile(state, body, 0o600); err != nil {
				return err
			}
		}
	case "destroy":
		fmt.Fprintf(out, "Destroy complete! Resources: %d destroyed.\n", len(addrs))
	case runtask.OutputStep:
		var parts []string
		for _, body := range files {
			for _, m := range crossOutput.FindAllStringSubmatch(body, -1) {
				parts = append(parts, fmt.Sprintf(`%q: {"sensitive": %t, "type": "string", "value": %q}`,
					m[1], connectors.SensitiveValue(m[2]), crossValues[m[2]]))
			}
			for _, m := range resultOutput.FindAllStringSubmatch(body, -1) {
				parts = append(parts, fmt.Sprintf(`%q: {"sensitive": false, "type": "string", "value": %q}`, m[1], resultValues[m[2]]))
			}
		}
		sort.Strings(parts)
		fmt.Fprintf(out, "{%s}\n", strings.Join(parts, ", "))
	}
	return nil
}

// Launches returns every step run so far.
func (f *FakeTerraform) Launches() []Launch {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Launch(nil), f.launches...)
}

func copyEnv(env map[string]string) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = v
	}
	return out
}

// Runner returns a Terraform runner whose steps go to fake, with its folders
// under root.
func Runner(root string, fake runtask.Launcher) *terraform.Runner {
	return &terraform.Runner{
		Bin: filepath.Join(root, "fake-terraform"), // never run: fake launches instead

		WorkDir:        filepath.Join(root, "work"),
		StateDir:       filepath.Join(root, "state"),
		PluginCacheDir: filepath.Join(root, "plugins"),
		Launcher:       fake,
	}
}

// OpenStack credentials the fake Keystone accepts by default.
const (
	OpenStackRegion   = "RegionOne"
	OpenStackCredID   = "4c1f0e1b2a3d4e5f8a9b0c1d2e3f4a5b"
	OpenStackSecret   = "app-cred-secret-good-0123456789"
	OpenStackBadValue = "app-cred-secret-bad-9876543210"
)

// Keystone is a fake OpenStack Identity v3 service. It accepts the
// application credentials in Accept and refuses every other pair.
type Keystone struct {
	*httptest.Server
	mu     sync.Mutex
	accept map[string]string
}

// NewKeystone starts a fake Keystone that accepts OpenStackCredID with
// OpenStackSecret.
func NewKeystone(tb testing.TB) *Keystone {
	tb.Helper()
	k := &Keystone{accept: map[string]string{OpenStackCredID: OpenStackSecret}}
	k.Server = httptest.NewServer(http.HandlerFunc(k.serve))
	tb.Cleanup(k.Close)
	return k
}

// Accept adds a credential pair the fake accepts.
func (k *Keystone) Accept(id, secret string) {
	k.mu.Lock()
	k.accept[id] = secret
	k.mu.Unlock()
}

// AuthURL is the fake's Keystone v3 URL.
func (k *Keystone) AuthURL() string { return k.URL + "/identity/v3" }

// Secrets returns a connection's credentials for this fake.
func (k *Keystone) Secrets(id, secret string) connectors.Secrets {
	return connectors.Secrets{"auth_url": k.AuthURL(), "application_credential_id": id, "application_credential_secret": secret}
}

func (k *Keystone) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/identity/v3/auth/tokens" {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Auth struct {
			Identity struct {
				Methods               []string `json:"methods"`
				ApplicationCredential struct {
					ID     string `json:"id"`
					Secret string `json:"secret"`
				} `json:"application_credential"`
			} `json:"identity"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	ac := req.Auth.Identity.ApplicationCredential
	k.mu.Lock()
	want, ok := k.accept[ac.ID]
	k.mu.Unlock()
	if !ok || want != ac.Secret {
		http.Error(w, `{"error": {"code": 401}}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("X-Subject-Token", "gAAAAA-fake-token")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"token": {"methods": ["application_credential"], "catalog": [
	  {"type": "compute", "name": "nova", "endpoints": [{"interface": "public", "region": %[1]q, "region_id": %[1]q, "url": "http://x/compute"}]},
	  {"type": "identity", "name": "keystone", "endpoints": [{"interface": "public", "region": %[1]q, "region_id": %[1]q, "url": "http://x/identity"}]}
	]}}`, OpenStackRegion)
}

// AWS values the fake STS accepts.
const (
	AWSRegion       = "ca-central-1"
	AWSGoodRole     = "arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd34"
	AWSGoodExternal = "6f1c2a0e-8a43-4d4e-9d55-2b7f3c1e9a10"
	AWSKeyID        = "ASIAFAKEKEYID0000001"
	AWSSecret       = "fakeSecretAccessKey/0123456789abcdefABCDEF"
	AWSToken        = "FakeSessionToken//////0123456789abcdefABCDEF=="
)

// FakeSTS lets in only AWSGoodRole with AWSGoodExternal.
type FakeSTS struct{}

// AssumeRole implements aws.Assumer.
func (FakeSTS) AssumeRole(_ context.Context, in aws.AssumeInput) (aws.SessionKeys, error) {
	if in.RoleARN != AWSGoodRole || in.ExternalID != AWSGoodExternal {
		return aws.SessionKeys{}, errors.New("AccessDenied: not authorized to perform: sts:AssumeRole")
	}
	return aws.SessionKeys{AccessKeyID: AWSKeyID, SecretAccessKey: AWSSecret, SessionToken: AWSToken, Expires: time.Now().Add(in.Duration)}, nil
}

// AWSSecrets are credentials FakeSTS accepts (good) or refuses.
func AWSSecrets(good bool) connectors.Secrets {
	if good {
		return connectors.Secrets{aws.CredRoleARN: AWSGoodRole, aws.CredExternalID: AWSGoodExternal}
	}
	return connectors.Secrets{aws.CredRoleARN: "arn:aws:iam::210987654321:role/NotClutters", aws.CredExternalID: "not-the-external-id"}
}

// BackendDir is the repo's backend folder.
func BackendDir() string {
	_, here, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(here), "..", "..", "..")
}

// RepoRoot is the repository's root folder.
func RepoRoot() string { return filepath.Join(BackendDir(), "..") }

// NewMock returns the mock connector.
func NewMock() connectors.Connector { return mock.New() }

// NewOpenStack returns General's OpenStack connector with its Terraform runs
// going to fake. It allows private targets, because the fake Keystone listens
// on 127.0.0.1 over plain http, as A2 does for DevStack.
func NewOpenStack(tb testing.TB, fake runtask.Launcher) connectors.Connector {
	tb.Helper()
	return NewOpenStackIn(tb.TempDir(), fake)
}

// NewOpenStackIn is NewOpenStack with its run, state and plugin folders
// under root, so a test can read what the runs left there.
func NewOpenStackIn(root string, fake runtask.Launcher) connectors.Connector {
	return generalopenstack.New(Runner(root, fake), generalopenstack.AllowPrivateTargets())
}

// NewAWS returns the AWS connector rendering the repo's own templates from
// disk, with STS faked and its Terraform runs going to fake.
func NewAWS(tb testing.TB, fake runtask.Launcher) connectors.Connector {
	tb.Helper()
	l, err := tftemplate.NewTemplateLoaderFromDir(filepath.Join(BackendDir(), "infra", "modules", "s3", "uploads", "aws"))
	if err != nil {
		tb.Fatal(err)
	}
	return aws.New(Runner(tb.TempDir(), fake), l, FakeSTS{})
}

// ReadAll reads r to the end, failing tb on error.
func ReadAll(tb testing.TB, r io.Reader) string {
	tb.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		tb.Fatal(err)
	}
	return string(b)
}
