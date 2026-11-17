// Package contract holds the tests every connector must pass. A connector's
// own test calls Run with a Fixture; the tests drive it through B's real job
// runner, so they prove the connector and the core agree.
package contract

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/jobs"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/split"
	"github.com/google/uuid"
)

// Fixture is what a connector's test supplies.
type Fixture struct {
	// New returns a fresh connector.
	New func() connectors.Connector
	// Region is a region the connector accepts.
	Region string
	// Good are credentials VerifyConnection accepts; Bad are ones it refuses.
	Good connectors.Secrets
	Bad  connectors.Secrets
	// Graph is a graph the connector can build with Settings.
	Graph    connectors.Graph
	Settings map[string]any
	// Timeout bounds one run. Zero means one minute.
	Timeout time.Duration
	// Cross is required when the connector implements connectors.CrossCloud.
	Cross *CrossFixture
}

// CrossFixture is a split deployment over two clouds of the connector: Graph
// holds both clouds' blocks and an edge from the VM SourceVM (on the first
// cloud) to the public VM TargetVM (on the second, with every block in
// Placed), so the second cloud hands the first an address.
type CrossFixture struct {
	Graph    connectors.Graph
	Placed   []string
	SourceVM string
	TargetVM string
}

// Run runs every contract test against the fixture's connector.
func Run(t *testing.T, f Fixture) {
	t.Helper()
	if f.Timeout == 0 {
		f.Timeout = time.Minute
	}
	t.Run("manifest is well formed", func(t *testing.T) { manifest(t, f) })
	t.Run("manifest says how the UI shows and explains the cloud, every block it builds and every edge", func(t *testing.T) { display(t, f) })
	t.Run("display keys name icon files the frontend serves", func(t *testing.T) { iconFiles(t, f) })
	t.Run("credential spec is well formed", func(t *testing.T) { credentialSpec(t, f) })
	t.Run("verify refuses bad credentials", func(t *testing.T) { verify(t, f) })
	t.Run("the core's checks refuse unsupported blocks, honour exclusions, and Validate adds only its own", func(t *testing.T) { validate(t, f) })
	t.Run("plan apply destroy move a run through its states with logs", func(t *testing.T) { lifecycle(t, f) })
	t.Run("a run with refused credentials fails", func(t *testing.T) { refusedRun(t, f) })
	if _, ok := f.New().(connectors.CrossCloud); ok {
		if f.Cross == nil {
			t.Fatal("the connector implements connectors.CrossCloud, so its fixture needs Cross")
		}
		t.Run("a split deployment runs on two clouds in order", func(t *testing.T) { splitLifecycle(t, f) })
		t.Run("a preview never holds a value another cloud hands a VM", func(t *testing.T) { splitPreview(t, f) })
	}
}

// HasProblem reports whether ps holds a problem with this code on the block
// nodeID; an empty nodeID matches a problem on any block or none.
func HasProblem(ps []connectors.Problem, nodeID, code string) bool {
	for _, p := range ps {
		if p.Code == code && (nodeID == "" || p.NodeID == nodeID) {
			return true
		}
	}
	return false
}

// splitPlan places the fixture's second cloud on its own connection.
func splitPlan(t *testing.T, c connectors.Connector, f Fixture) split.Plan {
	t.Helper()
	plan := split.Build(f.Cross.Graph, uuid.NewString(), nil, []split.Cloud{
		{ConnectionID: uuid.NewString(), Name: "first", Connector: c, Region: f.Region, Settings: f.Settings, SecretRef: "first", Primary: true},
		{ConnectionID: uuid.NewString(), Name: "second", Connector: c, Region: f.Region, Settings: f.Settings, SecretRef: "second", Nodes: f.Cross.Placed},
	})
	if len(plan.Problems) > 0 {
		t.Fatalf("the cross fixture does not split: %+v", plan.Problems)
	}
	if len(plan.Edges) != 1 || plan.Edges[0].Source != f.Cross.SourceVM || plan.Edges[0].Target != f.Cross.TargetVM {
		t.Fatalf("the cross fixture's edge: %+v", plan.Edges)
	}
	for _, p := range plan.Parts {
		cc := p.Connector.(connectors.CrossCloud)
		ps := append(capability.CheckConnector(p.Connector, f.Cross.Graph, p.Deployment).Problems, cc.CrossCheck(f.Cross.Graph, p.Deployment)...)
		if len(ps) > 0 {
			t.Fatalf("cloud %s refuses its part: %+v", p.Name, ps)
		}
	}
	return plan
}

