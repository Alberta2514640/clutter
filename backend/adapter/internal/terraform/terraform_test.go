package terraform

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/google/uuid"
)

var testProviders = []string{"hashicorp/aws", "terraform-provider-openstack/openstack"}

func TestCheckFilesRefusesDangerousTerraform(t *testing.T) {
	cases := map[string]string{
		"provisioner":                  "resource \"x\" \"y\" {\n  provisioner \"local-exec\" {\n    command = \"id\"\n  }\n}\n",
		"bare provisioner":             "resource x y {\n  provisioner local-exec {}\n}\n",
		"provisioner in a block":       "resource \"x\" \"y\" {\n  lifecycle {\n    provisioner \"file\" {}\n  }\n}\n",
		"dynamic provisioner":          "resource \"x\" \"y\" {\n  dynamic \"provisioner\" {\n    for_each = []\n    content {}\n  }\n}\n",
		"external":                     "data \"external\" \"x\" {\n  program = [\"sh\"]\n}\n",
		"bare external":                "data external x {}\n",
		"http":                         "data \"http\" \"x\" {\n  url = \"https://example.com\"\n}\n",
		"module":                       "module \"x\" {\n  source = \"git::https://example.com/x\"\n}\n",
		"backend":                      "terraform {\n  backend \"s3\" {}\n}\n",
		"cloud":                        "terraform {\n  cloud {\n    organization = \"x\"\n  }\n}\n",
		"provider not allowed":         "terraform {\n  required_providers {\n    evil = {\n      source = \"evilcorp/evil\"\n    }\n  }\n}\n",
		"provider on another registry": "terraform {\n  required_providers {\n    aws = {\n      source = \"example.com/hashicorp/aws\"\n    }\n  }\n}\n",
		"implied provider not allowed": "terraform {\n  required_providers {\n    random = {\n      version = \"~> 3.0\"\n    }\n  }\n}\n",
		"old version-only form":        "terraform {\n  required_providers {\n    random = \"~> 3.0\"\n  }\n}\n",
		"source from a reference":      "terraform {\n  required_providers {\n    aws = {\n      source = var.x\n    }\n  }\n}\n",
		"does not parse":               "resource \"x\" {\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			err := CheckFiles(map[string][]byte{"main.tf": []byte(body)}, testProviders)
			if !errors.Is(err, ErrRefused) {
				t.Fatalf("accepted: %v", err)
			}
		})
	}
	for _, name := range []string{"main.tf.json", "../main.tf", "Main.tf", "main.sh", "terraform.tfvars", ".terraformrc"} {
		if err := CheckFiles(map[string][]byte{name: []byte("")}, testProviders); err == nil {
			t.Fatalf("file name %q accepted", name)
		}
	}
	if err := CheckFiles(map[string][]byte{"main.tf": []byte("")}, []string{"evil/../x"}); err == nil {
		t.Fatal("a malformed allowed provider was accepted")
	}

	// Text inside strings, comments and heredocs is not a block, and an
	// allowed provider in any of its spellings passes.
	ok := map[string]string{
		"plain": "resource \"terraform_data\" \"n1\" {\n  input = \"web\"\n}\ndata \"openstack_images_image_v2\" \"x\" {}\n",
		"words in strings": "resource \"aws_api_gateway_rest_api\" \"a\" {\n  name        = \"module x\"\n  description = \"module x provisioner \\\"local-exec\\\" data \\\"http\\\" \\\"y\\\" backend \\\"s3\\\" {}\"\n}\n" +
			"locals {\n  x = \"remote-exec\"\n  y = <<EOT\nprovisioner \"local-exec\" {}\nmodule \"m\" {}\nEOT\n}\n# module \"c\" {}\n",
		"allowed providers": "terraform {\n  required_providers {\n    aws = {\n      source  = \"hashicorp/aws\"\n      version = \"~> 5.0\"\n    }\n" +
			"    openstack = {\n      source = \"Registry.Terraform.io/terraform-provider-openstack/openstack\"\n    }\n  }\n}\n",
		"implied allowed provider":    "terraform {\n  required_providers {\n    aws = \"~> 5.0\"\n  }\n}\n",
		"data source named like http": "data \"aws_iam_policy_document\" \"http\" {}\n",
	}
	for name, body := range ok {
		if err := CheckFiles(map[string][]byte{"main.tf": []byte(body)}, testProviders); err != nil {
			t.Errorf("%s refused: %v", name, err)
		}
	}
}

