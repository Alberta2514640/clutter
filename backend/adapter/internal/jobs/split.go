package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/split"
)

// errStatusRecorded marks a split run that recorded its deployment's status
// itself (after a roll-back), so run does not overwrite it.
var errStatusRecorded = errors.New("the deployment's status is recorded")

// The steps of a split run, as PartProgress names them.
const (
	stepRollback = "rollback"
)

// Part step states beyond the run states.
const (
	StatusSkipped    Status = "skipped"
	StatusRolledBack Status = "rolled_back"
)

// PartProgress is where one cloud of a split run stands.
type PartProgress struct {
	ConnectionID string `json:"connection_id"`
	Name         string `json:"name"`
	Step         string `json:"step"`
	Status       Status `json:"status"`
}

// PartRecorder is optional for a Store. A store that implements it keeps each
// cloud's progress of a split run, so the deploy panel can show it.
type PartRecorder interface {
	RunParts(ctx context.Context, runID string, parts []PartProgress) error
}

// splitRun is one split run in progress.
type splitRun struct {
	r        *Runner
	j        Job
	rw       *redactWriter
	log      *slog.Logger
	progress []PartProgress
	// outputs are what the parts applied so far returned, by edge key.
	outputs connectors.Outputs
}

func (s *splitRun) record(i int, step string, st Status) {
	s.progress[i].Step, s.progress[i].Status = step, st
	pr, ok := s.r.cfg.Store.(PartRecorder)
	if !ok {
		return
	}
	cp := append([]PartProgress(nil), s.progress...)
	if err := pr.RunParts(context.Background(), s.j.RunID, cp); err != nil {
		s.log.Error("could not record a split run's progress", "err", err)
	}
}

// cloudName is how the log and the failure name a part: its connection's
// name and its cloud's title.
func cloudName(p split.Part, c connectors.Connector) string {
	return fmt.Sprintf("%s (%s)", p.Name, c.Manifest().Title)
}

// stepDeployment is a part's deployment for one step: its imports carry the
// values the parts before it returned, and its connection the credentials the
// broker hands this step only. No other cloud's credential is in it.
func (s *splitRun) stepDeployment(p split.Part, secrets connectors.Secrets) connectors.Deployment {
	d := p.Deployment
	d.Connection.Credentials = secrets
	x := connectors.Cross{Exports: append([]connectors.CrossEdge(nil), p.Deployment.Cross.Exports...)}
	for _, imp := range p.Deployment.Cross.Imports {
		imp.Values = map[string]connectors.Value{}
		for name, v := range s.outputs[imp.Key] {
			imp.Values[name] = v
		}
		x.Imports = append(x.Imports, imp)
	}
	d.Cross = &x
	return d
}

// credentials asks the broker for one part's credentials and adds them to
// what the log removes.
func (s *splitRun) credentials(ctx context.Context, p split.Part) (connectors.Secrets, error) {
	secrets, err := s.r.cfg.Broker.Credentials(ctx, p.SecretRef)
	if err != nil {
		return nil, err
	}
	s.rw.add(secrets.Values())
	return secrets, nil
}

