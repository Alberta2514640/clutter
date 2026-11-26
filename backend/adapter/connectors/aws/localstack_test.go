package aws

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/jobs"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/google/uuid"
)

// localstackEndpoint returns the LocalStack endpoint the gated tests use, or
// skips the test when CLUTTER_LOCALSTACK is not set.
func localstackEndpoint(t *testing.T) string {
	t.Helper()
	if os.Getenv("CLUTTER_LOCALSTACK") == "" {
		t.Skip("set CLUTTER_LOCALSTACK=1 and start LocalStack (npm run localstack -- start) to run against it")
	}
	endpoint := os.Getenv("CLUTTER_LOCALSTACK_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:4566"
	}
	ep, err := localEndpoint(endpoint)
	if err != nil {
		t.Fatalf("CLUTTER_LOCALSTACK_ENDPOINT: %v", err)
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(ep + "/_localstack/health")
	if err != nil {
		t.Fatalf("LocalStack does not answer at %s (npm run localstack -- start): %v", ep, err)
	}
	resp.Body.Close()
	return ep
}

// lsTerraform runs terraform in one folder against LocalStack, with an
// environment built from nothing: LocalStack's test keys, never this
// machine's AWS credentials.
type lsTerraform struct {
	t        *testing.T
	bin, dir string
	env      []string
}

func newLSTerraform(t *testing.T, bin, dir, endpoint, pluginCache string, vars map[string]string) *lsTerraform {
	env := []string{
		"PATH=/usr/bin:/bin", "HOME=" + dir, "TF_IN_AUTOMATION=1", "TF_INPUT=0", "CHECKPOINT_DISABLE=1",
		"TF_PLUGIN_CACHE_DIR=" + pluginCache,
		"AWS_ACCESS_KEY_ID=test", "AWS_SECRET_ACCESS_KEY=test", "AWS_REGION=" + lsRegion,
		"AWS_ENDPOINT_URL=" + endpoint, "AWS_EC2_METADATA_DISABLED=true",
	}
	for k, v := range vars {
		env = append(env, "TF_VAR_"+k+"="+v)
	}
	return &lsTerraform{t: t, bin: bin, dir: dir, env: env}
}

func (r *lsTerraform) run(args ...string) (string, error) {
	cmd := exec.Command(r.bin, append([]string{"-chdir=" + r.dir}, args...)...)
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *lsTerraform) must(args ...string) string {
	r.t.Helper()
	out, err := r.run(args...)
	if err != nil {
		r.t.Fatalf("terraform %s: %v\n%s", args[0], err, out)
	}
	return out
}

// managed lists the AWS resources in the folder's state.
func (r *lsTerraform) managed() []string {
	r.t.Helper()
	var out []string
	for _, a := range strings.Fields(r.must("state", "list")) {
		if strings.HasPrefix(a, "aws_") {
			out = append(out, a)
		}
	}
	sort.Strings(out)
	return out
}

// drift is a refresh-only plan's view: each resource the provider read as
// changed outside Terraform, with the action ("delete" when it is gone).
func (r *lsTerraform) drift() map[string]string {
	r.t.Helper()
	out := r.must("plan", "-refresh-only", "-json", "-input=false", "-lock=false")
	got := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for sc.Scan() {
		var line struct {
			Type   string `json:"type"`
			Change struct {
				Resource struct {
					Addr string `json:"addr"`
				} `json:"resource"`
				Action string `json:"action"`
			} `json:"change"`
		}
		if json.Unmarshal(sc.Bytes(), &line) == nil && line.Type == "resource_drift" {
			got[line.Change.Resource.Addr] = line.Change.Action
		}
	}
	return got
}

const lsRegion = "ca-central-1"

// lsStore records run and deployment states for the job runner, and signals
// each run's end.
type lsStore struct {
	mu          sync.Mutex
	runs        map[string]jobs.Status
	failures    map[string]string
	deployments map[string]jobs.DeploymentStatus
	done        chan string
}

func (s *lsStore) RunStarted(context.Context, string, time.Time) error { return nil }

func (s *lsStore) RunFinished(_ context.Context, runID string, status jobs.Status, failure string, _ time.Time, change *jobs.DeploymentChange) error {
	s.mu.Lock()
	s.runs[runID], s.failures[runID] = status, failure
	if change != nil {
		s.deployments[change.ID] = change.Status
	}
	s.mu.Unlock()
	s.done <- runID
	return nil
}

func (s *lsStore) SetDeploymentStatus(_ context.Context, id string, status jobs.DeploymentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deployments[id] = status
	return nil
}

// waitDeployment waits for the deployment's status, which the job runner
// records with the run's end.
func (s *lsStore) waitDeployment(id string, want jobs.DeploymentStatus) jobs.DeploymentStatus {
	var got jobs.DeploymentStatus
	for i := 0; i < 50; i++ {
		s.mu.Lock()
		got = s.deployments[id]
		s.mu.Unlock()
		if got == want {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return got
}

type lsBroker struct{ secrets connectors.Secrets }

func (b lsBroker) Credentials(context.Context, string) (connectors.Secrets, error) {
	return b.secrets, nil
}

// lsCloud is LocalStack as a user's AWS account: Clutter's role stack,
// deployed from the repo's template, the AWS connector as A2 runs it, and B's
// job runner over it.
type lsCloud struct {
	c                            *Connector
	tf                           *terraform.Runner
	tfBin, endpoint, pluginCache string
	boundary, userBoundary       string
	secrets                      connectors.Secrets
	store                        *lsStore
	logs                         *logstore.Memory
	runner                       *jobs.Runner
}

// newLSCloud deploys the role stack into LocalStack and starts a job runner
// with the AWS connector. It skips the test unless CLUTTER_LOCALSTACK=1.
func newLSCloud(t *testing.T) *lsCloud {
	t.Helper()
	endpoint := localstackEndpoint(t)
	tfBin, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("terraform is not on the PATH")
	}
	pluginCache := os.Getenv("TF_PLUGIN_CACHE_DIR")
	if pluginCache == "" {
		pluginCache = t.TempDir()
	}

	// Clutter's role stack, as a user's account would hold it.
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	unique, externalID := hex.EncodeToString(suffix), uuid.NewString()
	stackDir := t.TempDir()
	stackTF := fmt.Sprintf(`terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region                      = %q
  skip_credentials_validation = true
  skip_metadata_api_check     = true

  endpoints {
    cloudformation = %q
    iam            = %q
    sts            = %q
  }
}

resource "aws_cloudformation_stack" "role" {
  name          = "clutter-role-%s"
  template_body = file(%q)
  capabilities  = ["CAPABILITY_NAMED_IAM"]
  parameters = {
    AccountId  = "000000000000"
    ExternalId = %q
    UniqueId   = %q
  }
}

output "role_arn" {
  value = aws_cloudformation_stack.role.outputs["RoleArn"]
}

output "boundary_arn" {
  value = aws_cloudformation_stack.role.outputs["PermissionsBoundaryArn"]
}

output "user_boundary_arn" {
  value = aws_cloudformation_stack.role.outputs["UserPermissionsBoundaryArn"]
}

output "version" {
  value = aws_cloudformation_stack.role.outputs["RoleTemplateVersion"]
}
`, lsRegion, endpoint, endpoint, endpoint, unique,
		filepath.Join(backendDir(), "infra", "modules", "s3", "uploads", "client_side_terraform_deployer_role.yaml"),
		externalID, unique)
	if err := os.WriteFile(filepath.Join(stackDir, "main.tf"), []byte(stackTF), 0o600); err != nil {
		t.Fatal(err)
	}
	stack := newLSTerraform(t, tfBin, stackDir, endpoint, pluginCache, nil)
	stack.must("init", "-input=false", "-no-color")
	t.Cleanup(func() {
		if out, err := stack.run("destroy", "-auto-approve", "-input=false", "-no-color"); err != nil {
			t.Errorf("deleting the role stack: %v\n%s", err, out)
		}
	})
	stack.must("apply", "-auto-approve", "-input=false", "-no-color")
	roleARN := strings.TrimSpace(stack.must("output", "-raw", "role_arn"))
	boundary := strings.TrimSpace(stack.must("output", "-raw", "boundary_arn"))
	if v := strings.TrimSpace(stack.must("output", "-raw", "version")); v != RoleTemplateVersion {
		t.Errorf("the stack says role template version %q, the connector expects %q", v, RoleTemplateVersion)
	}
	want, err := boundaryARN(roleARN)
	switch {
	case err != nil:
		t.Fatalf("the connector does not take the stack's role %s: %v", roleARN, err)
	case want == boundary:
	case strings.Replace(want, ":policy/clutter/", ":policy/", 1) == boundary:
		// LocalStack 4.14.0's CloudFormation drops a managed policy's Path.
		// AWS keeps it, and LocalStack accepts a boundary that does not
		// exist, so the runs below use the ARN the connector derives.
		t.Logf("LocalStack made the boundary at %s, without the stack's /clutter/ path", boundary)
		boundary = want
	default:
		t.Fatalf("the connector derives boundary %q from %s; the stack made %q", want, roleARN, boundary)
	}
	userBoundary := strings.TrimSpace(stack.must("output", "-raw", "user_boundary_arn"))
	wantUser, err := userBoundaryARN(roleARN)
	if err != nil || (wantUser != userBoundary && strings.Replace(wantUser, ":policy/clutter/", ":policy/", 1) != userBoundary) {
		t.Fatalf("the connector derives the users' boundary %q (%v) from %s; the stack made %q", wantUser, err, roleARN, userBoundary)
	}
	userBoundary = wantUser
	t.Logf("role stack: %s, boundary %s, users' boundary %s", roleARN, boundary, userBoundary)

	// The connector, as A2 runs it against LocalStack.
	assumer, err := NewLocalStackAssumer(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	tf := &terraform.Runner{
		Bin:            tfBin,
		WorkDir:        filepath.Join(root, "work"),
		StateDir:       filepath.Join(root, "state"),
		PluginCacheDir: pluginCache,
		Launcher:       runtask.Local{},
	}
	c, err := New(tf, diskTemplates(t), assumer).WithEndpoint(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	reg := connectors.NewRegistry()
	if err := reg.Register(c); err != nil {
		t.Fatal(err)
	}
	secrets := connectors.Secrets{CredRoleARN: roleARN, CredExternalID: externalID}

	store := &lsStore{runs: map[string]jobs.Status{}, failures: map[string]string{}, deployments: map[string]jobs.DeploymentStatus{}, done: make(chan string, 4)}
	logs := logstore.NewMemory()
	runner := jobs.New(jobs.Config{
		Registry: reg, Store: store, Broker: lsBroker{secrets: secrets}, Logs: logs,
		Workers: 1, RunTimeout: 20 * time.Minute,
	})
	runner.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = runner.Stop(ctx)
	})
	return &lsCloud{c: c, tf: tf, tfBin: tfBin, endpoint: endpoint, pluginCache: pluginCache, boundary: boundary, userBoundary: userBoundary,
		secrets: secrets, store: store, logs: logs, runner: runner}
}

// run runs one action of g for d through the job runner, fails the test
// unless the run completes, and returns its log.
func (l *lsCloud) run(t *testing.T, action connectors.Action, g connectors.Graph, d connectors.Deployment) string {
	t.Helper()
	id := uuid.NewString()
	key := logstore.KeyForRun(id)
	if err := l.runner.Enqueue(jobs.Job{RunID: id, Action: action, Graph: g, Deployment: d, SecretRef: "localstack", LogKey: key}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-l.store.done:
		if got != id {
			t.Fatalf("run %s ended, want %s", got, id)
		}
	case <-time.After(25 * time.Minute):
		t.Fatalf("%s did not end", action)
	}
	r, err := l.logs.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	log := string(body)
	for _, v := range l.secrets.Values() {
		if strings.Contains(log, v) {
			t.Errorf("the %s log holds a credential value", action)
		}
	}
	l.store.mu.Lock()
	status, failure := l.store.runs[id], l.store.failures[id]
	l.store.mu.Unlock()
	if status != jobs.StatusCompleted {
		t.Fatalf("%s %s (%s); log:\n%s", action, status, failure, log)
	}
	return log
}

// destroyAtEnd destroys d when the test ends, unless done reports it was
// destroyed already.
func (l *lsCloud) destroyAtEnd(t *testing.T, g connectors.Graph, d connectors.Deployment, done func() bool) {
	t.Cleanup(func() {
		if done() {
			return
		}
		id := uuid.NewString()
		_ = l.runner.Enqueue(jobs.Job{RunID: id, Action: connectors.ActionDestroy, Graph: g, Deployment: d, SecretRef: "localstack", LogKey: logstore.KeyForRun(id)})
		select {
		case <-l.store.done:
		case <-time.After(25 * time.Minute):
		}
	})
}

// TestEveryBlockAppliesOnLocalStack runs plan, apply and destroy for real,
// against LocalStack (backend/devserver/localstack.sh), through B's job runner,
// Terraform runner and a real terraform: a diagram with every AWS block and
// edge, as the role Clutter's CloudFormation stack creates. The stack itself
// is deployed into LocalStack first, from the repo's template. It checks that
// every block's main resource is reported built, that every AWS resource
// exists after apply, and that every one is gone after destroy. It runs only
// with CLUTTER_LOCALSTACK=1. LocalStack 4.14.0 enforces no IAM and ignores the
// external id, so this proves the call path, the rendered Terraform, ordering
// and that each call is accepted, not that the customer role allows it.
func TestEveryBlockAppliesOnLocalStack(t *testing.T) {
	l := newLSCloud(t)
	c, tf, tfBin, endpoint, pluginCache := l.c, l.tf, l.tfBin, l.endpoint, l.pluginCache
	boundary, userBoundary, secrets, store := l.boundary, l.userBoundary, l.secrets, l.store

	d := testDeployment(lsRegion)
	d.Connection.Credentials = nil

	// LocalStack does not check the external id; record what it does.
	wrong := d.Connection
	wrong.Credentials = connectors.Secrets{CredRoleARN: secrets[CredRoleARN], CredExternalID: "not-the-external-id"}
	if err := c.VerifyConnection(context.Background(), wrong); err == nil {
		t.Log("LocalStack let the role in with a wrong external id: it does not enforce sts:ExternalId")
	} else {
		t.Logf("LocalStack refused a wrong external id: %v", err)
	}

	g := fullGraph()
	run := func(action connectors.Action) string {
		t.Helper()
		return l.run(t, action, g, d)
	}

	// The check folder reads the deployment's state with the same files a
	// run applies, so a refresh-only plan asks LocalStack about each one.
	files, err := c.Preview(context.Background(), g, d)
	if err != nil {
		t.Fatal(err)
	}
	checkDir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(checkDir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	check := newLSTerraform(t, tfBin, checkDir, endpoint, pluginCache, map[string]string{"permissions_boundary_arn": boundary, "user_permissions_boundary_arn": userBoundary})
	statePath := filepath.Join(tf.StateDir, d.ID, "terraform.tfstate")
	useState := func(state []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(checkDir, "terraform.tfstate"), state, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	planLog := run(connectors.ActionPlan)
	if m := regexp.MustCompile(`Plan: (\d+) to add`).FindStringSubmatch(planLog); m == nil || m[1] == "0" {
		t.Fatalf("the plan adds nothing:\n%s", planLog)
	}

	destroyed := false
	l.destroyAtEnd(t, g, d, func() bool { return destroyed })
	applyLog := run(connectors.ActionApply)
	if want := fmt.Sprintf("Apply complete: %d blocks built or changed.", len(g.Blocks)); !strings.Contains(applyLog, want) {
		t.Errorf("not every block's main resource was reported built (want %q):\n%s", want, applyLog)
	}
	if s := store.waitDeployment(d.ID, jobs.DeploymentLive); s != jobs.DeploymentLive {
		t.Errorf("deployment %s after apply, want live", s)
	}

	state, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	useState(state)
	check.must("init", "-input=false", "-no-color")
	resources := check.managed()
	t.Logf("applied %d resources: %s", len(resources), strings.Join(resources, " "))
	for addr, action := range check.drift() {
		if action == "delete" {
			t.Errorf("%s does not exist after apply", addr)
		}
	}
	if out, err := check.run("plan", "-detailed-exitcode", "-input=false", "-no-color", "-lock=false"); err != nil {
		var keep []string
		for _, l := range strings.Split(out, "\n") {
			s := strings.TrimSpace(l)
			if strings.Contains(l, " will be ") || strings.Contains(l, " must be ") || strings.Contains(l, "forces replacement") ||
				strings.HasPrefix(s, "~ ") || strings.HasPrefix(s, "Plan:") || strings.HasPrefix(s, "Error") {
				keep = append(keep, l)
			}
		}
		t.Logf("a plan after apply is not empty (LocalStack does not store every attribute):\n%s", strings.Join(keep, "\n"))
	}

	run(connectors.ActionDestroy)
	destroyed = true
	if s := store.waitDeployment(d.ID, jobs.DeploymentDestroyed); s != jobs.DeploymentDestroyed {
		t.Errorf("deployment %s after destroy, want destroyed", s)
	}
	// Read the state from before the destroy again: every resource in it must
	// now be gone from LocalStack.
	drift := check.drift()
	var still []string
	for _, addr := range resources {
		if drift[addr] != "delete" {
			still = append(still, addr)
		}
	}
	if len(still) > 0 {
		t.Errorf("still in LocalStack after destroy: %s", strings.Join(still, " "))
	}

	// A split deployment's AWS part, as the second cloud's VM needs it: a
	// scoped user with its policy and key for the bucket and the table, and
	// the public VM's address.
	x := exportDeployment()
	x.Connection.Region = lsRegion
	x.Connection.Credentials = secrets
	var xlog bytes.Buffer
	ctx := context.Background()
	_, out, err := c.ApplyCross(ctx, connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &xlog}, exportGraph(), x)
	xDestroy := func() {
		if _, err := c.Destroy(ctx, connectors.Run{ID: uuid.NewString(), Action: connectors.ActionDestroy, Log: &xlog}, x); err != nil {
			t.Errorf("destroying the split part: %v\n%s", err, xlog.String())
		}
	}
	if err != nil {
		xDestroy()
		t.Fatalf("applying the split part: %v\n%s", err, xlog.String())
	}
	kb := x.Cross.Exports[0].Key
	id, secret := out[kb][connectors.ValueAccessKeyID], out[kb][connectors.ValueSecretAccessKey]
	if id.Text == "" || secret.Text == "" || !secret.Sensitive || out[kb][connectors.ValueBucket].Text == "" {
		t.Errorf("the scoped key's outputs: id %q, secret set %t, sensitive %t", id.Text, secret.Text != "", secret.Sensitive)
	}
	if secret.Text != "" && strings.Contains(xlog.String(), secret.Text) {
		t.Error("the scoped key's secret reached the run's log")
	}
	xstate, err := os.ReadFile(filepath.Join(tf.StateDir, x.ID, "terraform.tfstate"))
	if err != nil {
		t.Fatal(err)
	}
	// The user carries the users' boundary, and the provider's default tags
	// put ManagedBy on what the run created.
	for _, want := range []string{`"type": "aws_iam_user"`, `"type": "aws_iam_user_policy"`, `"type": "aws_iam_access_key"`, `"path": "/clutter/"`,
		`"permissions_boundary": "` + userBoundary + `"`, `"ManagedBy": "Clutter"`} {
		if !strings.Contains(string(xstate), want) {
			t.Errorf("the split part's state lacks %s", want)
		}
	}
	xDestroy()
	xstate, err = os.ReadFile(filepath.Join(tf.StateDir, x.ID, "terraform.tfstate"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(xstate), `"type": "aws_iam_access_key"`) || strings.Contains(string(xstate), `"type": "aws_iam_user"`) {
		t.Error("the scoped user or key is still in state after destroy")
	}
}
