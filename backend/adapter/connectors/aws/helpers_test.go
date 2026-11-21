package aws

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
	"github.com/google/uuid"
)

// backendDir is the repo's backend folder.
func backendDir() string {
	_, here, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(here), "..", "..", "..")
}

// templatesDir is the folder the platform Terraform uploads to templates/aws:
// A's five template folders and B's three. Tests render these files, so a
// change to A's templates that breaks B fails here.
func templatesDir() string {
	return filepath.Join(backendDir(), "infra", "modules", "s3", "uploads", "aws")
}

func diskTemplates(t *testing.T) *tftemplate.TemplateLoader {
	t.Helper()
	l, err := tftemplate.NewTemplateLoaderFromDir(templatesDir())
	if err != nil {
		t.Fatal(err)
	}
	return l
}

const (
	goodRole     = "arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd34"
	goodExternal = "6f1c2a0e-8a43-4d4e-9d55-2b7f3c1e9a10"
	fakeKeyID    = "ASIAFAKEKEYID0000001"
	fakeSecret   = "fakeSecretAccessKey/0123456789abcdefABCDEF"
	fakeToken    = "FakeSessionToken//////0123456789abcdefABCDEF=="
)

// fakeSTS lets in only goodRole with goodExternal.
type fakeSTS struct {
	mu    sync.Mutex
	calls []AssumeInput
}

func (f *fakeSTS) AssumeRole(_ context.Context, in AssumeInput) (SessionKeys, error) {
	f.mu.Lock()
	f.calls = append(f.calls, in)
	f.mu.Unlock()
	if in.RoleARN != goodRole || in.ExternalID != goodExternal {
		return SessionKeys{}, errors.New("AccessDenied: User: arn:aws:sts::000000000000:assumed-role/x is not authorized to perform: sts:AssumeRole")
	}
	return SessionKeys{AccessKeyID: fakeKeyID, SecretAccessKey: fakeSecret, SessionToken: fakeToken, Expires: time.Now().Add(in.Duration)}, nil
}

// fakeTerraform stands in for the terraform binary behind runtask: it checks
// the run's folder and environment, and prints what Terraform would.
type fakeTerraform struct {
	t      *testing.T
	mu     sync.Mutex
	args   [][]string
	envs   []map[string]string
	leaked bool
}

var resourceLine = regexp.MustCompile(`(?m)^resource "([^"]+)" "([^"]+)"`)

func (f *fakeTerraform) Launch(_ context.Context, task runtask.Task) error {
	f.mu.Lock()
	f.args = append(f.args, task.Args)
	f.envs = append(f.envs, task.Env)
	f.mu.Unlock()

	entries, err := os.ReadDir(task.Dir)
	if err != nil {
		return err
	}
	var addrs []string
	for _, e := range entries {
		body, err := os.ReadFile(filepath.Join(task.Dir, e.Name()))
		if err != nil {
			return err
		}
		for _, v := range []string{fakeKeyID, fakeSecret, fakeToken} {
			if strings.Contains(string(body), v) {
				f.mu.Lock()
				f.leaked = true
				f.mu.Unlock()
			}
		}
		if e.Name() == "main.tf" {
			for _, m := range resourceLine.FindAllStringSubmatch(string(body), -1) {
				addrs = append(addrs, m[1]+"."+m[2])
			}
		}
	}
	sort.Strings(addrs)

	out := task.Output
	switch task.Args[0] {
	case "init":
		fmt.Fprintln(out, "Terraform has been successfully initialized!")
	case "plan":
		// A provider that printed a key would be caught by the connector's log.
		fmt.Fprintf(out, "debug: using key %s\n", task.Env["AWS_SECRET_ACCESS_KEY"])
		fmt.Fprintf(out, "Plan: %d to add, 0 to change, 0 to destroy.\n", len(addrs))
	case "apply":
		for _, a := range addrs {
			fmt.Fprintf(out, "%s: Creation complete after 1s [id=fake-%s]\n", a, strings.ReplaceAll(a, ".", "-"))
		}
		fmt.Fprintf(out, "Apply complete! Resources: %d added, 0 changed, 0 destroyed.\n", len(addrs))
	case "destroy":
		fmt.Fprintln(out, "Destroy complete! Resources: 0 destroyed.")
	case runtask.OutputStep:
		fmt.Fprint(out, fakeOutputs(task.Dir))
	}
	return nil
}

// fakeScopedSecret is the secret of the scoped key a fake apply returns.
const fakeScopedSecret = "fakeScopedSecret/0123456789abcdefABCDEF"

var (
	crossOutputLine  = regexp.MustCompile(`(?m)^output "(cross_x[0-9a-f]{10}_([a-z_]+))"`)
	resultOutputLine = regexp.MustCompile(`(?m)^output "(result_(private_address|public_address|url|bucket|table)_[A-Za-z0-9_-]+)"`)
)

// fakeResults are the values a fake apply returns for each result.
var fakeResults = map[string]string{
	connectors.ResultPrivateAddress: "10.1.0.10",
	connectors.ResultPublicAddress:  "198.51.100.7",
	connectors.ResultURL:            "https://fake.execute-api.ca-central-1.amazonaws.com/v1/api_handler",
	connectors.ResultBucket:         "c1-uploads-0a1b2c3d",
	connectors.ResultTable:          "c1-orders",
}

