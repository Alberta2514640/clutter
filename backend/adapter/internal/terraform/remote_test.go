package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask/runtasktest"
	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/uuid"
)

// fakeRunLauncher records the run it is handed and returns err.
type fakeRunLauncher struct {
	backend []byte
	err     error
	run     *runtask.Run
	files   map[string]string
}

func (f *fakeRunLauncher) Launch(context.Context, runtask.Task) error {
	return errors.New("a remote launcher is never handed a single command")
}

func (f *fakeRunLauncher) StateBackend(string) ([]byte, error) {
	if f.backend == nil {
		return []byte("terraform {\n  backend \"s3\" {\n    bucket = \"b\"\n  }\n}\n"), nil
	}
	return f.backend, nil
}

func (f *fakeRunLauncher) LaunchRun(_ context.Context, r runtask.Run) error {
	f.run = &r
	f.files = map[string]string{}
	entries, err := os.ReadDir(r.Dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(r.Dir, e.Name()))
		if err != nil {
			return err
		}
		f.files[e.Name()] = string(b)
	}
	return f.err
}

func remoteJob() Job {
	return Job{
		DeploymentID: uuid.NewString(), RunID: uuid.NewString(), Action: connectors.ActionApply,
		Files: map[string][]byte{"main.tf": []byte("resource \"terraform_data\" \"n1\" {}\n")},
		Env:   map[string]string{"OS_APPLICATION_CREDENTIAL_SECRET": "run-secret"},
		Log:   &bytes.Buffer{},
	}
}

func TestRemoteRunHandsTheWholeRunToTheLauncher(t *testing.T) {
	l := &fakeRunLauncher{}
	// No terraform binary: B needs none when the launcher runs Terraform.
	r := newRunner(t, "", l)
	j := remoteJob()
	j.PublicHosts = []string{"cloud.example.com"}
	if err := r.Run(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	if l.run == nil || l.run.RunID != j.RunID || l.run.DeploymentID != j.DeploymentID || l.run.Output != j.Log {
		t.Fatalf("run %+v", l.run)
	}
	if fmt.Sprint(l.run.PublicHosts) != "[cloud.example.com]" {
		t.Fatalf("public hosts %v, want the job's", l.run.PublicHosts)
	}
	var names []string
	for _, s := range l.run.Steps {
		names = append(names, s[0])
	}
	if fmt.Sprint(names) != "[init apply]" {
		t.Fatalf("steps %v", l.run.Steps)
	}
	if err := runtask.CheckSteps(l.run.Steps); err != nil {
		t.Fatalf("the steps are not ones the run task allows: %v", err)
	}
	if !strings.Contains(l.files[backendFileName], `backend "s3"`) || strings.Contains(l.files[backendFileName], "local") {
		t.Fatalf("backend file %q", l.files[backendFileName])
	}
	if _, ok := l.files["main.tf"]; !ok {
		t.Fatalf("files %v", l.files)
	}
	if l.run.Env["OS_APPLICATION_CREDENTIAL_SECRET"] != "run-secret" ||
		l.run.Env["TF_CLI_CONFIG_FILE"] != filepath.Join(l.run.Dir, cliConfigName) {
		t.Fatalf("env %v", l.run.Env)
	}
	// No local state or plugin folder is made for a remote run.
	for _, d := range []string{r.StateDir, r.PluginCacheDir} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Fatalf("%s was created", d)
		}
	}
	if _, err := os.Stat(l.run.Dir); !os.IsNotExist(err) {
		t.Fatal("the run's work folder was not removed")
	}
}

func TestRemoteRunOnlyFailuresAfterInitMayHaveBuiltSomething(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		nothing bool
	}{
		{"not started", fmt.Errorf("%w: start run task: RESOURCE:ENI", runtask.ErrNotStarted), true},
		{"init fails", &runtask.StepError{Step: "init", ExitCode: 1}, true},
		{"apply fails", &runtask.StepError{Step: "apply", ExitCode: 1}, false},
		{"unexplained failure", &runtask.StepError{Step: "run", ExitCode: 137}, false},
		{"cancelled while running", context.Canceled, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRunner(t, "", &fakeRunLauncher{err: tc.err})
			err := r.Run(context.Background(), remoteJob())
			if !errors.Is(err, tc.err) {
				t.Fatalf("err %v does not wrap %v", err, tc.err)
			}
			if got := errors.Is(err, connectors.ErrNothingBuilt); got != tc.nothing {
				t.Fatalf("nothing built is %t, want %t: %v", got, tc.nothing, err)
			}
		})
	}
}

