// Package jobs runs plan, apply and destroy jobs in-process. A run moves from
// queued to running to completed or failed; its log goes to a log store with
// the run's credential values removed.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/split"
)

// Status is a run's state.
type Status string

// The run states.
const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// DeploymentStatus is a deployment's state.
type DeploymentStatus string

// The deployment states. Live and failed deployments may hold resources in the
// user's cloud, so they cannot be deleted until destroyed.
const (
	DeploymentNew       DeploymentStatus = "new"
	DeploymentLive      DeploymentStatus = "live"
	DeploymentFailed    DeploymentStatus = "failed"
	DeploymentDestroyed DeploymentStatus = "destroyed"
)

// Errors from Enqueue.
var (
	ErrQueueFull = errors.New("the run queue is full")
	ErrStopped   = errors.New("the job runner is stopped")
)

// Failure messages a user sees. Details go to the run's log, except a
// launcher's, which go to the service log.
const (
	failGeneric     = "The run failed. Its log has the details."
	failNoConnector = "This cloud is not available on this Clutter server."
	failCredentials = "Clutter could not read this connection's credentials."
	failVerify      = "The connection's credentials did not work. Check them on the Connections page."
	failValidate    = connectors.MsgCannotBuild
	failInterrupted = "The run stopped because the Clutter server stopped."
	// failPlatform replaces a launcher error in the user's log and the run's
	// failure. Those errors name Clutter's own buckets, roles and ECS reasons,
	// so their detail goes to the service log only.
	failPlatform     = "Clutter could not start this run. Try again; if it keeps failing, contact support."
	maxFailureLength = 300
)

// Store records run and deployment states.
type Store interface {
	RunStarted(ctx context.Context, runID string, at time.Time) error
	RunFinished(ctx context.Context, runID string, status Status, failure string, at time.Time) error
	SetDeploymentStatus(ctx context.Context, deploymentID string, status DeploymentStatus) error
}

// Broker hands a run its connection's credentials, and nothing else.
type Broker interface {
	Credentials(ctx context.Context, secretRef string) (connectors.Secrets, error)
}

// Job is one queued run.
type Job struct {
	RunID      string
	Action     connectors.Action
	Graph      connectors.Graph
	Deployment connectors.Deployment
	SecretRef  string
	LogKey     string
	// Parts, when set, make the run a split run (split.go): one step per
	// cloud, in this order, each with only its own cloud's credentials.
	// Deployment and SecretRef are then the primary's and are not used.
	Parts []split.Part
	// Prior is the deployment's status when the run was queued. A split apply
	// rolls back only a deployment that held nothing before the run.
	Prior DeploymentStatus
}

// Config builds a Runner.
type Config struct {
	Registry   *connectors.Registry
	Store      Store
	Broker     Broker
	Logs       logstore.Store
	Logger     *slog.Logger
	Workers    int
	QueueSize  int
	RunTimeout time.Duration
	Now        func() time.Time
}

// Runner is an in-process queue with a fixed number of workers.
type Runner struct {
	cfg     Config
	queue   chan Job
	quit    chan struct{}
	runCtx  context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	stopped bool
}

// New returns a Runner. Call Start to begin work.
func New(cfg Config) *Runner {
	if cfg.Workers <= 0 {
		cfg.Workers = 2
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}
	if cfg.RunTimeout <= 0 {
		cfg.RunTimeout = time.Hour
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Runner{cfg: cfg, queue: make(chan Job, cfg.QueueSize), quit: make(chan struct{}), runCtx: ctx, cancel: cancel}
}

// Start starts the workers.
func (r *Runner) Start() {
	for i := 0; i < r.cfg.Workers; i++ {
		r.wg.Add(1)
		go r.work()
	}
}

// Enqueue adds a job. The run must already be recorded as queued.
func (r *Runner) Enqueue(j Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return ErrStopped
	}
	select {
	case r.queue <- j:
		return nil
	default:
		return ErrQueueFull
	}
}

// Stop takes no new jobs and waits for running ones until ctx ends, then
// cancels them and waits for them to record their failure. Jobs still in the
// queue stay queued; FailInterrupted fails them at the next start.
func (r *Runner) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.stopped {
		r.stopped = true
		close(r.quit)
	}
	r.mu.Unlock()

	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
		r.cancel()
		return nil
	case <-ctx.Done():
		r.cancel()
		<-done
		return ctx.Err()
	}
}

func (r *Runner) work() {
	defer r.wg.Done()
	for {
		select {
		case <-r.quit:
			return
		default:
		}
		select {
		case <-r.quit:
			return
		case j := <-r.queue:
			r.run(j)
		}
	}
}

