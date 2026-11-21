package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/split"
	"github.com/google/uuid"
)

// scopedSecret is the key a target cloud hands on. It must never reach a log.
const scopedSecret = "spy-scoped-secret/ABCDEF0123456789"

// call is one connector call a split run made.
type call struct {
	step       string // plan, apply or destroy
	connection string
	token      string
	runID      string
	imports    map[string]map[string]connectors.Value
}

// spy is a cross-cloud connector that records every call with the
// credentials it was handed.
type spy struct {
	mu    sync.Mutex
	calls []call
	// fail makes the apply on these connections fail; nothingBuilt says the
	// failure came before the cloud.
	fail         map[string]bool
	nothingBuilt bool
}

func (*spy) ID() string { return "spy" }
func (*spy) Manifest() connectors.Manifest {
	m := connectors.Manifest{Connector: "spy", Title: "Spy", Version: "1"}
	for _, k := range connectors.StandardBlocks() {
		m.Blocks = append(m.Blocks, connectors.BlockSpec{Kind: k, Treatment: connectors.TreatmentAsIs, Title: string(k)})
	}
	m.Edges = []connectors.EdgeSpec{{Source: connectors.KindVM, Target: connectors.KindBucket, Meaning: "reads"}}
	return m
}
func (*spy) Credentials() connectors.CredentialSpec {
	return connectors.CredentialSpec{Kind: connectors.CredentialStored, Fields: []connectors.CredentialField{{Name: "token", Label: "Token", Secret: true}}}
}

func (s *spy) record(step string, run connectors.Run, d connectors.Deployment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := call{step: step, connection: d.Connection.ID, token: d.Connection.Credentials["token"], runID: run.ID, imports: map[string]map[string]connectors.Value{}}
	if d.Cross != nil {
		for _, imp := range d.Cross.Imports {
			c.imports[imp.Key] = imp.Values
		}
	}
	s.calls = append(s.calls, c)
}

func (s *spy) VerifyConnection(_ context.Context, c connectors.Connection) error {
	if c.Credentials["token"] != tokenFor(c.ID) {
		return errors.New("wrong token")
	}
	return nil
}
func (*spy) Validate(connectors.Graph, connectors.Deployment) []connectors.Problem   { return nil }
func (*spy) CrossCheck(connectors.Graph, connectors.Deployment) []connectors.Problem { return nil }
func (s *spy) Plan(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.PlanResult, error) {
	res, _, err := s.PlanCross(ctx, run, g, d)
	return res, err
}
func (s *spy) Apply(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.RunResult, error) {
	res, _, err := s.ApplyCross(ctx, run, g, d)
	return res, err
}
func (s *spy) PlanCross(_ context.Context, run connectors.Run, _ connectors.Graph, d connectors.Deployment) (connectors.PlanResult, connectors.Outputs, error) {
	s.record("plan", run, d)
	return connectors.PlanResult{Add: 1}, connectors.Outputs{}, nil
}
func (s *spy) ApplyCross(_ context.Context, run connectors.Run, _ connectors.Graph, d connectors.Deployment) (connectors.RunResult, connectors.Outputs, error) {
	s.record("apply", run, d)
	// A careless provider would print what it was handed.
	for _, imp := range d.Cross.Imports {
		fmt.Fprintf(run.Log, "debug: got %s\n", imp.Values[connectors.ValueSecretAccessKey].Text)
	}
	if s.fail[d.Connection.ID] {
		if s.nothingBuilt {
			return connectors.RunResult{}, nil, fmt.Errorf("%w: refused", connectors.ErrNothingBuilt)
		}
		return connectors.RunResult{}, nil, errors.New("terraform apply failed")
	}
	out := connectors.Outputs{}
	for _, e := range d.Cross.Exports {
		out[e.Key] = map[string]connectors.Value{
			connectors.ValueBucket:          {Text: "c1-files-0a1b"},
			connectors.ValueRegion:          {Text: "ca-central-1"},
			connectors.ValueAccessKeyID:     {Text: "AKIASPY0000000000001"},
			connectors.ValueSecretAccessKey: {Text: scopedSecret, Sensitive: true},
		}
	}
	return connectors.RunResult{Resources: map[string]string{}}, out, nil
}
func (s *spy) Destroy(_ context.Context, run connectors.Run, d connectors.Deployment) (connectors.RunResult, error) {
	s.record("destroy", run, d)
	return connectors.RunResult{}, nil
}

func (s *spy) steps() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, c := range s.calls {
		out = append(out, c.step+":"+c.connection)
	}
	return out
}

func tokenFor(connectionID string) string { return "tok-" + connectionID }

// splitBroker hands each secret reference (a connection id) its own token.
type splitBroker struct{}

func (splitBroker) Credentials(_ context.Context, ref string) (connectors.Secrets, error) {
	return connectors.Secrets{"token": tokenFor(ref)}, nil
}