func splitLifecycle(t *testing.T, f Fixture) {
	c := f.New()
	plan := splitPlan(t, c, f)
	h := newHarness(t, c, f.Good)
	if plan.Parts[0].Name != "second" {
		t.Fatalf("the cloud that hands the address is not first: %s", plan.Parts[0].Name)
	}
	for _, a := range []connectors.Action{connectors.ActionPlan, connectors.ActionApply, connectors.ActionDestroy} {
		runID := uuid.NewString()
		h.rec.Queued(runID)
		j := jobs.Job{RunID: runID, Action: a, Graph: f.Cross.Graph, Deployment: plan.Parts[1].Deployment, LogKey: logstore.KeyForRun(runID),
			Parts: plan.Parts, Prior: jobs.DeploymentNew}
		if err := h.runner.Enqueue(j); err != nil {
			t.Fatal(err)
		}
		if !h.rec.Wait(runID, f.Timeout) {
			t.Fatalf("%s did not finish", a)
		}
		want := []jobs.Status{jobs.StatusQueued, jobs.StatusRunning, jobs.StatusCompleted}
		if got := h.rec.History(runID); !equalStatuses(got, want) {
			t.Fatalf("%s moved through %v, want %v (failure %q)\n%s", a, got, want, h.rec.Failure(runID), h.log(t, runID))
		}
		log := h.log(t, runID)
		for _, v := range f.Good.Values() {
			if strings.Contains(log, v) {
				t.Fatalf("the %s log holds a credential value", a)
			}
		}
		parts := h.rec.Parts(runID)
		if len(parts) != 2 || parts[0].Status != jobs.StatusCompleted || parts[1].Status != jobs.StatusCompleted {
			t.Fatalf("%s progress %+v", a, parts)
		}
	}
	if s := h.rec.Deployment(plan.Parts[1].Deployment.ID); s != jobs.DeploymentDestroyed {
		t.Fatalf("after destroy the deployment is %q", s)
	}
}

func splitPreview(t *testing.T, f Fixture) {
	c := f.New()
	pv, ok := c.(connectors.Previewer)
	if !ok {
		t.Skip("the connector has no preview")
	}
	plan := splitPlan(t, c, f)
	const secret = "contract-secret/0123456789ABCDEF"
	d := plan.Parts[1].Deployment
	x := connectors.Cross{Exports: d.Cross.Exports}
	for _, imp := range d.Cross.Imports {
		imp.Values = map[string]connectors.Value{connectors.ValueAddress: {Text: "203.0.113.9"}, connectors.ValueSecretAccessKey: {Text: secret, Sensitive: true}}
		x.Imports = append(x.Imports, imp)
	}
	d.Cross = &x
	files, err := pv.Preview(context.Background(), f.Cross.Graph, d)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if strings.Contains(string(body), secret) || strings.Contains(string(body), "203.0.113.9") {
			t.Fatalf("%s holds a value another cloud handed the VM", name)
		}
	}
}

func manifest(t *testing.T, f Fixture) {
	c := f.New()
	m := c.Manifest()
	if m.Connector != c.ID() {
		t.Fatalf("manifest names %q, connector is %q", m.Connector, c.ID())
	}
	if err := m.Check(); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if !m.AllowsRegion(f.Region) {
		t.Fatalf("fixture region %q is not in the manifest", f.Region)
	}
	if err := connectors.NewRegistry().Register(c); err != nil {
		t.Fatalf("register: %v", err)
	}
	// One diagram must pass every connector, so a standard block offered as-is
	// declares exactly the portable fields B's core owns.
	for _, block := range m.Blocks {
		standard := connectors.StandardFields(block.Kind)
		if standard == nil || block.Treatment != connectors.TreatmentAsIs {
			continue
		}
		if !reflect.DeepEqual(block.Fields, standard) {
			t.Fatalf("block %q declares fields %+v; an as-is standard block declares connectors.StandardFields: %+v", block.Kind, block.Fields, standard)
		}
	}
}

