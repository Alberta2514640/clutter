package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask/runtasktest"
)

const (
	testRun    = "11111111-2222-4333-8444-555555555555"
	testBucket = "clutter-adapter-runs-test"
	testKeyID  = "ASIAEXAMPLEEXAMPLE01"
	testSecret = "wJalrXUtnFEMIexampleSECRETkeyEXAMPLE0001"
	testToken  = "FwoGZXIvYXdzEXAMPLEtokenEXAMPLEtokenEXAMPLE=="
)

// fakeTerraform records each call and its environment, and fails the step
// named by FAIL_STEP with exit code 3.
const fakeTerraform = `#!/bin/sh
echo "terraform says $1"
{
  echo "args: $*"
  echo "home: $HOME"
  echo "cli: $TF_CLI_CONFIG_FILE"
  echo "other: $OTHER_PATH"
  echo "cache: $TF_PLUGIN_CACHE_DIR"
  echo "customer: $CUSTOMER_TOKEN"
  echo "leak: ${CLUTTER_RUN_SECRET_ACCESS_KEY:-none}"
  echo "pwd: $(pwd)"
} >> "$CALLS"
[ "$1" = "$FAIL_STEP" ] && exit 3
exit 0
`

func setup(t *testing.T, spec runtask.RunSpec) (*worker, *runtasktest.S3, string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tf := filepath.Join(dir, "terraform")
	if err := os.WriteFile(tf, []byte(fakeTerraform), 0o700); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(dir, "calls.txt")
	if spec.Env == nil {
		spec.Env = map[string]string{}
	}
	spec.Env["CALLS"] = calls
	// The task's own variables are in this process's environment; Terraform
	// must never see them.
	t.Setenv(runtask.EnvSecretAccessKey, testSecret)

	m := runtasktest.New()
	keys := runtask.KeysFor(testRun)
	body, _ := json.Marshal(spec)
	m.Set(testBucket, keys.Spec, body)
	m.Set(testBucket, keys.TaskEnv, []byte("CLUTTER_RUN_ACCESS_KEY_ID="+testKeyID+"\n"))
	m.Set(testBucket, keys.FilesPrefix+"main.tf", []byte("# main\n"))
	m.Set(testBucket, keys.FilesPrefix+"zz_clutter_backend.tf", []byte("# backend\n"))
	w := &worker{
		S3: m, Bucket: testBucket, RunID: testRun,
		AccessKeyID: testKeyID, SecretAccessKey: testSecret, SessionToken: testToken,
		Terraform: tf, WorkDir: filepath.Join(dir, "work"),
		LogEvery: 10 * time.Millisecond, Grace: time.Second, Status: &bytes.Buffer{},
	}
	return w, m, calls
}

var allSteps = [][]string{
	{"init", "-input=false", "-no-color"},
	{"plan", "-input=false", "-no-color", "-lock-timeout=60s"},
}

