package unit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/jobs"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/tests/testkit"
	"github.com/google/uuid"
)

// renamed is a connector whose id does not match its manifest.
type renamed struct{ *mock.Connector }

func (renamed) ID() string { return "not-mock" }

func TestRegistryLookup(t *testing.T) {
	reg := connectors.NewRegistry()
	for _, c := range realConnectors(t) {
		if err := reg.Register(c); err != nil {
			t.Fatalf("register %s: %v", c.ID(), err)
		}
	}
	for _, id := range []string{"aws", "general-openstack", "mock"} {
		if c, ok := reg.Get(id); !ok || c.ID() != id {
			t.Fatalf("Get(%q) = %v, %v", id, c, ok)
		}
	}
	for _, id := range []string{"", "patrii", "AWS", "general-digitalocean"} {
		if _, ok := reg.Get(id); ok {
			t.Fatalf("Get(%q) found a connector", id)
		}
	}
	all := reg.All()
	if len(all) != 3 || all[0].ID() != "aws" || all[2].ID() != "mock" {
		t.Fatal("All is not every connector sorted by id")
	}
	if err := reg.Register(testkit.NewMock()); err == nil {
		t.Fatal("a second connector with the same id was registered")
	}
	if err := connectors.NewRegistry().Register(renamed{mock.New()}); err == nil {
		t.Fatal("a connector whose manifest names another id was registered")
	}
}

// leaky is the mock connector, except that its plan prints the token it was
// handed, as a careless provider might.
type leaky struct{ *mock.Connector }

func (l leaky) Plan(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.PlanResult, error) {
	fmt.Fprintf(run.Log, "debug: token=%s\n", d.Connection.Credentials["token"])
	return l.Connector.Plan(ctx, run, g, d)
}

type runnerKit struct {
	rec    *contract.Recorder
	logs   *logstore.Memory
	runner *jobs.Runner
}

func newRunner(t *testing.T, c connectors.Connector, creds connectors.Secrets) *runnerKit {
	reg := connectors.NewRegistry()
	if err := reg.Register(c); err != nil {
		t.Fatal(err)
	}
	k := &runnerKit{rec: contract.NewRecorder(), logs: logstore.NewMemory()}
	k.runner = jobs.New(jobs.Config{Registry: reg, Store: k.rec, Broker: contract.StaticBroker{Secrets: creds}, Logs: k.logs, Workers: 1})
	k.runner.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = k.runner.Stop(ctx) //nolint:errcheck // test cleanup
	})
	return k
}

func (k *runnerKit) run(t *testing.T, a connectors.Action, g connectors.Graph, d connectors.Deployment, prior jobs.DeploymentStatus) string {
	t.Helper()
	id := uuid.NewString()
	k.rec.Queued(id)
	if err := k.runner.Enqueue(jobs.Job{RunID: id, Action: a, Graph: g, Deployment: d, SecretRef: "ref", LogKey: logstore.KeyForRun(id), Prior: prior}); err != nil {
		t.Fatal(err)
	}
	if !k.rec.Wait(id, 10*time.Second) {
		t.Fatalf("%s did not finish", a)
	}
	return id
}