// executeSplit runs a split deployment's run: every cloud is verified and
// checked first, then each cloud runs as its own step, in the parts' order
// for a plan or an apply and in reverse for a destroy, each with only its own
// credentials and in its own run task (through its connector's Terraform
// runner). Values one cloud returns reach the clouds after it.
func (r *Runner) executeSplit(ctx context.Context, j Job, rw *redactWriter, log *slog.Logger) (string, error) {
	s := &splitRun{r: r, j: j, rw: rw, log: log, outputs: connectors.Outputs{}}
	conns := make([]connectors.CrossCloud, len(j.Parts))
	plain := make([]connectors.Connector, len(j.Parts))
	for i, p := range j.Parts {
		s.progress = append(s.progress, PartProgress{ConnectionID: p.ConnectionID, Name: p.Name, Step: string(j.Action), Status: StatusQueued})
		c, ok := r.cfg.Registry.Get(p.Deployment.Connection.ConnectorID)
		if !ok {
			fmt.Fprintln(rw, failNoConnector)
			return failNoConnector, fmt.Errorf("%w: connector not registered", connectors.ErrNothingBuilt)
		}
		cc, ok := c.(connectors.CrossCloud)
		if !ok || p.Deployment.Cross == nil {
			fmt.Fprintln(rw, failValidate)
			return failValidate, fmt.Errorf("%w: a cloud cannot take part in a split", connectors.ErrNothingBuilt)
		}
		conns[i], plain[i] = cc, c
	}
	if pr, ok := r.cfg.Store.(PartRecorder); ok {
		if err := pr.RunParts(context.Background(), j.RunID, append([]PartProgress(nil), s.progress...)); err != nil {
			log.Error("could not record a split run's progress", "err", err)
		}
	}
	fmt.Fprintf(rw, "Run %s: %s on deployment %s across %d clouds.\n", j.RunID, j.Action, j.Deployment.ID, len(j.Parts))

	// Every cloud is verified, and checked, before any cloud changes.
	for i, p := range j.Parts {
		name := cloudName(p, plain[i])
		secrets, err := s.credentials(ctx, p)
		if err != nil {
			log.Error("could not fetch a cloud's credentials", "connection_id", p.ConnectionID, "err", err)
			fmt.Fprintf(rw, "%s: %s\n", name, failCredentials)
			return truncate(name + ": " + failCredentials), fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
		}
		if failure, err := preflight(ctx, rw, log, name, plain[i], j.Graph, s.stepDeployment(p, secrets), j.Action); err != nil {
			return failure, err
		}
	}

	switch j.Action {
	case connectors.ActionPlan:
		return s.plan(ctx, conns, plain)
	case connectors.ActionApply:
		return s.apply(ctx, conns, plain)
	case connectors.ActionDestroy:
		return s.destroy(ctx, plain)
	}
	return failGeneric, fmt.Errorf("unknown action %q", j.Action)
}

// stepRun is one part's step.
func (s *splitRun) stepRun(p split.Part, step string, a connectors.Action) connectors.Run {
	return connectors.Run{ID: split.StepRunID(s.j.RunID, p.ConnectionID, step), Action: a, Log: s.rw}
}

// keep adds a part's outputs to what later parts import, after removing their
// secrets from the log.
func (s *splitRun) keep(out connectors.Outputs) {
	s.rw.add(out.SecretValues())
	for k, vs := range out {
		if s.outputs[k] == nil {
			s.outputs[k] = map[string]connectors.Value{}
		}
		for n, v := range vs {
			s.outputs[k][n] = v
		}
	}
}

func (s *splitRun) plan(ctx context.Context, conns []connectors.CrossCloud, plain []connectors.Connector) (string, error) {
	var total connectors.PlanResult
	for i, p := range s.j.Parts {
		name := cloudName(p, plain[i])
		fmt.Fprintf(s.rw, "\n== %s: plan ==\n", name)
		s.record(i, string(connectors.ActionPlan), StatusRunning)
		secrets, err := s.credentials(ctx, p)
		if err != nil {
			s.record(i, string(connectors.ActionPlan), StatusFailed)
			return truncate(name + ": " + failCredentials), fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
		}
		res, out, err := conns[i].PlanCross(ctx, s.stepRun(p, string(connectors.ActionPlan), connectors.ActionPlan), s.j.Graph, s.stepDeployment(p, secrets))
		if err != nil {
			s.record(i, string(connectors.ActionPlan), StatusFailed)
			s.skipFrom(i+1, string(connectors.ActionPlan))
			return s.stepFailure(name, err)
		}
		s.keep(out)
		s.record(i, string(connectors.ActionPlan), StatusCompleted)
		fmt.Fprintf(s.rw, "%s: %d to add, %d to change, %d to destroy.\n", name, res.Add, res.Change, res.Destroy)
		total.Add, total.Change, total.Destroy = total.Add+res.Add, total.Change+res.Change, total.Destroy+res.Destroy
	}
	fmt.Fprintf(s.rw, "Plan: %d to add, %d to change, %d to destroy.\n", total.Add, total.Change, total.Destroy)
	return "", nil
}