// display checks that the cloud has a short name and a logo, that every
// block it builds names its service and says what it is for, and that every
// edge says what it means and why it helps. Check refuses a malformed
// display; this refuses a missing one.
func display(t *testing.T, f Fixture) {
	m := f.New().Manifest()
	if m.Display.ShortName == "" || m.Display.Logo == "" {
		t.Fatalf("manifest display %+v needs a short name and a logo key", m.Display)
	}
	for _, b := range m.Blocks {
		if b.Treatment == connectors.TreatmentRemoved {
			continue
		}
		if b.Display == nil || b.Display.Name == "" {
			t.Fatalf("block %q is built (%s) but has no display name", b.Kind, b.Treatment)
		}
		if b.Summary == "" {
			t.Fatalf("block %q is built (%s) but has no summary", b.Kind, b.Treatment)
		}
	}
	for _, e := range m.Edges {
		if e.Meaning == "" || e.Why == "" {
			t.Fatalf("edge %s->%s needs a meaning and a why", e.Source, e.Target)
		}
	}
}

// cloudsDir is the frontend folder the display keys resolve in, found from
// this file's place in the repository.
func cloudsDir() (string, bool) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", false
	}
	dir := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "frontend", "clutter", "public", "clouds")
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", false
	}
	return dir, true
}

// iconFiles checks that the logo and every icon key name a file in
// frontend/clutter/public/clouds/<connector>/, so a manifest cannot point
// the UI at an icon that does not exist.
func iconFiles(t *testing.T, f Fixture) {
	dir, ok := cloudsDir()
	if !ok {
		t.Skip("frontend/clutter/public/clouds is not beside this checkout")
	}
	m := f.New().Manifest()
	keys := []string{m.Display.Logo}
	for _, b := range m.Blocks {
		if b.Display != nil {
			keys = append(keys, b.Display.Icon)
		}
	}
	for _, k := range keys {
		if k == "" {
			continue
		}
		p := filepath.Join(dir, m.Connector, k+".svg")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("icon key %q has no file at %s", k, p)
		}
	}
}

func credentialSpec(t *testing.T, f Fixture) {
	spec := f.New().Credentials()
	if err := spec.Check(); err != nil {
		t.Fatalf("credential spec: %v", err)
	}
	if err := spec.CheckSupplied(f.Good); err != nil {
		t.Fatalf("good credentials do not fit the spec: %v", err)
	}
	for _, fld := range spec.Fields {
		if fld.Required {
			if err := spec.CheckSupplied(map[string]string{}); err == nil {
				t.Fatalf("empty credentials pass although %q is required", fld.Name)
			}
			break
		}
	}
	if err := spec.CheckSupplied(map[string]string{"contract_unknown_field": "x"}); err == nil {
		t.Fatal("an unknown credential field is accepted")
	}
}

func verify(t *testing.T, f Fixture) {
	c := f.New()
	ctx := context.Background()
	conn := connectors.Connection{ID: uuid.NewString(), ConnectorID: c.ID(), Region: f.Region}

	conn.Credentials = f.Good
	if err := c.VerifyConnection(ctx, conn); err != nil {
		t.Fatalf("good credentials refused: %v", err)
	}
	for name, creds := range map[string]connectors.Secrets{"bad": f.Bad, "empty": {}} {
		conn.Credentials = creds
		err := c.VerifyConnection(ctx, conn)
		if err == nil {
			t.Fatalf("%s credentials accepted", name)
		}
		for _, v := range creds.Values() {
			if strings.Contains(err.Error(), v) {
				t.Fatalf("the refusal of %s credentials repeats a credential value", name)
			}
		}
	}
}