func (k *runnerKit) log(t *testing.T, id string) string {
	r, err := k.logs.Open(logstore.KeyForRun(id))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

func mockGraph() connectors.Graph {
	return graph([]connectors.Edge{{Source: "n1", Target: "n2"}},
		block("n1", connectors.KindVM, "web", map[string]any{"size": "small"}),
		block("n2", connectors.KindNetwork, "net", nil))
}

func statuses(ss ...jobs.Status) string { return fmt.Sprint(ss) }

func TestJobRunnerTracksAppliesAndDestroys(t *testing.T) {
	k := newRunner(t, mock.New(), connectors.Secrets{"token": mock.ValidToken})
	d := mockDeployment(nil)
	for _, step := range []struct {
		a    connectors.Action
		want jobs.DeploymentStatus
	}{{connectors.ActionPlan, jobs.DeploymentNew}, {connectors.ActionApply, jobs.DeploymentLive}, {connectors.ActionDestroy, jobs.DeploymentDestroyed}} {
		id := k.run(t, step.a, mockGraph(), d, jobs.DeploymentNew)
		got := k.rec.History(id)
		if fmt.Sprint(got) != statuses(jobs.StatusQueued, jobs.StatusRunning, jobs.StatusCompleted) {
			t.Fatalf("%s moved through %v (failure %q)", step.a, got, k.rec.Failure(id))
		}
		if step.a != connectors.ActionPlan {
			if s := k.rec.Deployment(d.ID); s != step.want {
				t.Fatalf("after %s the deployment is %q, want %q", step.a, s, step.want)
			}
		}
		if k.log(t, id) == "" {
			t.Fatalf("%s left no log", step.a)
		}
	}
}

func TestJobRunnerFailsRunsWithASafeReason(t *testing.T) {
	cases := []struct {
		name  string
		creds connectors.Secrets
		g     connectors.Graph
		d     connectors.Deployment
		want  string
	}{
		{"refused credentials", connectors.Secrets{"token": "wrong-token-0000"}, mockGraph(), mockDeployment(nil), "refused this token"},
		{"no credentials at all", nil, mockGraph(), mockDeployment(nil), "could not read this connection's credentials"},
		{"a block the cloud cannot build", connectors.Secrets{"token": mock.ValidToken},
			graph(nil, block("b", connectors.KindBucket, "files", nil)), mockDeployment(nil), ""},
		{"a connector this server does not have", connectors.Secrets{"token": mock.ValidToken}, mockGraph(),
			func() connectors.Deployment { d := mockDeployment(nil); d.Connection.ConnectorID = "patrii"; return d }(), "not available"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k := newRunner(t, mock.New(), c.creds)
			id := k.run(t, connectors.ActionApply, c.g, c.d, jobs.DeploymentNew)
			h := k.rec.History(id)
			if h[len(h)-1] != jobs.StatusFailed {
				t.Fatalf("history %v", h)
			}
			f := k.rec.Failure(id)
			if f == "" || (c.want != "" && !strings.Contains(f, c.want)) {
				t.Fatalf("failure %q, want it to mention %q", f, c.want)
			}
			if strings.Contains(f, mock.ValidToken) {
				t.Fatal("the failure holds the token")
			}
			if s := k.rec.Deployment(c.d.ID); s == jobs.DeploymentLive || s == jobs.DeploymentFailed {
				t.Fatalf("a run that built nothing changed the deployment to %q", s)
			}
		})
	}
}

func TestJobRunnerRemovesCredentialValuesFromLogs(t *testing.T) {
	k := newRunner(t, leaky{mock.New()}, connectors.Secrets{"token": mock.ValidToken})
	id := k.run(t, connectors.ActionPlan, mockGraph(), mockDeployment(nil), jobs.DeploymentNew)
	log := k.log(t, id)
	if strings.Contains(log, mock.ValidToken) {
		t.Fatalf("the log holds the token:\n%s", log)
	}
	if !strings.Contains(log, "debug: token=") {
		t.Fatalf("the connector's line is missing:\n%s", log)
	}
}

func TestJobRunnerQueueLimits(t *testing.T) {
	reg := connectors.NewRegistry()
	_ = reg.Register(mock.New()) //nolint:errcheck // a fresh registry
	r := jobs.New(jobs.Config{Registry: reg, Store: contract.NewRecorder(), Broker: contract.StaticBroker{}, Logs: logstore.NewMemory(), QueueSize: 1})
	// Not started, so the one slot stays full.
	if err := r.Enqueue(jobs.Job{RunID: uuid.NewString(), Action: connectors.ActionPlan}); err != nil {
		t.Fatal(err)
	}
	if err := r.Enqueue(jobs.Job{RunID: uuid.NewString(), Action: connectors.ActionPlan}); !errors.Is(err, jobs.ErrQueueFull) {
		t.Fatalf("a full queue answered %v", err)
	}
	r.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.Enqueue(jobs.Job{RunID: uuid.NewString(), Action: connectors.ActionPlan}); !errors.Is(err, jobs.ErrStopped) {
		t.Fatalf("a stopped runner answered %v", err)
	}
}
