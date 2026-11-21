package runtask

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask/runtasktest"
	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

const (
	tRun     = "11111111-2222-4333-8444-555555555555"
	tDep     = "99999999-8888-4777-8666-555555555555"
	tKeyID   = "ASIAEXAMPLEEXAMPLE01"
	tSecret  = "wJalrXUtnFEMIexampleSECRETkeyEXAMPLE0001"
	tToken   = "FwoGZXIvYXdzEXAMPLEtokenEXAMPLEtokenEXAMPLE=="
	tCustKey = "customer-secret-value-0001"
)

func running() ecstypes.Task { return runtasktest.Running() }

func stopped(code *int32, reason string) func() ecstypes.Task {
	return func() ecstypes.Task { return runtasktest.StoppedTask("run", code, reason) }
}

func testSettings() ECSSettings {
	return ECSSettings{
		Region: "ca-central-1", Cluster: "clutter-adapter-runs", TaskDefinition: "clutter-adapter-run", Container: "run",
		Subnets: []string{"subnet-0123456789abcdef0"}, SecurityGroups: []string{"sg-0123456789abcdef0"},
		AssignPublicIP: true, RunBucket: "clutter-adapter-runs-x", StateBucket: "clutter-adapter-state-x",
		RunFilesRoleARN: "arn:aws:iam::111122223333:role/ClutterAdapterRunFilesRole",
		PollInterval:    time.Millisecond, StopGrace: time.Second,
	}
}

func newTestECS(t *testing.T, script ...func() ecstypes.Task) (*ECS, *runtasktest.ECS, *runtasktest.S3, *runtasktest.STS, Run) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{"main.tf": "# main\n", "zz_clutter_backend.tf": "# backend\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fe, m := &runtasktest.ECS{Script: script}, runtasktest.New()
	fs := &runtasktest.STS{AccessKeyID: tKeyID, SecretAccessKey: tSecret, SessionToken: tToken}
	e := &ECS{Settings: testSettings(), ECS: fe, S3: m, STS: fs}
	r := Run{
		RunID: tRun, DeploymentID: tDep, Dir: dir,
		Steps:  [][]string{{"init", "-input=false", "-no-color"}, {"apply", "-input=false", "-no-color", "-auto-approve", "-lock-timeout=60s"}},
		Env:    map[string]string{"OS_APPLICATION_CREDENTIAL_SECRET": tCustKey, "PATH": "/usr/bin:/bin"},
		Output: &bytes.Buffer{},
	}
	return e, fe, m, fs, r
}

