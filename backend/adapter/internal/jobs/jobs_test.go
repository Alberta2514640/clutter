package jobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/google/uuid"
)

func TestRedactWriter(t *testing.T) {
	var out bytes.Buffer
	w := newRedactWriter(&out)
	w.set([]string{"s3cr3t", "s3cr3t-long"})
	for _, chunk := range []string{"token=s3c", "r3t ok\nlong=s3cr3t-l", "ong\npartial s3cr3t"} {
		if _, err := io.WriteString(w, chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.flush(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "s3cr3t") || strings.Contains(got, "-long") {
		t.Fatalf("leaked: %q", got)
	}
	if got != "token=[redacted] ok\nlong=[redacted]\npartial [redacted]" {
		t.Fatalf("got %q", got)
	}
}

func TestRedactWriterLeavesOutALineOverTheCap(t *testing.T) {
	var out bytes.Buffer
	w := newRedactWriter(&out)
	w.set([]string{"s3cr3t"})
	long := strings.Repeat("x", terraform.MaxLine) + "s3cr3t"
	for _, chunk := range []string{"before\n", long[:100], long[100:], " tail\nafter s3cr3t\n"} {
		if _, err := io.WriteString(w, chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.flush(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "before\n"+terraform.LongLine+"after [redacted]\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTruncateKeepsValidText(t *testing.T) {
	s := truncate(strings.Repeat("é", 400) + "\xff")
	if len([]rune(s)) != maxFailureLength || !strings.HasPrefix(s, "é") {
		t.Fatalf("truncated to %d runes", len([]rune(s)))
	}
}

// store records states for one test.
type store struct {
	mu          sync.Mutex
	runs        map[string][]Status
	fail        map[string]string
	done        chan string
	deployments map[string]DeploymentStatus
}

func newStore() *store {
	return &store{runs: map[string][]Status{}, fail: map[string]string{}, done: make(chan string, 10)}
}

func (s *store) RunStarted(_ context.Context, id string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[id] = append(s.runs[id], StatusRunning)
	return nil
}

func (s *store) RunFinished(_ context.Context, id string, st Status, failure string, _ time.Time) error {
	s.mu.Lock()
	s.runs[id] = append(s.runs[id], st)
	s.fail[id] = failure
	s.mu.Unlock()
	s.done <- id
	return nil
}

func (s *store) SetDeploymentStatus(_ context.Context, id string, st DeploymentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deployments == nil {
		s.deployments = map[string]DeploymentStatus{}
	}
	s.deployments[id] = st
	return nil
}

type broker struct{}

func (broker) Credentials(context.Context, string) (connectors.Secrets, error) {
	return connectors.Secrets{"token": "tok-abcdef"}, nil
}

// fake is a connector whose plan does what the test needs.
type fake struct {
	plan  func(ctx context.Context, run connectors.Run) error
	apply func() error
	built map[string]string
}

func (fake) ID() string { return "fake" }
func (fake) Manifest() connectors.Manifest {
	m := connectors.Manifest{Connector: "fake", Title: "Fake", Version: "1"}
	for _, k := range connectors.StandardBlocks() {
		m.Blocks = append(m.Blocks, connectors.BlockSpec{Kind: k, Treatment: connectors.TreatmentAsIs, Title: string(k)})
	}
	return m
}
func (fake) Credentials() connectors.CredentialSpec {
	return connectors.CredentialSpec{Kind: connectors.CredentialStored, Fields: []connectors.CredentialField{{Name: "token", Label: "Token", Secret: true}}}
}
func (fake) VerifyConnection(context.Context, connectors.Connection) error { return nil }
func (fake) Validate(connectors.Graph, connectors.Deployment) []connectors.Problem {
	return nil
}
func (f fake) Plan(ctx context.Context, run connectors.Run, _ connectors.Graph, d connectors.Deployment) (connectors.PlanResult, error) {
	return connectors.PlanResult{}, f.plan(ctx, run)
}
func (f fake) Apply(context.Context, connectors.Run, connectors.Graph, connectors.Deployment) (connectors.RunResult, error) {
	if f.apply != nil {
		return connectors.RunResult{}, f.apply()
	}
	return connectors.RunResult{Resources: f.built}, nil
}
func (fake) Destroy(context.Context, connectors.Run, connectors.Deployment) (connectors.RunResult, error) {
	return connectors.RunResult{}, nil
}

func setup(t *testing.T, c connectors.Connector, logBuf io.Writer) (*Runner, *store, *logstore.Memory) {
	reg := connectors.NewRegistry()
	if err := reg.Register(c); err != nil {
		t.Fatal(err)
	}
	st, logs := newStore(), logstore.NewMemory()
	r := New(Config{Registry: reg, Store: st, Broker: broker{}, Logs: logs, Workers: 1,
		Logger: slog.New(slog.NewTextHandler(logBuf, nil))})
	r.Start()
	return r, st, logs
}

func job(id string) Job {
	dep := uuid.NewString()
	return Job{RunID: id, Action: connectors.ActionPlan, LogKey: logstore.KeyForRun(id),
		Deployment: connectors.Deployment{ID: dep, Connection: connectors.Connection{ConnectorID: "fake"}}}
}

func TestAFailingConnectorNeverLeaksTheCredential(t *testing.T) {
	var service bytes.Buffer
	r, st, logs := setup(t, fake{plan: func(_ context.Context, run connectors.Run) error {
		fmt.Fprintln(run.Log, "using tok-abcdef")
		return errors.New("cloud said: bad token tok-abcdef")
	}}, &service)
	defer r.Stop(context.Background()) //nolint:errcheck // test cleanup
	id := uuid.NewString()
	if err := r.Enqueue(job(id)); err != nil {
		t.Fatal(err)
	}
	<-st.done
	rc, err := logs.Open(logstore.KeyForRun(id))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc) //nolint:errcheck // memory reader
	for name, text := range map[string]string{"run log": string(b), "failure": st.fail[id], "service log": service.String()} {
		if strings.Contains(text, "tok-abcdef") {
			t.Fatalf("the %s holds the credential: %q", name, text)
		}
	}
	if st.runs[id][len(st.runs[id])-1] != StatusFailed {
		t.Fatalf("states %v", st.runs[id])
	}
}

// B's own apply line counts the blocks a connector built or changed in the
// run, and says so: it must not read as Terraform's resource count, which is
// printed earlier and larger, and an apply that changed nothing says so.
func TestTheApplyLineCountsBlocks(t *testing.T) {
	for n, want := range map[int]string{
		0: "Apply complete: everything is up to date.\n",
		1: "Apply complete: 1 block built or changed.\n",
		6: "Apply complete: 6 blocks built or changed.\n",
	} {
		built := map[string]string{}
		for i := range n {
			built[fmt.Sprintf("n%d", i)] = fmt.Sprintf("id-%d", i)
		}
		r, st, logs := setup(t, fake{built: built}, io.Discard)
		j := job(uuid.NewString())
		j.Action = connectors.ActionApply
		if err := r.Enqueue(j); err != nil {
			t.Fatal(err)
		}
		<-st.done
		if err := r.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		rc, err := logs.Open(j.LogKey)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(rc) //nolint:errcheck // memory reader
		if !strings.HasSuffix(string(b), want) || strings.Contains(string(b), "resources") {
			t.Errorf("%d blocks: log ends %q, want %q", n, string(b), want)
		}
	}
}

// runOnce runs one plan whose connector returns err, and returns the run's
// log, its failure and the service log.
func runOnce(t *testing.T, err error) (runLog, failure, service string) {
	t.Helper()
	var svc bytes.Buffer
	r, st, logs := setup(t, fake{plan: func(context.Context, connectors.Run) error { return err }}, &svc)
	id := uuid.NewString()
	if err := r.Enqueue(job(id)); err != nil {
		t.Fatal(err)
	}
	<-st.done
	if err := r.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	rc, oerr := logs.Open(logstore.KeyForRun(id))
	if oerr != nil {
		t.Fatal(oerr)
	}
	b, _ := io.ReadAll(rc) //nolint:errcheck // memory reader
	st.mu.Lock()
	defer st.mu.Unlock()
	return string(b), st.fail[id], svc.String()
}

// A launcher error names Clutter's own bucket, role and ECS reason: the user
// sees a fixed line, and the detail goes to the service log only.
func TestALauncherErrorStaysOutOfTheUsersLog(t *testing.T) {
	platform := []string{"clutter-adapter-runs-0a1b2c", "arn:aws:iam::111122223333:role/ClutterAdapterRunFilesRole", "CannotPullContainerError"}
	launch := fmt.Errorf("%w: upload run files: AccessDenied on s3://%s by %s; %s", runtask.ErrNotStarted, platform[0], platform[1], platform[2])
	runLog, failure, service := runOnce(t, fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, launch))
	for _, p := range platform {
		if strings.Contains(runLog, p) || strings.Contains(failure, p) {
			t.Fatalf("the user sees %q: log %q, failure %q", p, runLog, failure)
		}
		if !strings.Contains(service, p) {
			t.Fatalf("the service log lacks %q: %q", p, service)
		}
	}
	if !strings.Contains(runLog, failPlatform) || failure != failPlatform {
		t.Fatalf("log %q, failure %q; want the fixed line", runLog, failure)
	}
}

// A connector's UserError is meant for the user, even inside a launcher error.
func TestAUserErrorKeepsItsMessage(t *testing.T) {
	const msg = "Link a real account first."
	runLog, failure, _ := runOnce(t, fmt.Errorf("%w: %w", runtask.ErrNotStarted, &connectors.UserError{Message: msg}))
	if failure != msg || !strings.Contains(runLog, msg) || strings.Contains(runLog, failPlatform) {
		t.Fatalf("log %q, failure %q", runLog, failure)
	}
}

func TestAPanickingConnectorFailsTheRun(t *testing.T) {
	r, st, _ := setup(t, fake{plan: func(context.Context, connectors.Run) error { panic("boom tok-abcdef") }}, io.Discard)
	defer r.Stop(context.Background()) //nolint:errcheck // test cleanup
	id := uuid.NewString()
	if err := r.Enqueue(job(id)); err != nil {
		t.Fatal(err)
	}
	<-st.done
	if got := st.runs[id]; got[len(got)-1] != StatusFailed {
		t.Fatalf("states %v", got)
	}
}

func TestStopCancelsARunningRun(t *testing.T) {
	started := make(chan struct{})
	r, st, _ := setup(t, fake{plan: func(ctx context.Context, _ connectors.Run) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}}, io.Discard)
	id := uuid.NewString()
	if err := r.Enqueue(job(id)); err != nil {
		t.Fatal(err)
	}
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := r.Stop(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stop: %v", err)
	}
	select {
	case <-st.done:
	default:
		t.Fatal("the cancelled run did not record its end")
	}
	if got := st.runs[id]; got[len(got)-1] != StatusFailed {
		t.Fatalf("states %v", got)
	}
	if err := r.Enqueue(job(uuid.NewString())); !errors.Is(err, ErrStopped) {
		t.Fatalf("enqueue after stop: %v", err)
	}
}

// A failed apply marks the deployment failed (it may have left resources), but
// stepFailer stands in for terraform: it fails the step named failAt.
type stepFailer struct{ failAt string }

func (s stepFailer) Launch(_ context.Context, t runtask.Task) error {
	if t.Args[0] == s.failAt {
		return errors.New("exit status 1")
	}
	return nil
}

// runnerError is what the shared Terraform runner returns for an apply of
// files when the step failAt fails.
func runnerError(t *testing.T, files map[string][]byte, failAt string) error {
	t.Helper()
	root := t.TempDir()
	r := &terraform.Runner{Bin: "/bin/terraform-not-run", WorkDir: filepath.Join(root, "work"), StateDir: filepath.Join(root, "state"),
		PluginCacheDir: filepath.Join(root, "plugins"), Launcher: stepFailer{failAt: failAt}}
	err := r.Run(context.Background(), terraform.Job{DeploymentID: uuid.NewString(), RunID: uuid.NewString(),
		Action: connectors.ActionApply, Files: files, Log: io.Discard})
	if err == nil {
		t.Fatal("the runner did not fail")
	}
	return err
}

// one that failed before Terraform touched the cloud leaves the status alone,
// so it does not lock the connection against deletes.
func TestOnlyAFailureAfterTheCloudMarksTheDeploymentFailed(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want DeploymentStatus
		set  bool
	}{
		{"failed in the cloud", errors.New("terraform apply failed"), DeploymentFailed, true},
		{"failed before the cloud", fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, &connectors.UserError{Message: "needs a real account"}), "", false},
		// The shared runner's own failures before plan, apply or destroy.
		{"refused render", runnerError(t, map[string][]byte{"main.tf": []byte("module \"m\" {\n  source = \"./x\"\n}\n")}, ""), "", false},
		{"terraform init failed", runnerError(t, map[string][]byte{"main.tf": []byte("resource \"terraform_data\" \"n1\" {}\n")}, "init"), "", false},
		{"terraform apply failed", runnerError(t, map[string][]byte{"main.tf": []byte("resource \"terraform_data\" \"n1\" {}\n")}, "apply"), DeploymentFailed, true},
		{"succeeded", nil, DeploymentLive, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, st, _ := setup(t, fake{apply: func() error { return c.err }}, io.Discard)
			j := job(uuid.NewString())
			j.Action = connectors.ActionApply
			if err := r.Enqueue(j); err != nil {
				t.Fatal(err)
			}
			<-st.done
			if err := r.Stop(context.Background()); err != nil {
				t.Fatal(err)
			}
			st.mu.Lock()
			got, set := st.deployments[j.Deployment.ID]
			st.mu.Unlock()
			if set != c.set || got != c.want {
				t.Fatalf("deployment status = %q (set %v), want %q (set %v)", got, set, c.want, c.set)
			}
		})
	}
}