// partStore also records a split run's progress.
type partStore struct {
	*store
	pmu   sync.Mutex
	parts map[string][]PartProgress
}

func (p *partStore) RunParts(_ context.Context, runID string, parts []PartProgress) error {
	p.pmu.Lock()
	defer p.pmu.Unlock()
	p.parts[runID] = parts
	return nil
}

// splitJob is a split deployment: vm on the primary reads files, a bucket
// placed on a second cloud.
func splitJob(action connectors.Action, prior DeploymentStatus) (Job, string, string) {
	g := connectors.Graph{Blocks: map[string]connectors.Block{
		"vm":    {ID: "vm", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{}},
		"files": {ID: "files", Kind: connectors.KindBucket, Name: "files", Fields: map[string]any{}},
	}, Edges: []connectors.Edge{{Source: "vm", Target: "files"}}}
	c := &spy{}
	primary, other := uuid.NewString(), uuid.NewString()
	plan := split.Build(g, uuid.NewString(), nil, []split.Cloud{
		{ConnectionID: primary, Name: "main", Connector: c, Region: "r1", SecretRef: primary, Primary: true},
		{ConnectionID: other, Name: "data", Connector: c, Region: "r1", SecretRef: other, Nodes: []string{"files"}},
	})
	id := uuid.NewString()
	return Job{RunID: id, Action: action, Graph: g, Deployment: plan.Parts[1].Deployment, LogKey: logstore.KeyForRun(id),
		Parts: plan.Parts, Prior: prior}, primary, other
}