// fakeOutputs is `terraform output -json` for every cross_ and result_
// output the run's files declare.
func fakeOutputs(dir string) string {
	values := map[string]string{
		connectors.ValueAddress:         "198.51.100.7",
		connectors.ValueBucket:          "c1-uploads-0a1b2c3d",
		connectors.ValueTable:           "c1-orders",
		connectors.ValueRegion:          "ca-central-1",
		connectors.ValueAccessKeyID:     "AKIAFAKESCOPED000001",
		connectors.ValueSecretAccessKey: fakeScopedSecret,
	}
	entries, _ := os.ReadDir(dir)
	var parts []string
	for _, e := range entries {
		body, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		for _, m := range crossOutputLine.FindAllStringSubmatch(string(body), -1) {
			parts = append(parts, fmt.Sprintf(`%q: {"sensitive": %t, "type": "string", "value": %q}`,
				m[1], connectors.SensitiveValue(m[2]), values[m[2]]))
		}
		for _, m := range resultOutputLine.FindAllStringSubmatch(string(body), -1) {
			parts = append(parts, fmt.Sprintf(`%q: {"sensitive": false, "type": "string", "value": %q}`, m[1], fakeResults[m[2]]))
		}
	}
	sort.Strings(parts)
	return "{\n" + strings.Join(parts, ",\n") + "\n}\n"
}

func newTestConnector(t *testing.T) (*Connector, *fakeSTS, *fakeTerraform) {
	t.Helper()
	root := t.TempDir()
	ft := &fakeTerraform{t: t}
	tf := &terraform.Runner{
		Bin:            "/usr/bin/false",
		WorkDir:        filepath.Join(root, "work"),
		StateDir:       filepath.Join(root, "state"),
		PluginCacheDir: filepath.Join(root, "plugins"),
		Launcher:       ft,
	}
	fs := &fakeSTS{}
	return New(tf, diskTemplates(t), fs), fs, ft
}

// fullGraph has every block and every edge the manifest declares, two APIs on
// one function, and a node id with an underscore (as React Flow makes them).
func fullGraph() connectors.Graph {
	blocks := []connectors.Block{
		{ID: "net1", Kind: connectors.KindNetwork, Name: "main-net", Fields: map[string]any{"cidr": "10.1.0.0/24"}},
		{ID: "web", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"size": "medium", "public": true}},
		{ID: "worker", Kind: connectors.KindVM, Name: "worker", Fields: map[string]any{}},
		{ID: "fw", Kind: connectors.KindFirewall, Name: "web-fw", Fields: map[string]any{"ingress": "tcp/22, tcp/443,udp/1000-2000", "source_cidr": "203.0.113.0/24"}},
		{ID: "dndnode_3", Kind: connectors.KindVolume, Name: "data", Fields: map[string]any{"size_gb": float64(20)}},
		{ID: "logs", Kind: connectors.KindVolume, Name: "logs", Fields: map[string]any{}},
		{ID: "bucket", Kind: connectors.KindBucket, Name: "uploads", Fields: map[string]any{"versioning": true}},
		{ID: "fn", Kind: connectors.KindFunction, Name: "api_handler", Fields: map[string]any{"memory_mb": float64(256), "timeout_s": float64(10)}},
		{ID: "table", Kind: connectors.KindTable, Name: "orders", Fields: map[string]any{"key": "order_id", "key_type": "number"}},
		{ID: "api", Kind: connectors.KindHTTPAPI, Name: "public-api", Fields: map[string]any{"description": "Orders API", "cors": true}},
		{ID: "api2", Kind: connectors.KindHTTPAPI, Name: "admin-api", Fields: map[string]any{"cors": false}},
	}
	g := connectors.Graph{Blocks: map[string]connectors.Block{}}
	for _, b := range blocks {
		g.Blocks[b.ID] = b
	}
	g.Edges = []connectors.Edge{
		{Source: "web", Target: "net1"},
		{Source: "worker", Target: "net1"},
		{Source: "fw", Target: "web"},
		{Source: "fw", Target: "worker"},
		{Source: "dndnode_3", Target: "web"},
		{Source: "logs", Target: "web"},
		{Source: "fn", Target: "bucket"},
		{Source: "fn", Target: "table"},
		{Source: "api", Target: "fn"},
		{Source: "api2", Target: "fn"},
	}
	return g
}

func testDeployment(region string) connectors.Deployment {
	id := uuid.NewString()
	return connectors.Deployment{
		ID:     id,
		Prefix: connectors.PrefixFor(id),
		Connection: connectors.Connection{
			ID: uuid.NewString(), ConnectorID: ID, Region: region,
			Credentials: connectors.Secrets{CredRoleARN: goodRole, CredExternalID: goodExternal},
		},
		Settings: map[string]any{"function_runtime": "python3.12"},
	}
}

// renderFor validates and renders g for d, failing the test on any problem.
func renderFor(t *testing.T, c *Connector, g connectors.Graph, d connectors.Deployment) map[string][]byte {
	t.Helper()
	if ps := c.Validate(g, d); len(ps) > 0 {
		t.Fatalf("problems: %+v", ps)
	}
	files, err := newModel(g, d).render(context.Background(), c.templates)
	if err != nil {
		t.Fatal(err)
	}
	if err := terraform.CheckFiles(files, providers); err != nil {
		t.Fatalf("the runner refuses the rendered files: %v", err)
	}
	return files
}

func joined(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		b.WriteString("### " + n + "\n")
		b.Write(files[n])
	}
	return b.String()
}