// run executes one job and records its outcome. It never panics out.
func (r *Runner) run(j Job) {
	log := r.cfg.Logger.With("run_id", j.RunID, "deployment_id", j.Deployment.ID, "action", string(j.Action))
	ctx, cancel := context.WithTimeout(r.runCtx, r.cfg.RunTimeout)
	defer cancel()
	// Records go through a context that outlives a cancelled run.
	recordCtx := context.WithoutCancel(ctx)

	if err := r.cfg.Store.RunStarted(recordCtx, j.RunID, r.cfg.Now()); err != nil {
		log.Error("could not mark the run running", "err", err)
		return
	}

	failure, err := r.execute(ctx, j, log)
	status := StatusCompleted
	if err != nil {
		status = StatusFailed
	}
	if err := r.cfg.Store.RunFinished(recordCtx, j.RunID, status, failure, r.cfg.Now()); err != nil {
		log.Error("could not record the run's end", "err", err)
	}
	// A run that failed before Terraform touched the cloud changed nothing, so
	// the deployment keeps its status. A split run that rolled back has
	// recorded the deployment's status itself.
	if errors.Is(err, connectors.ErrNothingBuilt) || errors.Is(err, errStatusRecorded) {
		return
	}
	if ds, ok := deploymentAfter(j.Action, status); ok {
		if err := r.cfg.Store.SetDeploymentStatus(recordCtx, j.Deployment.ID, ds); err != nil {
			log.Error("could not record the deployment's status", "err", err)
		}
	}
}

// deploymentAfter is a deployment's status after a run. A failed apply or
// destroy may leave resources behind, so the deployment counts as failed.
func deploymentAfter(a connectors.Action, s Status) (DeploymentStatus, bool) {
	switch {
	case a == connectors.ActionPlan:
		return "", false
	case s == StatusFailed:
		return DeploymentFailed, true
	case a == connectors.ActionApply:
		return DeploymentLive, true
	case a == connectors.ActionDestroy:
		return DeploymentDestroyed, true
	}
	return "", false
}

// execute runs the job and returns the user-facing failure on error.
func (r *Runner) execute(ctx context.Context, j Job, log *slog.Logger) (failure string, err error) {
	sink, err := r.cfg.Logs.Create(j.LogKey)
	if err != nil {
		log.Error("could not open the run's log", "err", err)
		return failGeneric, err
	}
	rw := newRedactWriter(sink)
	defer func() {
		if ferr := rw.flush(); ferr != nil {
			log.Error("could not write the run's log", "err", ferr)
		}
		if cerr := sink.Close(); cerr != nil {
			log.Error("could not close the run's log", "err", cerr)
		}
	}()
	defer func() {
		if p := recover(); p != nil {
			log.Error("the connector panicked", "panic", rw.redact(fmt.Sprint(p)))
			fmt.Fprintln(rw, "The run stopped on an internal error.")
			failure, err = failGeneric, fmt.Errorf("connector panic")
		}
	}()

	if ctx.Err() != nil {
		fmt.Fprintln(rw, "The run was cancelled before it started.")
		return failInterrupted, fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, ctx.Err())
	}
	if len(j.Parts) > 0 {
		return r.executeSplit(ctx, j, rw, log)
	}

	c, ok := r.cfg.Registry.Get(j.Deployment.Connection.ConnectorID)
	if !ok {
		fmt.Fprintln(rw, failNoConnector)
		return failNoConnector, fmt.Errorf("%w: connector not registered", connectors.ErrNothingBuilt)
	}

	secrets, err := r.cfg.Broker.Credentials(ctx, j.SecretRef)
	if err != nil {
		log.Error("could not fetch the run's credentials", "err", err)
		fmt.Fprintln(rw, failCredentials)
		return failCredentials, fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
	}
	rw.set(secrets.Values())
	d := j.Deployment
	d.Connection.Credentials = secrets

	fail := func(msg string, err error) (string, error) {
		detail := rw.redact(err.Error())
		if platformError(err) {
			log.Error("the run did not start", "err", detail)
			fmt.Fprintln(rw, failPlatform)
			return failPlatform, err
		}
		fmt.Fprintf(rw, "Error: %s\n", detail)
		log.Warn("run failed", "err", detail)
		return truncate(rw.redact(connectors.UserMessage(err, msg))), err
	}

	fmt.Fprintf(rw, "Run %s: %s on deployment %s.\n", j.RunID, j.Action, d.ID)
	if failure, err := preflight(ctx, rw, log, "", c, j.Graph, d, j.Action); err != nil {
		return failure, err
	}

	run := connectors.Run{ID: j.RunID, Action: j.Action, Log: rw}
	switch j.Action {
	case connectors.ActionPlan:
		res, err := c.Plan(ctx, run, j.Graph, d)
		if err != nil {
			return fail(failGeneric, err)
		}
		fmt.Fprintf(rw, "Plan: %d to add, %d to change, %d to destroy.\n", res.Add, res.Change, res.Destroy)
	case connectors.ActionApply:
		res, err := c.Apply(ctx, run, j.Graph, d)
		if err != nil {
			return fail(failGeneric, err)
		}
		fmt.Fprintf(rw, "Apply complete: %s.\n", blocksChanged(len(res.Resources)))
		writeResults(rw, res.Results)
	case connectors.ActionDestroy:
		if _, err := c.Destroy(ctx, run, d); err != nil {
			return fail(failGeneric, err)
		}
		fmt.Fprintln(rw, "Destroy complete.")
	default:
		return fail(failGeneric, fmt.Errorf("unknown action %q", j.Action))
	}
	return "", nil
}