func result(t *testing.T, m *runtasktest.S3) runtask.Result {
	t.Helper()
	body, ok := m.Object(testBucket, runtask.KeysFor(testRun).Result)
	if !ok {
		t.Fatal("no result uploaded")
	}
	var r runtask.Result
	if err := json.Unmarshal(body, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRunSucceeds(t *testing.T) {
	bDir := "/b/data/work/" + testRun
	w, m, calls := setup(t, runtask.RunSpec{Steps: allSteps, Dir: bDir, Env: map[string]string{
		"CUSTOMER_TOKEN":     "tok",
		"HOME":               bDir,
		"TF_CLI_CONFIG_FILE": bDir + "/clutter.tfrc",
		"OTHER_PATH":         bDir + "-not-inside/x",
	}})
	if code := w.Run(context.Background()); code != 0 {
		t.Fatalf("exit %d, status %s", code, w.Status)
	}
	keys := runtask.KeysFor(testRun)
	if r := result(t, m); r.Step != "" || r.ExitCode != 0 {
		t.Fatalf("result %+v", r)
	}
	log, _ := m.Object(testBucket, keys.Log)
	if !strings.Contains(string(log), "terraform says init") || !strings.Contains(string(log), "terraform says plan") {
		t.Fatalf("log %q", log)
	}
	for _, k := range []string{keys.Spec, keys.TaskEnv} {
		if _, ok := m.Object(testBucket, k); ok {
			t.Errorf("%s was not deleted", k)
		}
	}
	got, _ := os.ReadFile(calls)
	s := string(got)
	backendConfig := filepath.Join(w.WorkDir, "state.s3.tfbackend")
	for _, want := range []string{
		"args: init -input=false -no-color -backend-config=" + backendConfig,
		"args: plan -input=false -no-color -lock-timeout=60s\n",
		"home: " + filepath.Join(w.WorkDir, "run") + "\n",
		"cli: " + filepath.Join(w.WorkDir, "run", "clutter.tfrc"),
		"other: " + bDir + "-not-inside/x",
		"cache: " + filepath.Join(w.WorkDir, "plugin-cache"),
		"customer: tok",
		"leak: none",
		"pwd: " + filepath.Join(w.WorkDir, "run"),
	} {
		if !strings.Contains(s, want) {
			t.Errorf("calls lack %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, testSecret) {
		t.Error("terraform saw the task's own key")
	}
	cfg, _ := os.ReadFile(backendConfig)
	if !strings.Contains(string(cfg), `access_key = "`+testKeyID+`"`) || !strings.Contains(string(cfg), `token      = "`+testToken+`"`) {
		t.Errorf("backend config %q", cfg)
	}
	if strings.Contains(string(log), testSecret) || strings.Contains(w.Status.(*bytes.Buffer).String(), testSecret) {
		t.Error("a key reached the log or the status output")
	}
	for _, name := range []string{"main.tf", "zz_clutter_backend.tf"} {
		if _, err := os.Stat(filepath.Join(w.WorkDir, "run", name)); err != nil {
			t.Errorf("%s not downloaded: %v", name, err)
		}
	}
}

func TestRunStopsAtTheFailedStep(t *testing.T) {
	steps := append(allSteps, []string{"apply", "-input=false", "-auto-approve"})
	w, m, calls := setup(t, runtask.RunSpec{Steps: steps, Env: map[string]string{"FAIL_STEP": "plan"}})
	if code := w.Run(context.Background()); code != 3 {
		t.Fatalf("exit %d, want 3", code)
	}
	if r := result(t, m); r.Step != "plan" || r.ExitCode != 3 {
		t.Fatalf("result %+v", r)
	}
	got, _ := os.ReadFile(calls)
	if strings.Contains(string(got), "args: apply") {
		t.Fatal("apply ran after plan failed")
	}
}

func TestRunRefusesAStepNotAllowed(t *testing.T) {
	w, m, calls := setup(t, runtask.RunSpec{Steps: [][]string{{"init", "-chdir=/etc"}}})
	if code := w.Run(context.Background()); code != exitSetup {
		t.Fatalf("exit %d", code)
	}
	if r := result(t, m); r.Step != "prepare" {
		t.Fatalf("result %+v", r)
	}
	if _, err := os.Stat(calls); err == nil {
		t.Fatal("terraform ran")
	}
	log, _ := m.Object(testBucket, runtask.KeysFor(testRun).Log)
	if !strings.Contains(string(log), "not allowed") {
		t.Fatalf("log %q", log)
	}
}

func TestRunRefusesAFileOutsideTheFolder(t *testing.T) {
	w, m, calls := setup(t, runtask.RunSpec{Steps: allSteps})
	m.Set(testBucket, runtask.KeysFor(testRun).FilesPrefix+"../escape.tf", []byte("x"))
	if code := w.Run(context.Background()); code != exitSetup {
		t.Fatalf("exit %d", code)
	}
	if _, err := os.Stat(calls); err == nil {
		t.Fatal("terraform ran")
	}
	if _, err := os.Stat(filepath.Join(w.WorkDir, "escape.tf")); err == nil {
		t.Fatal("a file was written outside the run folder")
	}
}

type fakeResolver map[string]string

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ip, ok := f[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	return []net.IPAddr{{IP: net.ParseIP(ip)}}, nil
}

// TestRunChecksPublicHosts: a cloud address that resolves inside Clutter's
// network, or not at all, stops the run before Terraform starts.
func TestRunChecksPublicHosts(t *testing.T) {
	dns := fakeResolver{"cloud.example.com": "203.0.113.10", "inside.example.com": "10.0.0.5", "meta.example.com": "169.254.169.254"}
	for host, wantLog := range map[string]string{
		"inside.example.com":  "resolves to a private",
		"meta.example.com":    "resolves to a private",
		"10.1.2.3":            "resolves to a private",
		"missing.example.com": "could not be looked up",
	} {
		t.Run(host, func(t *testing.T) {
			w, m, calls := setup(t, runtask.RunSpec{Steps: allSteps, PublicHosts: []string{"cloud.example.com", host}})
			w.Resolver = dns
			if code := w.Run(context.Background()); code != exitSetup {
				t.Fatalf("exit %d", code)
			}
			if _, err := os.Stat(calls); err == nil {
				t.Fatal("terraform ran")
			}
			if r := result(t, m); r.Step != "prepare" {
				t.Fatalf("result %+v", r)
			}
			log, _ := m.Object(testBucket, runtask.KeysFor(testRun).Log)
			if !strings.Contains(string(log), wantLog) {
				t.Fatalf("log %q", log)
			}
		})
	}
	w, _, calls := setup(t, runtask.RunSpec{Steps: allSteps, PublicHosts: []string{"cloud.example.com", "203.0.113.11"}})
	w.Resolver = dns
	if code := w.Run(context.Background()); code != 0 {
		t.Fatalf("a public host was refused: exit %d, status %s", code, w.Status)
	}
	if _, err := os.Stat(calls); err != nil {
		t.Fatal("terraform did not run")
	}
}

func TestRunReportsAnUnreadableSpec(t *testing.T) {
	w, m, _ := setup(t, runtask.RunSpec{Steps: allSteps})
	m.Set(testBucket, runtask.KeysFor(testRun).Spec, []byte("{"))
	if code := w.Run(context.Background()); code != exitSetup {
		t.Fatalf("exit %d", code)
	}
	if r := result(t, m); r.Step != "prepare" || r.ExitCode != exitSetup {
		t.Fatalf("result %+v", r)
	}
}

func TestRunFailsWhenTheResultCannotBeUploaded(t *testing.T) {
	w, m, _ := setup(t, runtask.RunSpec{Steps: allSteps})
	m.FailPut = "result.json"
	if code := w.Run(context.Background()); code != exitSetup {
		t.Fatalf("exit %d, want %d", code, exitSetup)
	}
}

func TestMainRefusesMissingSettings(t *testing.T) {
	if code := run(context.Background(), func(string) string { return "" }); code != exitSetup {
		t.Fatalf("exit %d", code)
	}
}