// validate drives the checks B's core runs before a run (the capability check
// and the connector's Validate, capability.CheckConnector): an unsupported
// block is refused, an excluded one is not, and Validate never repeats a
// problem only the capability check reports.
func validate(t *testing.T, f Fixture) {
	c := f.New()
	m := c.Manifest()
	d := deployment(c, f)
	if ps := capability.CheckConnector(c, f.Graph, d).Problems; len(ps) > 0 {
		t.Fatalf("fixture graph refused: %+v", ps)
	}

	kind := connectors.BlockKind("contract_unknown_block")
	for _, b := range m.Blocks {
		if b.Treatment == connectors.TreatmentRemoved {
			kind = b.Kind
			break
		}
	}
	const nodeID = "contract-unsupported"
	g := cloneGraph(f.Graph)
	g.Blocks[nodeID] = connectors.Block{ID: nodeID, Kind: kind, Name: "unsupported", Fields: map[string]any{}}

	if ps := capability.CheckConnector(c, g, d).Problems; !HasProblem(ps, nodeID, capability.CodeUnsupportedBlock) {
		t.Fatalf("a %q block is not refused: %+v", kind, ps)
	}
	for _, p := range c.Validate(g, d) {
		switch p.Code {
		case capability.CodeUnsupportedBlock, capability.CodeUnsupportedEdge, capability.CodeInvalidName, capability.CodeUnknownField:
			t.Fatalf("Validate repeats the capability check: %+v", p)
		}
	}

	d.Exclusions = []string{nodeID}
	if ps := capability.CheckConnector(c, g, d).Problems; len(ps) > 0 {
		t.Fatalf("the excluded block is still refused: %+v", ps)
	}
}

func lifecycle(t *testing.T, f Fixture) {
	c := f.New()
	h := newHarness(t, c, f.Good)
	d := deployment(c, f)

	for _, a := range []connectors.Action{connectors.ActionPlan, connectors.ActionApply, connectors.ActionDestroy} {
		runID := h.run(t, a, f.Graph, d, f.Timeout)
		got := h.rec.History(runID)
		want := []jobs.Status{jobs.StatusQueued, jobs.StatusRunning, jobs.StatusCompleted}
		if !equalStatuses(got, want) {
			t.Fatalf("%s moved through %v, want %v (failure %q)", a, got, want, h.rec.Failure(runID))
		}
		log := h.log(t, runID)
		if strings.TrimSpace(log) == "" {
			t.Fatalf("%s left an empty log", a)
		}
		for _, v := range f.Good.Values() {
			if strings.Contains(log, v) {
				t.Fatalf("the %s log holds a credential value", a)
			}
		}
		want2 := map[connectors.Action]jobs.DeploymentStatus{connectors.ActionApply: jobs.DeploymentLive, connectors.ActionDestroy: jobs.DeploymentDestroyed}
		if ds, ok := want2[a]; ok && h.rec.Deployment(d.ID) != ds {
			t.Fatalf("after %s the deployment is %q, want %q", a, h.rec.Deployment(d.ID), ds)
		}
	}
}

func refusedRun(t *testing.T, f Fixture) {
	c := f.New()
	h := newHarness(t, c, f.Bad)
	d := deployment(c, f)
	runID := h.run(t, connectors.ActionPlan, f.Graph, d, f.Timeout)
	got := h.rec.History(runID)
	want := []jobs.Status{jobs.StatusQueued, jobs.StatusRunning, jobs.StatusFailed}
	if !equalStatuses(got, want) {
		t.Fatalf("plan with refused credentials moved through %v, want %v", got, want)
	}
	log := h.log(t, runID)
	for _, v := range f.Bad.Values() {
		if strings.Contains(log, v) || strings.Contains(h.rec.Failure(runID), v) {
			t.Fatal("the failed run's log or failure holds a credential value")
		}
	}
}

func deployment(c connectors.Connector, f Fixture) connectors.Deployment {
	id := uuid.NewString()
	return connectors.Deployment{
		ID:         id,
		Prefix:     connectors.PrefixFor(id),
		Connection: connectors.Connection{ID: uuid.NewString(), ConnectorID: c.ID(), Region: f.Region},
		Settings:   f.Settings,
	}
}

func cloneGraph(g connectors.Graph) connectors.Graph {
	out := connectors.Graph{Blocks: make(map[string]connectors.Block, len(g.Blocks)+1), Edges: append([]connectors.Edge(nil), g.Edges...)}
	for k, v := range g.Blocks {
		out.Blocks[k] = v
	}
	return out
}

func equalStatuses(a, b []jobs.Status) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type harness struct {
	rec    *Recorder
	logs   *logstore.Memory
	runner *jobs.Runner
}