func TestLaunchRunSucceeds(t *testing.T) {
	var e *ECS
	var m *runtasktest.S3
	keys := KeysFor(tRun)
	// The task writes its log in two parts while it runs.
	e, fe, m, fs, r := newTestECS(t,
		func() ecstypes.Task { return running() },
		func() ecstypes.Task {
			m.Set("clutter-adapter-runs-x", keys.Log, []byte("init done\n"))
			return running()
		},
		func() ecstypes.Task {
			m.Set("clutter-adapter-runs-x", keys.Log, []byte("init done\napply done\n"))
			return stopped(aws.Int32(0), "Essential container in task exited")()
		},
	)
	if err := e.LaunchRun(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if got := r.Output.(*bytes.Buffer).String(); got != "init done\napply done\n" {
		t.Fatalf("output %q", got)
	}
	// Every object is deleted at the end, so the deletions list the uploads.
	seen := m.Deleted
	for _, want := range []string{keys.FilesPrefix + "main.tf", keys.FilesPrefix + "zz_clutter_backend.tf", keys.Spec, keys.TaskEnv, keys.Log} {
		if !contains(seen, "clutter-adapter-runs-x/"+want) {
			t.Errorf("%s was not uploaded and cleaned up (deleted: %v)", want, seen)
		}
	}
	if left := m.Keys(); len(left) != 0 {
		t.Errorf("objects left after the run: %v", left)
	}

	in := fe.Run
	if aws.ToString(in.Cluster) != "clutter-adapter-runs" || aws.ToString(in.TaskDefinition) != "clutter-adapter-run" ||
		in.LaunchType != ecstypes.LaunchTypeFargate || aws.ToString(in.ClientToken) != tRun {
		t.Errorf("RunTask input %+v", in)
	}
	if in.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp != ecstypes.AssignPublicIpEnabled {
		t.Error("public IP not assigned")
	}
	co := in.Overrides.ContainerOverrides[0]
	if aws.ToString(co.Name) != "run" || len(co.EnvironmentFiles) != 1 ||
		aws.ToString(co.EnvironmentFiles[0].Value) != "arn:aws:s3:::clutter-adapter-runs-x/"+keys.TaskEnv {
		t.Errorf("container override %+v", co)
	}
	// Overrides are shown by DescribeTasks: no key or credential may be in them.
	ov, _ := json.Marshal(in.Overrides)
	for _, secret := range []string{tKeyID, tSecret, tToken, tCustKey} {
		if strings.Contains(string(ov), secret) {
			t.Errorf("overrides hold a secret: %s", ov)
		}
	}
	if aws.ToString(fs.In.RoleSessionName) != "b-run-"+tRun || aws.ToInt32(fs.In.DurationSeconds) != 3600 {
		t.Errorf("AssumeRole input %+v", fs.In)
	}
}

func TestLaunchRunUploadsSpecAndKeys(t *testing.T) {
	e, _, m, _, r := newTestECS(t, stopped(aws.Int32(0), ""))
	// The run's objects are deleted at the end, so read them as they are put.
	var spec, env []byte
	m.OnPut = func(_, key string, body []byte) {
		switch key {
		case KeysFor(tRun).Spec:
			spec = body
		case KeysFor(tRun).TaskEnv:
			env = body
		}
	}
	if err := e.LaunchRun(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	var s RunSpec
	if err := json.Unmarshal(spec, &s); err != nil || s.Env["OS_APPLICATION_CREDENTIAL_SECRET"] != tCustKey || len(s.Steps) != 2 {
		t.Fatalf("spec %s (%v)", spec, err)
	}
	want := EnvAccessKeyID + "=" + tKeyID + "\n" + EnvSecretAccessKey + "=" + tSecret + "\n" + EnvSessionToken + "=" + tToken + "\n"
	if string(env) != want {
		t.Fatalf("task.env %q", env)
	}
}

func TestRunPolicyIsScopedToOneRunAndOneDeployment(t *testing.T) {
	p, err := runPolicy("runs-b", "state-b", tRun, tDep)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"arn:aws:s3:::runs-b/runs/` + tRun + `/in/*"`,
		`"arn:aws:s3:::runs-b/runs/` + tRun + `/out/*"`,
		`"arn:aws:s3:::state-b/state/` + tDep + `/terraform.tfstate"`,
		`"arn:aws:s3:::state-b/state/` + tDep + `/terraform.tfstate.tflock"`,
		`"s3:prefix":"state/` + tDep + `/*"`,
	} {
		if !strings.Contains(p, want) {
			t.Errorf("policy lacks %s:\n%s", want, p)
		}
	}
	if strings.Contains(p, "runs-b/*") || strings.Contains(p, "state-b/*") || strings.Contains(p, "runs/*") {
		t.Errorf("policy reaches beyond the run:\n%s", p)
	}
	if len(p) > 2048 {
		t.Errorf("policy is %d characters; STS allows 2048", len(p))
	}
}

func TestLaunchRunReportsTheFailedStep(t *testing.T) {
	e, _, m, _, r := newTestECS(t, func() ecstypes.Task {
		return stopped(aws.Int32(1), "Essential container in task exited")()
	})
	m.Set("clutter-adapter-runs-x", KeysFor(tRun).Result, []byte(`{"step":"apply","exit_code":1}`))
	err := e.LaunchRun(context.Background(), r)
	var se *StepError
	if !errors.As(err, &se) || se.Step != "apply" || se.ExitCode != 1 || err.Error() != "terraform apply: exit status 1" {
		t.Fatalf("err %v", err)
	}
	if errors.Is(err, ErrNotStarted) {
		t.Fatal("a failed apply is reported as not started")
	}
}

func TestLaunchRunReportsAFailureWithoutAResultAsReachingTheCloud(t *testing.T) {
	e, _, _, _, r := newTestECS(t, stopped(aws.Int32(137), "OutOfMemoryError"))
	err := e.LaunchRun(context.Background(), r)
	var se *StepError
	if !errors.As(err, &se) || se.Step != "run" || errors.Is(err, ErrNotStarted) {
		t.Fatalf("err %v", err)
	}
}

func TestLaunchRunReportsARunThatCouldNotBePrepared(t *testing.T) {
	e, _, m, _, r := newTestECS(t, stopped(aws.Int32(70), "Essential container in task exited"))
	m.Set("clutter-adapter-runs-x", KeysFor(tRun).Result, []byte(`{"step":"prepare","exit_code":70}`))
	if err := e.LaunchRun(context.Background(), r); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("err %v", err)
	}
}

func TestLaunchRunReportsATaskThatNeverRan(t *testing.T) {
	e, _, _, _, r := newTestECS(t, stopped(nil, "CannotPullContainerError: image not found"))
	err := e.LaunchRun(context.Background(), r)
	if !errors.Is(err, ErrNotStarted) || !strings.Contains(err.Error(), "CannotPullContainerError") {
		t.Fatalf("err %v", err)
	}
}

func TestLaunchRunStopsTheTaskWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := 0
	var fe *runtasktest.ECS
	e, fe, _, _, r := newTestECS(t, func() ecstypes.Task {
		n++
		if n == 2 {
			cancel()
		}
		if fe.Stopped {
			return stopped(aws.Int32(130), "Clutter stopped the run")()
		}
		return running()
	})
	err := e.LaunchRun(ctx, r)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err %v", err)
	}
	if !fe.Stopped {
		t.Fatal("the task was not stopped")
	}
}

func TestLaunchRunRefusesWhatECSCannotPlace(t *testing.T) {
	e, fe, m, _, r := newTestECS(t, running)
	fe.Failures = []ecstypes.Failure{{Reason: aws.String("RESOURCE:ENI")}}
	if err := e.LaunchRun(context.Background(), r); !errors.Is(err, ErrNotStarted) || !strings.Contains(err.Error(), "RESOURCE:ENI") {
		t.Fatalf("err %v", err)
	}
	if left := m.Keys(); len(left) != 0 {
		t.Errorf("objects left after a failed start: %v", left)
	}
}

func TestLaunchRunRefusesBadInput(t *testing.T) {
	cases := map[string]func(r *Run){
		"run id":     func(r *Run) { r.RunID = "../x" },
		"deployment": func(r *Run) { r.DeploymentID = "x" },
		"step":       func(r *Run) { r.Steps = [][]string{{"console"}} },
		"flag":       func(r *Run) { r.Steps = [][]string{{"init", "-backend-config=/tmp/x"}} },
		"env name":   func(r *Run) { r.Env = map[string]string{"A=B": "c"} },
		"subfolder": func(r *Run) {
			if err := os.Mkdir(filepath.Join(r.Dir, ".terraform"), 0o700); err != nil {
				panic(err)
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e, fe, _, _, r := newTestECS(t, running)
			mutate(&r)
			// Nothing started, so the Terraform runner must be able to say
			// that nothing was built.
			if err := e.LaunchRun(context.Background(), r); !errors.Is(err, ErrNotStarted) {
				t.Fatalf("not refused as not started: %v", err)
			}
			if fe.Run != nil {
				t.Fatal("a task was started")
			}
		})
	}
}

func TestLaunchRefusesASingleCommand(t *testing.T) {
	e, _, _, _, _ := newTestECS(t, running)
	if err := e.Launch(context.Background(), Task{}); err == nil {
		t.Fatal("not refused")
	}
}

func TestStateBackend(t *testing.T) {
	e, _, _, _, _ := newTestECS(t, running)
	b, err := e.StateBackend(tDep)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`backend "s3"`,
		`bucket               = "clutter-adapter-state-x"`,
		`key                  = "state/` + tDep + `/terraform.tfstate"`,
		`workspace_key_prefix = "state/` + tDep + `/workspaces"`,
		`region               = "ca-central-1"`,
		`use_lockfile         = true`,
		`encrypt              = true`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("backend lacks %q:\n%s", want, b)
		}
	}
	if strings.Contains(string(b), "access_key") || strings.Contains(string(b), "dynamodb") {
		t.Errorf("backend holds keys or a lock table:\n%s", b)
	}
	if _, err := e.StateBackend("../other"); err == nil {
		t.Fatal("a bad deployment id was accepted")
	}
}

func TestLoadECSSettings(t *testing.T) {
	env := map[string]string{
		"AWS_REGION":                   "ca-central-1",
		"ADAPTER_RUN_CLUSTER":          "clutter-adapter-runs",
		"ADAPTER_RUN_TASK_DEFINITION":  "clutter-adapter-run",
		"ADAPTER_RUN_CONTAINER":        "run",
		"ADAPTER_RUN_SUBNETS":          "subnet-0123456789abcdef0, subnet-0123456789abcdef1",
		"ADAPTER_RUN_SECURITY_GROUPS":  "sg-0123456789abcdef0",
		"ADAPTER_RUN_BUCKET":           "clutter-adapter-runs-x",
		"ADAPTER_STATE_BUCKET":         "clutter-adapter-state-x",
		"ADAPTER_RUN_FILES_ROLE_ARN":   "arn:aws:iam::111122223333:role/ClutterAdapterRunFilesRole",
		"ADAPTER_RUN_ASSIGN_PUBLIC_IP": "true",
	}
	s, err := LoadECSSettings(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Subnets) != 2 || !s.AssignPublicIP || s.StateBucket != "clutter-adapter-state-x" {
		t.Fatalf("settings %+v", s)
	}

	env["ADAPTER_RUN_BUCKET"] = "Not A Bucket"
	delete(env, "ADAPTER_RUN_CLUSTER")
	env["ADAPTER_RUN_ASSIGN_PUBLIC_IP"] = "yes"
	_, err = LoadECSSettings(func(k string) string { return env[k] })
	if err == nil {
		t.Fatal("bad settings were accepted")
	}
	for _, want := range []string{"ADAPTER_RUN_BUCKET is not valid", "ADAPTER_RUN_CLUSTER is not set", "ADAPTER_RUN_ASSIGN_PUBLIC_IP"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "Not A Bucket") {
		t.Error("the error holds a value")
	}
}

func TestCheckSteps(t *testing.T) {
	ok := [][]string{{"init", "-input=false", "-no-color"}, {"destroy", "-auto-approve", "-lock-timeout=60s"}}
	if err := CheckSteps(ok); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][][]string{nil, {{}}, {{"console"}}, {{"plan", "-var-file=x"}}, {{"plan", "-chdir=/"}}, {{"plan", "-lock-timeout=1h; rm"}}} {
		if CheckSteps(bad) == nil {
			t.Errorf("%v accepted", bad)
		}
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