func TestCLIConfigInstallsOnlyTheAllowedProviders(t *testing.T) {
	allowed, err := allowedSet([]string{"hashicorp/aws", "registry.terraform.io/hashicorp/random"})
	if err != nil {
		t.Fatal(err)
	}
	got := string(cliConfig("/run", allowed))
	for _, want := range []string{"direct {", `"registry.terraform.io/hashicorp/aws",`, `"registry.terraform.io/hashicorp/random",`} {
		if !strings.Contains(got, want) {
			t.Fatalf("config lacks %q:\n%s", want, got)
		}
	}
	// An empty include list would mean every provider, so none allowed means
	// an empty mirror and no direct installation at all.
	none := string(cliConfig("/run", map[string]bool{}))
	if strings.Contains(none, "direct") || !strings.Contains(none, "filesystem_mirror") {
		t.Fatalf("config with no providers:\n%s", none)
	}
}

// failingLauncher fails the step named in failAt and runs the others.
type failingLauncher struct {
	failAt string
	steps  []string
}

func (f *failingLauncher) Launch(_ context.Context, t runtask.Task) error {
	f.steps = append(f.steps, t.Args[0])
	if t.Args[0] == f.failAt {
		return errors.New("exit status 1")
	}
	return nil
}

func TestOnlyFailuresAfterInitMayHaveBuiltSomething(t *testing.T) {
	plain := map[string][]byte{"main.tf": []byte("resource \"terraform_data\" \"n1\" {}\n")}
	job := func(files map[string][]byte, env map[string]string) Job {
		return Job{DeploymentID: uuid.NewString(), RunID: uuid.NewString(), Action: connectors.ActionApply, Files: files, Env: env, Log: &bytes.Buffer{}}
	}
	cases := []struct {
		name    string
		failAt  string
		job     Job
		nothing bool
	}{
		{"refused file", "", job(map[string][]byte{"main.tf": []byte("module \"m\" {}\n")}, nil), true},
		{"refused variable", "", job(plain, map[string]string{"TF_LOG": "TRACE"}), true},
		{"bad run id", "", Job{DeploymentID: uuid.NewString(), RunID: "x", Action: connectors.ActionApply, Files: plain}, true},
		{"init fails", "init", job(plain, nil), true},
		{"apply fails", "apply", job(plain, nil), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRunner(t, "/bin/terraform-not-run", &failingLauncher{failAt: tc.failAt})
			err := r.Run(context.Background(), tc.job)
			if err == nil {
				t.Fatal("no error")
			}
			if got := errors.Is(err, connectors.ErrNothingBuilt); got != tc.nothing {
				t.Fatalf("nothing built is %t, want %t: %v", got, tc.nothing, err)
			}
		})
	}

	// A run that gives up waiting for the deployment's lock built nothing.
	r := newRunner(t, "/bin/terraform-not-run", &failingLauncher{})
	j := job(plain, nil)
	unlock, err := r.lock(context.Background(), j.DeploymentID)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := r.Run(ctx, j); !errors.Is(err, connectors.ErrNothingBuilt) {
		t.Fatalf("lock wait: %v", err)
	}
}

func TestCheckEnvRefusesTerraformSwitches(t *testing.T) {
	for _, k := range []string{"TF_LOG", "TF_LOG_PROVIDER", "TF_CLI_ARGS_plan", "TF_DATA_DIR"} {
		if err := CheckEnv(map[string]string{k: "x"}); err == nil {
			t.Fatalf("%s accepted", k)
		}
	}
	if err := CheckEnv(map[string]string{"TF_VAR_token": "x", "OS_AUTH_URL": "x"}); err != nil {
		t.Fatalf("plain variables refused: %v", err)
	}
}

// A fake terraform that records its environment and arguments.
func fakeTerraform(t *testing.T) (bin, record string) {
	t.Helper()
	dir := t.TempDir()
	record = filepath.Join(dir, "record")
	bin = filepath.Join(dir, "terraform")
	script := "#!/bin/sh\n{ echo \"ARGS $*\"; env; } >> '" + record + "'\necho fake terraform $1\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return bin, record
}

func newRunner(t *testing.T, bin string, l runtask.Launcher) *Runner {
	root := t.TempDir()
	return &Runner{
		Bin:            bin,
		WorkDir:        filepath.Join(root, "work"),
		StateDir:       filepath.Join(root, "state"),
		PluginCacheDir: filepath.Join(root, "plugins"),
		Launcher:       l,
	}
}