func TestRemoteRunRefusesABackendTheRunnerDidNotWrite(t *testing.T) {
	// A rendered file may not take the backend file's name.
	l := &fakeRunLauncher{}
	j := remoteJob()
	j.Files[backendFileName] = []byte("")
	if err := newRunner(t, "", l).Run(context.Background(), j); !errors.Is(err, ErrRefused) || l.run != nil {
		t.Fatalf("a rendered backend file name: %v", err)
	}
	// The launcher's backend must be exactly one backend block.
	for name, body := range map[string]string{
		"extra resource": "terraform {\n  backend \"s3\" {}\n}\nresource \"terraform_data\" \"x\" {}\n",
		"extra setting":  "terraform {\n  backend \"s3\" {}\n  required_version = \">= 1\"\n}\n",
		"no backend":     "terraform {}\n",
		"does not parse": "terraform {\n",
	} {
		t.Run(name, func(t *testing.T) {
			l := &fakeRunLauncher{backend: []byte(body)}
			err := newRunner(t, "", l).Run(context.Background(), remoteJob())
			if err == nil || l.run != nil || !errors.Is(err, connectors.ErrNothingBuilt) {
				t.Fatalf("accepted: %v", err)
			}
		})
	}
}

func TestLocalRunStillNeedsTheTerraformBinary(t *testing.T) {
	err := newRunner(t, "", &failingLauncher{}).Run(context.Background(), remoteJob())
	if err == nil || !strings.Contains(err.Error(), "terraform binary") {
		t.Fatalf("err %v", err)
	}
}

// TestRemoteRunThroughTheECSLauncher runs a job through the real ECS launcher
// with fake AWS clients: the backend, the upload, the log and the outcome.
func TestRemoteRunThroughTheECSLauncher(t *testing.T) {
	const bucket = "clutter-adapter-runs-x"
	j := remoteJob()
	keys := runtask.KeysFor(j.RunID)
	m := runtasktest.New()
	uploaded := map[string]string{}
	m.OnPut = func(_, key string, body []byte) { uploaded[key] = string(body) }

	run := func(t *testing.T, script ...func() ecstypes.Task) (*runtasktest.ECS, error) {
		fe := &runtasktest.ECS{Script: script}
		e := &runtask.ECS{
			Settings: runtask.ECSSettings{
				Region: "ca-central-1", Cluster: "clutter-adapter", TaskDefinition: "clutter-adapter-run", Container: "run",
				Subnets: []string{"subnet-0123456789abcdef0"}, SecurityGroups: []string{"sg-0123456789abcdef0"},
				RunBucket: bucket, StateBucket: "clutter-adapter-state-x",
				RunFilesRoleARN: "arn:aws:iam::111122223333:role/ClutterAdapterRunFilesRole",
				PollInterval:    time.Millisecond, StopGrace: time.Second,
			},
			ECS: fe, S3: m,
			STS: &runtasktest.STS{AccessKeyID: "ASIAEXAMPLEEXAMPLE01", SecretAccessKey: "wJalrXUtnFEMIexampleSECRETkeyEXAMPLE0001", SessionToken: "FwoGZXIvYXdzEXAMPLEtokenEXAMPLE=="},
		}
		return fe, newRunner(t, "", e).Run(context.Background(), j)
	}

	fe, err := run(t, func() ecstypes.Task {
		m.Set(bucket, keys.Log, []byte("init done\napply done\n"))
		return runtasktest.StoppedTask("run", aws.Int32(0), "Essential container in task exited")
	})
	if err != nil {
		t.Fatal(err)
	}
	if fe.Run == nil {
		t.Fatal("no task was started")
	}
	if got := j.Log.(*bytes.Buffer).String(); got != "init done\napply done\n" {
		t.Fatalf("log %q", got)
	}
	backend := uploaded[keys.FilesPrefix+backendFileName]
	if !strings.Contains(backend, `backend "s3"`) || !strings.Contains(backend, "state/"+j.DeploymentID+"/terraform.tfstate") {
		t.Fatalf("uploaded backend %q", backend)
	}
	var spec runtask.RunSpec
	if err := json.Unmarshal([]byte(uploaded[keys.Spec]), &spec); err != nil || len(spec.Steps) != 2 || spec.Steps[1][0] != "apply" {
		t.Fatalf("spec %s (%v)", uploaded[keys.Spec], err)
	}
	if left := m.Keys(); len(left) != 0 {
		t.Fatalf("objects left after the run: %v", left)
	}

	// A failed apply in the task may have built something.
	_, err = run(t, func() ecstypes.Task {
		m.Set(bucket, keys.Result, []byte(`{"step":"apply","exit_code":1}`))
		return runtasktest.StoppedTask("run", aws.Int32(1), "Essential container in task exited")
	})
	if err == nil || errors.Is(err, connectors.ErrNothingBuilt) || err.Error() != "terraform apply: exit status 1" {
		t.Fatalf("failed apply: %v", err)
	}

	// A task that never ran built nothing.
	_, err = run(t, func() ecstypes.Task {
		return runtasktest.StoppedTask("run", nil, "CannotPullContainerError")
	})
	if !errors.Is(err, connectors.ErrNothingBuilt) {
		t.Fatalf("task that never ran: %v", err)
	}
}