func newHarness(t *testing.T, c connectors.Connector, creds connectors.Secrets) *harness {
	reg := connectors.NewRegistry()
	if err := reg.Register(c); err != nil {
		t.Fatalf("register: %v", err)
	}
	h := &harness{rec: NewRecorder(), logs: logstore.NewMemory()}
	h.runner = jobs.New(jobs.Config{
		Registry: reg,
		Store:    h.rec,
		Broker:   StaticBroker{Secrets: creds},
		Logs:     h.logs,
		Workers:  1,
	})
	h.runner.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.runner.Stop(ctx); err != nil {
			t.Errorf("stop: %v", err)
		}
	})
	return h
}

func (h *harness) run(t *testing.T, a connectors.Action, g connectors.Graph, d connectors.Deployment, timeout time.Duration) string {
	t.Helper()
	runID := uuid.NewString()
	h.rec.Queued(runID)
	err := h.runner.Enqueue(jobs.Job{RunID: runID, Action: a, Graph: g, Deployment: d, SecretRef: "contract", LogKey: logstore.KeyForRun(runID)})
	if err != nil {
		t.Fatalf("enqueue %s: %v", a, err)
	}
	if !h.rec.Wait(runID, timeout) {
		t.Fatalf("%s did not finish within %s", a, timeout)
	}
	return runID
}

func (h *harness) log(t *testing.T, runID string) string {
	t.Helper()
	r, err := h.logs.Open(logstore.KeyForRun(runID))
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	return string(b)
}

// StaticBroker hands every run the same credentials.
type StaticBroker struct{ Secrets connectors.Secrets }

// Credentials implements jobs.Broker.
func (b StaticBroker) Credentials(context.Context, string) (connectors.Secrets, error) {
	if b.Secrets == nil {
		return nil, errors.New("no credentials")
	}
	return b.Secrets, nil
}

// Recorder is an in-memory jobs.Store that keeps every state a run passes
// through, and each cloud's progress of a split run (jobs.PartRecorder).
type Recorder struct {
	mu          sync.Mutex
	history     map[string][]jobs.Status
	failures    map[string]string
	deployments map[string]jobs.DeploymentStatus
	done        map[string]chan struct{}
	parts       map[string][]jobs.PartProgress
}

// NewRecorder returns an empty Recorder.
func NewRecorder() *Recorder {
	return &Recorder{
		history:     map[string][]jobs.Status{},
		failures:    map[string]string{},
		deployments: map[string]jobs.DeploymentStatus{},
		done:        map[string]chan struct{}{},
		parts:       map[string][]jobs.PartProgress{},
	}
}

// RunParts implements jobs.PartRecorder.
func (r *Recorder) RunParts(_ context.Context, runID string, parts []jobs.PartProgress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parts[runID] = append([]jobs.PartProgress(nil), parts...)
	return nil
}

// Parts returns a split run's last recorded progress.
func (r *Recorder) Parts(runID string) []jobs.PartProgress {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]jobs.PartProgress(nil), r.parts[runID]...)
}

// Queued records a run as queued, as the API does before it enqueues.
func (r *Recorder) Queued(runID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history[runID] = []jobs.Status{jobs.StatusQueued}
	r.done[runID] = make(chan struct{})
}

// RunStarted implements jobs.Store.
func (r *Recorder) RunStarted(_ context.Context, runID string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history[runID] = append(r.history[runID], jobs.StatusRunning)
	return nil
}

// RunFinished implements jobs.Store.
func (r *Recorder) RunFinished(_ context.Context, runID string, s jobs.Status, failure string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history[runID] = append(r.history[runID], s)
	r.failures[runID] = failure
	if ch, ok := r.done[runID]; ok {
		close(ch)
		delete(r.done, runID)
	}
	return nil
}

// SetDeploymentStatus implements jobs.Store.
func (r *Recorder) SetDeploymentStatus(_ context.Context, id string, s jobs.DeploymentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deployments[id] = s
	return nil
}

// Wait waits for a queued run to finish.
func (r *Recorder) Wait(runID string, timeout time.Duration) bool {
	r.mu.Lock()
	ch, ok := r.done[runID]
	r.mu.Unlock()
	if !ok {
		return true
	}
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

// History returns the states a run passed through.
func (r *Recorder) History(runID string) []jobs.Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]jobs.Status(nil), r.history[runID]...)
}

// Failure returns a run's recorded failure.
func (r *Recorder) Failure(runID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.failures[runID]
}

// Deployment returns a deployment's last recorded status.
func (r *Recorder) Deployment(id string) jobs.DeploymentStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.deployments[id]
}