func (s *splitRun) apply(ctx context.Context, conns []connectors.CrossCloud, plain []connectors.Connector) (string, error) {
	built := 0
	for i, p := range s.j.Parts {
		name := cloudName(p, plain[i])
		fmt.Fprintf(s.rw, "\n== %s: apply ==\n", name)
		s.record(i, string(connectors.ActionApply), StatusRunning)
		secrets, err := s.credentials(ctx, p)
		if err != nil {
			err = fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
		} else {
			d := s.stepDeployment(p, secrets)
			var res connectors.RunResult
			var out connectors.Outputs
			res, out, err = conns[i].ApplyCross(ctx, s.stepRun(p, string(connectors.ActionApply), connectors.ActionApply), s.j.Graph, d)
			if err == nil {
				err = s.complete(p, out)
			}
			if err == nil {
				s.keep(out)
				s.record(i, string(connectors.ActionApply), StatusCompleted)
				fmt.Fprintf(s.rw, "%s: apply complete, %s.\n", name, blocksChanged(len(res.Resources)))
				writeResults(s.rw, res.Results)
				built = i + 1
				continue
			}
		}
		s.record(i, string(connectors.ActionApply), StatusFailed)
		s.skipFrom(i+1, string(connectors.ActionApply))
		s.errorLine(name, err)
		if !errors.Is(err, connectors.ErrNothingBuilt) {
			built = i + 1
		}
		return s.rollback(ctx, plain, i, built, name, err)
	}
	return "", nil
}

// complete checks that an apply returned every value its exports promise.
func (s *splitRun) complete(p split.Part, out connectors.Outputs) error {
	for _, e := range p.Deployment.Cross.Exports {
		for _, name := range connectors.ExportValues(e.Class, s.j.Graph.Blocks[e.Target].Kind) {
			if out[e.Key][name].Text == "" {
				return &connectors.UserError{Message: fmt.Sprintf("The cloud did not return the %s that %s needs from %s.",
					strings.ReplaceAll(name, "_", " "), s.j.Graph.Blocks[e.Source].Name, s.j.Graph.Blocks[e.Target].Name)}
			}
		}
	}
	return nil
}

// rollback answers a failed apply of part failed. A deployment that held
// nothing before this run has the parts this run built (the first built ones)
// destroyed, newest first, so nothing is left half built. A deployment that
// was live is not rolled back: destroying would remove what ran before the run.
func (s *splitRun) rollback(ctx context.Context, plain []connectors.Connector, failed, built int, name string, cause error) (string, error) {
	first := s.j.Prior == DeploymentNew || s.j.Prior == DeploymentDestroyed || s.j.Prior == ""
	if built == 0 {
		// Nothing reached a cloud: the deployment keeps its status.
		return truncate(s.rw.redact(name + ": " + connectors.UserMessage(cause, failGeneric))), cause
	}
	if !first {
		msg := fmt.Sprintf("%s failed. The clouds before it were updated; Clutter does not undo an update of a live deployment. Fix the problem and apply again.", name)
		return truncate(s.rw.redact(msg)), cause
	}
	// The roll-back has its own deadline: the run's may be what ended it.
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.r.cfg.RunTimeout)
	defer cancel()
	var left []string
	for i := built - 1; i >= 0; i-- {
		p := s.j.Parts[i]
		pname := cloudName(p, plain[i])
		fmt.Fprintf(s.rw, "\n== %s: roll back ==\n", pname)
		s.record(i, stepRollback, StatusRunning)
		secrets, err := s.credentials(rctx, p)
		if err == nil {
			_, err = plain[i].Destroy(rctx, s.stepRun(p, stepRollback, connectors.ActionDestroy), s.stepDeployment(p, secrets))
		}
		if err != nil {
			s.record(i, stepRollback, StatusFailed)
			s.errorLine(pname+": roll-back", err)
			left = append(left, pname)
			continue
		}
		s.record(i, stepRollback, StatusRolledBack)
	}
	recordCtx := context.WithoutCancel(ctx)
	if len(left) > 0 {
		if err := s.r.cfg.Store.SetDeploymentStatus(recordCtx, s.j.Deployment.ID, DeploymentFailed); err != nil {
			s.log.Error("could not record the deployment's status", "err", err)
		}
		msg := fmt.Sprintf("%s failed, and rolling back %s failed too, so resources may remain. Destroy the deployment.", name, strings.Join(left, ", "))
		return truncate(s.rw.redact(msg)), fmt.Errorf("%w: %w", errStatusRecorded, cause)
	}
	prior := s.j.Prior
	if prior == "" {
		prior = DeploymentNew
	}
	if err := s.r.cfg.Store.SetDeploymentStatus(recordCtx, s.j.Deployment.ID, prior); err != nil {
		s.log.Error("could not record the deployment's status", "err", err)
	}
	msg := fmt.Sprintf("%s failed. Clutter rolled back what this run built on the other clouds, so nothing is left half built.", name)
	if failed == 0 {
		msg = fmt.Sprintf("%s failed. Clutter removed what this run built there, so nothing is left half built.", name)
	}
	return truncate(s.rw.redact(msg)), fmt.Errorf("%w: %w", errStatusRecorded, cause)
}