func TestRunGivesTerraformOnlyTheRunsEnvironment(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	t.Setenv("ADAPTER_DATABASE_URL", "postgres://user:db-secret@host/db")
	t.Setenv("ADAPTER_CREDENTIAL_KEY", "key-secret")
	t.Setenv("TF_LOG", "TRACE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "platform-secret")

	bin, record := fakeTerraform(t)
	r := newRunner(t, bin, runtask.Local{})
	var log bytes.Buffer
	j := Job{
		DeploymentID: uuid.NewString(), RunID: uuid.NewString(), Action: connectors.ActionPlan,
		Files: map[string][]byte{"main.tf": []byte("resource \"terraform_data\" \"n1\" {}\n")},
		Env:   map[string]string{"OS_APPLICATION_CREDENTIAL_SECRET": "run-secret"},
		Log:   &log,
	}
	if err := r.Run(context.Background(), j); err != nil {
		t.Fatalf("run: %v", err)
	}
	b, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, leak := range []string{"db-secret", "key-secret", "platform-secret", "TF_LOG=", "ADAPTER_DATABASE_URL"} {
		if strings.Contains(got, leak) {
			t.Fatalf("terraform saw %q", leak)
		}
	}
	for _, want := range []string{"OS_APPLICATION_CREDENTIAL_SECRET=run-secret", "TF_IN_AUTOMATION=1", "ARGS init", "ARGS plan",
		"TF_CLI_CONFIG_FILE=" + filepath.Join(r.WorkDir, j.RunID, cliConfigName)} {
		if !strings.Contains(got, want) {
			t.Fatalf("terraform did not see %q", want)
		}
	}
	if !strings.Contains(log.String(), "fake terraform plan") {
		t.Fatalf("log is %q", log.String())
	}
	if _, err := os.Stat(filepath.Join(r.WorkDir, j.RunID)); !os.IsNotExist(err) {
		t.Fatal("the run's work folder was not removed")
	}
}

// overlapLauncher counts how many tasks run at once.
type overlapLauncher struct {
	now, max atomic.Int32
}

func (o *overlapLauncher) Launch(context.Context, runtask.Task) error {
	n := o.now.Add(1)
	for {
		m := o.max.Load()
		if n <= m || o.max.CompareAndSwap(m, n) {
			break
		}
	}
	time.Sleep(20 * time.Millisecond)
	o.now.Add(-1)
	return nil
}

func TestRunLocksEachDeployment(t *testing.T) {
	l := &overlapLauncher{}
	r := newRunner(t, "/bin/terraform-not-run", l)
	dep := uuid.NewString()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j := Job{DeploymentID: dep, RunID: uuid.NewString(), Action: connectors.ActionPlan,
				Files: map[string][]byte{"main.tf": []byte("")}, Log: &bytes.Buffer{}}
			if err := r.Run(context.Background(), j); err != nil {
				t.Errorf("run: %v", err)
			}
		}()
	}
	wg.Wait()
	if l.max.Load() != 1 {
		t.Fatalf("%d runs of one deployment overlapped", l.max.Load())
	}
}

// TestRealTerraformKeepsStatePerDeployment runs the real terraform binary with
// the built-in terraform_data resource, so it needs no provider and no network.
func TestRealTerraformKeepsStatePerDeployment(t *testing.T) {
	bin, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform is not installed")
	}
	bin, err = filepath.Abs(bin)
	if err != nil {
		t.Fatal(err)
	}
	r := newRunner(t, bin, runtask.Local{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	files := map[string][]byte{"main.tf": []byte("resource \"terraform_data\" \"n1\" {\n  input = \"web\"\n}\n")}

	a, b := uuid.NewString(), uuid.NewString()
	for _, dep := range []string{a, b} {
		var log bytes.Buffer
		err := r.Run(ctx, Job{DeploymentID: dep, RunID: uuid.NewString(), Action: connectors.ActionApply, Files: files, Log: &log})
		if err != nil {
			t.Fatalf("apply: %v\n%s", err, log.String())
		}
		if _, err := os.Stat(filepath.Join(r.StateDir, dep, "terraform.tfstate")); err != nil {
			t.Fatalf("no state for deployment: %v", err)
		}
	}
	var log bytes.Buffer
	if err := r.Run(ctx, Job{DeploymentID: a, RunID: uuid.NewString(), Action: connectors.ActionDestroy, Files: files, Log: &log}); err != nil {
		t.Fatalf("destroy: %v\n%s", err, log.String())
	}
	if !strings.Contains(log.String(), "1 destroyed") {
		t.Fatalf("destroy log: %s", log.String())
	}

	// A provider the check cannot see (implied by a resource type) is still
	// refused by Terraform itself, through the run's CLI configuration,
	// before it asks any registry.
	log.Reset()
	implied := map[string][]byte{"main.tf": []byte("resource \"evil_thing\" \"x\" {}\n")}
	err = r.Run(ctx, Job{DeploymentID: uuid.NewString(), RunID: uuid.NewString(), Action: connectors.ActionPlan,
		Files: implied, Providers: []string{"hashicorp/aws"}, Log: &log})
	if !errors.Is(err, connectors.ErrNothingBuilt) || !strings.Contains(log.String(), "hashicorp/evil") ||
		!strings.Contains(log.String(), "search locations") {
		t.Fatalf("an implied provider outside the allow-list: %v\n%s", err, log.String())
	}
	sb, err := os.ReadFile(filepath.Join(r.StateDir, b, "terraform.tfstate"))
	if err != nil || !strings.Contains(string(sb), "terraform_data") {
		t.Fatal("destroying one deployment touched another's state")
	}
}