func runSplit(t *testing.T, c *spy, j Job) (*partStore, string) {
	t.Helper()
	reg := connectors.NewRegistry()
	if err := reg.Register(c); err != nil {
		t.Fatal(err)
	}
	st := &partStore{store: newStore(), parts: map[string][]PartProgress{}}
	logs := logstore.NewMemory()
	r := New(Config{Registry: reg, Store: st, Broker: splitBroker{}, Logs: logs, Workers: 1, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	r.Start()
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
	return st, string(b)
}

// spyOf returns the spy every part of j uses.
func spyOf(j Job) *spy { return j.Parts[0].Connector.(*spy) }

func TestASplitApplyRunsTargetsFirstAndHandsTheKeyOn(t *testing.T) {
	j, primary, other := splitJob(connectors.ActionApply, DeploymentNew)
	c := spyOf(j)
	st, log := runSplit(t, c, j)

	if got, want := strings.Join(c.steps(), " "), "apply:"+other+" apply:"+primary; got != want {
		t.Fatalf("steps %q, want %q", got, want)
	}
	// The VM's cloud got the bucket's cloud's values, the key included.
	last := c.calls[len(c.calls)-1]
	var handed map[string]connectors.Value
	for _, v := range last.imports {
		handed = v
	}
	if handed[connectors.ValueSecretAccessKey].Text != scopedSecret || handed[connectors.ValueBucket].Text != "c1-files-0a1b" {
		t.Fatalf("the VM's cloud was handed %v", handed)
	}
	if strings.Contains(log, scopedSecret) || strings.Contains(st.fail[j.RunID], scopedSecret) {
		t.Fatalf("the scoped key reached the log: %s", log)
	}
	if st.deployments[j.Deployment.ID] != DeploymentLive {
		t.Fatalf("deployment %q", st.deployments[j.Deployment.ID])
	}
	parts := st.parts[j.RunID]
	if len(parts) != 2 || parts[0].ConnectionID != other || parts[0].Status != StatusCompleted || parts[1].Status != StatusCompleted {
		t.Fatalf("progress %+v", parts)
	}
	// Each step has its own run id, so its folder and task are its own.
	if c.calls[0].runID == c.calls[1].runID || c.calls[0].runID == j.RunID {
		t.Fatal("two steps share a run id")
	}
}

// Each cloud of a split apply names its own blocks after its apply line, so
// "What was built" reads every block the deployment holds, each once.
func TestASplitApplyNamesEachCloudsBlocks(t *testing.T) {
	j, _, _ := splitJob(connectors.ActionApply, DeploymentNew)
	_, log := runSplit(t, spyOf(j), j)
	for _, line := range []string{"Built: files bucket files\n", "Built: vm vm web\n"} {
		if n := strings.Count(log, line); n != 1 {
			t.Fatalf("%q appears %d times in %s", line, n, log)
		}
	}
	// The bucket's cloud runs first, so its line comes first, after its own apply line.
	files, vm := strings.Index(log, "Built: files"), strings.Index(log, "Built: vm")
	if first := strings.Index(log, "apply complete"); first < 0 || first > files || files > vm {
		t.Fatalf("the Built lines are out of place: %s", log)
	}
}

func TestEachCloudGetsOnlyItsOwnCredentials(t *testing.T) {
	for _, a := range []connectors.Action{connectors.ActionPlan, connectors.ActionApply, connectors.ActionDestroy} {
		j, _, _ := splitJob(a, DeploymentLive)
		c := spyOf(j)
		runSplit(t, c, j)
		if len(c.calls) != 2 {
			t.Fatalf("%s made %d calls", a, len(c.calls))
		}
		for _, call := range c.calls {
			if call.token != tokenFor(call.connection) {
				t.Fatalf("%s on %s was handed another cloud's token", call.step, call.connection)
			}
		}
	}
}

func TestASplitDestroyRunsInReverse(t *testing.T) {
	j, primary, other := splitJob(connectors.ActionDestroy, DeploymentLive)
	c := spyOf(j)
	st, _ := runSplit(t, c, j)
	if got, want := strings.Join(c.steps(), " "), "destroy:"+primary+" destroy:"+other; got != want {
		t.Fatalf("steps %q, want %q", got, want)
	}
	if st.deployments[j.Deployment.ID] != DeploymentDestroyed {
		t.Fatalf("deployment %q", st.deployments[j.Deployment.ID])
	}
}

func TestAFailureInTheSecondCloudRollsBackTheFirst(t *testing.T) {
	j, primary, other := splitJob(connectors.ActionApply, DeploymentNew)
	c := spyOf(j)
	c.fail = map[string]bool{primary: true}
	st, log := runSplit(t, c, j)

	// The second cloud's partial build and then the first are destroyed,
	// newest first.
	want := "apply:" + other + " apply:" + primary + " destroy:" + primary + " destroy:" + other
	if got := strings.Join(c.steps(), " "); got != want {
		t.Fatalf("steps %q, want %q", got, want)
	}
	if got := st.runs[j.RunID]; got[len(got)-1] != StatusFailed {
		t.Fatalf("run %v", got)
	}
	if f := st.fail[j.RunID]; !strings.Contains(f, "main (Spy) failed") || !strings.Contains(f, "rolled back") {
		t.Fatalf("failure %q", f)
	}
	// Nothing is left, so the deployment is as it was before the run.
	if st.deployments[j.Deployment.ID] != DeploymentNew {
		t.Fatalf("deployment %q", st.deployments[j.Deployment.ID])
	}
	parts := st.parts[j.RunID]
	if parts[0].Status != StatusRolledBack || parts[1].Status != StatusRolledBack {
		t.Fatalf("progress %+v", parts)
	}
	if strings.Contains(log, scopedSecret) {
		t.Fatal("the scoped key reached the log")
	}
}

func TestALiveDeploymentIsNotRolledBack(t *testing.T) {
	j, primary, other := splitJob(connectors.ActionApply, DeploymentLive)
	c := spyOf(j)
	c.fail = map[string]bool{primary: true}
	st, _ := runSplit(t, c, j)
	if got, want := strings.Join(c.steps(), " "), "apply:"+other+" apply:"+primary; got != want {
		t.Fatalf("steps %q, want %q", got, want)
	}
	if st.deployments[j.Deployment.ID] != DeploymentFailed {
		t.Fatalf("deployment %q", st.deployments[j.Deployment.ID])
	}
	if !strings.Contains(st.fail[j.RunID], "does not undo an update") {
		t.Fatalf("failure %q", st.fail[j.RunID])
	}
}

func TestAFirstCloudThatBuiltNothingLeavesTheStatus(t *testing.T) {
	j, _, other := splitJob(connectors.ActionApply, DeploymentNew)
	c := spyOf(j)
	c.fail, c.nothingBuilt = map[string]bool{other: true}, true
	st, _ := runSplit(t, c, j)
	if got := strings.Join(c.steps(), " "); got != "apply:"+other {
		t.Fatalf("steps %q", got)
	}
	if _, set := st.deployments[j.Deployment.ID]; set {
		t.Fatal("a run that built nothing changed the deployment's status")
	}
	if parts := st.parts[j.RunID]; parts[1].Status != StatusSkipped {
		t.Fatalf("progress %+v", parts)
	}
}

func TestAPlanHandsOnWhatTheStateHolds(t *testing.T) {
	j, primary, other := splitJob(connectors.ActionPlan, DeploymentNew)
	c := spyOf(j)
	st, log := runSplit(t, c, j)
	if got, want := strings.Join(c.steps(), " "), "plan:"+other+" plan:"+primary; got != want {
		t.Fatalf("steps %q, want %q", got, want)
	}
	if !strings.Contains(log, "Plan: 2 to add, 0 to change, 0 to destroy.") {
		t.Fatalf("log %s", log)
	}
	if _, set := st.deployments[j.Deployment.ID]; set {
		t.Fatal("a plan changed the deployment's status")
	}
}