// destroy destroys every part, newest first. A part that fails does not stop
// the others.
func (s *splitRun) destroy(ctx context.Context, plain []connectors.Connector) (string, error) {
	var failed []string
	for i := len(s.j.Parts) - 1; i >= 0; i-- {
		p := s.j.Parts[i]
		name := cloudName(p, plain[i])
		fmt.Fprintf(s.rw, "\n== %s: destroy ==\n", name)
		s.record(i, string(connectors.ActionDestroy), StatusRunning)
		secrets, err := s.credentials(ctx, p)
		if err == nil {
			_, err = plain[i].Destroy(ctx, s.stepRun(p, string(connectors.ActionDestroy), connectors.ActionDestroy), s.stepDeployment(p, secrets))
		}
		if err != nil {
			s.record(i, string(connectors.ActionDestroy), StatusFailed)
			s.errorLine(name, err)
			failed = append(failed, name)
			continue
		}
		s.record(i, string(connectors.ActionDestroy), StatusCompleted)
	}
	if len(failed) > 0 {
		// Some cloud may still hold resources, so the deployment counts as
		// failed: the error does not say that nothing was built.
		msg := fmt.Sprintf("Destroying %s failed; Clutter destroyed the other clouds. Destroy the deployment again.", strings.Join(failed, ", "))
		return truncate(s.rw.redact(msg)), errors.New("destroy failed on a cloud")
	}
	return "", nil
}

// errorLine writes a step's error to the run's log. A launcher's own error
// names Clutter's buckets and roles, so its detail goes to the service log.
func (s *splitRun) errorLine(name string, err error) {
	if platformError(err) {
		s.log.Error("a split run's step did not start", "cloud", name, "err", s.rw.redact(err.Error()))
		fmt.Fprintf(s.rw, "%s: %s\n", name, failPlatform)
		return
	}
	fmt.Fprintf(s.rw, "%s: Error: %s\n", name, s.rw.redact(err.Error()))
}

func (s *splitRun) skipFrom(i int, step string) {
	for k := i; k < len(s.progress); k++ {
		s.record(k, step, StatusSkipped)
	}
}

// stepFailure is a failed plan's failure: the cloud's name and its message.
func (s *splitRun) stepFailure(name string, err error) (string, error) {
	s.errorLine(name, err)
	if platformError(err) {
		return failPlatform, err
	}
	return truncate(s.rw.redact(name + ": " + connectors.UserMessage(err, failGeneric))), err
}