// blocksChanged names what B's own apply line counts: the blocks a connector
// reports it built or changed in this run (RunResult.Resources, one per node).
// Terraform's own summary line, earlier in the log, counts its resources,
// which are more.
func blocksChanged(n int) string {
	switch n {
	case 0:
		return "everything is up to date"
	case 1:
		return "1 block built or changed"
	}
	return fmt.Sprintf("%d blocks built or changed", n)
}

// ResultLinePrefix starts each line of an apply's results in the run's log:
// "Result: <node id> <name> <value>", one per result, after the apply's
// "Apply complete" line. The deploy panel reads them back for "What was
// built" (frontend deployments/results.ts). A node id and a result name hold
// no space, and connectors.ReadResults keeps only values CheckCrossValue
// accepts, which hold none either.
const ResultLinePrefix = "Result: "

// writeResults writes an apply's results to the run's log, by node id, each
// block's in the order the connector gave them.
func writeResults(w io.Writer, results map[string][]connectors.Result) {
	ids := make([]string, 0, len(results))
	for id := range results {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, r := range results[id] {
			if strings.ContainsAny(id+r.Name, " \n") || connectors.CheckCrossValue(r.Value) != nil {
				continue
			}
			fmt.Fprintf(w, "%s%s %s %s\n", ResultLinePrefix, id, r.Name, r.Value)
		}
	}
}

// preflight is what every cloud of a run passes before any cloud changes: its
// connection verifies and, unless the run destroys, the graph passes the
// capability check, the connector's own checks and, on a part of a split
// deployment, its cross-cloud checks. name labels the log lines and the
// failure of one cloud of a split run; it is "" for a run on one cloud. On
// failure it returns the user's failure and an error that wraps
// connectors.ErrNothingBuilt.
func preflight(ctx context.Context, rw *redactWriter, log *slog.Logger, name string, c connectors.Connector, g connectors.Graph, d connectors.Deployment, a connectors.Action) (string, error) {
	label := func(s string) string {
		if name == "" {
			return s
		}
		return name + ": " + s
	}
	if err := c.VerifyConnection(ctx, d.Connection); err != nil {
		detail := rw.redact(err.Error())
		fmt.Fprintln(rw, label("Error: "+detail))
		log.Warn("the connection did not verify", "cloud", name, "err", detail)
		return truncate(rw.redact(label(connectors.UserMessage(err, failVerify)))), fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
	}
	if a == connectors.ActionDestroy {
		return "", nil
	}
	problems := capability.CheckConnector(c, g, d).Problems
	if cc, ok := c.(connectors.CrossCloud); ok && d.Cross != nil {
		problems = capability.Merge(problems, cc.CrossCheck(g, d))
	}
	if len(problems) == 0 {
		return "", nil
	}
	for _, p := range problems {
		fmt.Fprintln(rw, label(fmt.Sprintf("Problem: %s %s%s: %s", p.Code, p.NodeID, edgeText(p), p.Message)))
	}
	return failValidate, fmt.Errorf("%w: validation failed", connectors.ErrNothingBuilt)
}

// platformError reports whether err is the launcher's own failure to start a
// run (runtask.ErrNotStarted), which holds Clutter's platform details. A
// connector's UserError in the chain is meant for the user and wins.
func platformError(err error) bool {
	var ue *connectors.UserError
	return errors.Is(err, runtask.ErrNotStarted) && !errors.As(err, &ue)
}

func edgeText(p connectors.Problem) string {
	if p.EdgeSource == "" {
		return ""
	}
	return p.EdgeSource + "->" + p.EdgeTarget
}

func truncate(s string) string {
	s = strings.ToValidUTF8(strings.TrimSpace(s), "?")
	if r := []rune(s); len(r) > maxFailureLength {
		return string(r[:maxFailureLength])
	}
	return s
}

// FailInterruptedMessage is what FailInterrupted records on runs a restart cut short.
const FailInterruptedMessage = failInterrupted

var _ io.Writer = (*redactWriter)(nil)
