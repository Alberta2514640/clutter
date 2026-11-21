package jobs

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/google/uuid"
)

type mockBroker struct{}

func (mockBroker) Credentials(context.Context, string) (connectors.Secrets, error) {
	return connectors.Secrets{"token": mock.ValidToken}, nil
}

// mockRun runs one action on the mock connector and returns its log and
// failure.
func mockRun(t *testing.T, r *Runner, st *store, logs *logstore.Memory, a connectors.Action, g connectors.Graph, d connectors.Deployment) (string, string) {
	t.Helper()
	id := uuid.NewString()
	j := Job{RunID: id, Action: a, Graph: g, Deployment: d, LogKey: logstore.KeyForRun(id), ConnectionName: "audit mock"}
	if err := r.Enqueue(j); err != nil {
		t.Fatal(err)
	}
	<-st.done
	st.mu.Lock()
	failure := st.fail[id]
	st.mu.Unlock()
	return readLog(t, logs, j.LogKey), failure
}

// The mock's runs read as a person would say them: block names, not node ids
// or cloud-side names; an update says which blocks it left unchanged; a
// destroy counts the blocks it removed; and a block named fail-me fails the
// run on purpose, with the reason in the log.
func TestTheMockLogsNameBlocksAndCanFailOnPurpose(t *testing.T) {
	reg := connectors.NewRegistry()
	if err := reg.Register(mock.New()); err != nil {
		t.Fatal(err)
	}
	st, logs := newStore(), logstore.NewMemory()
	r := New(Config{Registry: reg, Store: st, Broker: mockBroker{}, Logs: logs, Workers: 1,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return testNow }})
	r.Start()
	defer r.Stop(context.Background()) //nolint:errcheck // test cleanup

	g := connectors.Graph{
		Blocks: map[string]connectors.Block{
			"n1": {ID: "n1", Kind: connectors.KindVM, Name: "server", Fields: map[string]any{"size": "small", "public": true}},
			"n2": {ID: "n2", Kind: connectors.KindNetwork, Name: "net", Fields: map[string]any{}},
		},
		Edges: []connectors.Edge{{Source: "n1", Target: "n2"}},
	}
	d := connectors.Deployment{ID: uuid.NewString(), Prefix: "c1a2b3", Settings: map[string]any{"image": "debian-12"},
		Connection: connectors.Connection{ConnectorID: mock.ID, Region: "mock-1"}}

	plan, _ := mockRun(t, r, st, logs, connectors.ActionPlan, g, d)
	deploy, _ := mockRun(t, r, st, logs, connectors.ActionApply, g, d)
	g.Blocks["n3"] = connectors.Block{ID: "n3", Kind: connectors.KindVolume, Name: "data", Fields: map[string]any{"size_gb": float64(20)}}
	g.Edges = append(g.Edges, connectors.Edge{Source: "n3", Target: "n1"})
	update, _ := mockRun(t, r, st, logs, connectors.ActionApply, g, d)
	destroy, _ := mockRun(t, r, st, logs, connectors.ActionDestroy, connectors.Graph{}, d)
	g.Blocks["n3"] = connectors.Block{ID: "n3", Kind: connectors.KindVolume, Name: mock.FailName, Fields: map[string]any{"size_gb": float64(20)}}
	failed, failure := mockRun(t, r, st, logs, connectors.ActionApply, g, d)

	for name, l := range map[string]string{"plan": plan, "deploy": deploy, "update": update, "destroy": destroy, "failed deploy": failed} {
		t.Logf("%s log:\n%s", name, l)
		if strings.Contains(l, d.ID) || strings.Contains(l, d.Prefix) {
			t.Errorf("the %s log names the deployment's id or prefix:\n%s", name, l)
		}
	}
	checks := []struct {
		name, log string
		want      []string
	}{
		{"plan", plan, []string{"Plan on Mock cloud (audit mock), started 10:42:03 UTC.\n", "  + server (VM)\n", "Plan: 2 to add, 0 to change, 0 to destroy.\n"}},
		{"deploy", deploy, []string{"Deploy on Mock cloud (audit mock)", "Created server (VM).\n", "Apply complete: 2 blocks built or changed.\n", "Built: n1 vm server\n"}},
		{"update", update, []string{"Updated server (VM).\n", "Unchanged net (Network).\n", "Created data (Volume).\n", "Apply complete: 2 blocks built or changed.\n"}},
		{"destroy", destroy, []string{"Destroyed data (Volume).\n", "Destroy complete: 3 blocks removed.\n"}},
		{"failed deploy", failed, []string{"Error: The mock cloud failed on purpose at fail-me (Volume)."}},
	}
	for _, c := range checks {
		for _, w := range c.want {
			if !strings.Contains(c.log, w) {
				t.Errorf("the %s log lacks %q:\n%s", c.name, w, c.log)
			}
		}
	}
	if !strings.Contains(failure, "failed on purpose") {
		t.Fatalf("failure %q", failure)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if got := st.deployments[d.ID]; got != DeploymentFailed {
		t.Fatalf("after the failed deploy the deployment is %q, want failed", got)
	}
}
