// Package runtask starts one run's commands in their own process, with an
// environment built from nothing but what that run needs (ADR-017).
//
// Local is the laptop implementation: a child process per command. ECS is the
// cloud implementation: one short-lived Fargate task per run, which also runs
// every command of the run (RunLauncher), with state in S3.
package runtask

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"time"
)

// Task is one command of one run.
type Task struct {
	// Dir is the working directory.
	Dir string
	// Path is the absolute path of the program; Args are its arguments.
	Path string
	Args []string
	// Env is the whole environment. Nothing from B's own environment is added.
	Env map[string]string
	// Output receives stdout, and stderr unless Stderr is set.
	Output io.Writer
	// Stderr, when set, receives stderr instead of Output.
	Stderr io.Writer
}

// Launcher runs a task to completion.
type Launcher interface {
	Launch(ctx context.Context, t Task) error
}

// Run is every Terraform command of one run, for a launcher that runs them
// all in one task somewhere else.
type Run struct {
	RunID        string
	DeploymentID string
	// Dir is the prepared run folder: the rendered files and the backend file
	// StateBackend returned. Only its regular files are sent.
	Dir string
	// Steps are Terraform's arguments for each command, in order. The first
	// failure stops the run. Only CheckSteps' flags are allowed.
	Steps [][]string
	// Env is the whole environment of every command, as for Task.
	Env map[string]string
	// PublicHosts are hosts the run task resolves before the first command;
	// one that resolves to an address netguard.Blocked refuses (private ones
	// included) stops the run (terraform.Job.PublicHosts).
	PublicHosts []string
	// Output receives the commands' stdout and stderr, except the stdout of
	// an "output" step.
	Output io.Writer
	// Outputs receives the stdout of the run's "output" step (Terraform's
	// outputs as JSON, sensitive values included), which never reaches
	// Output. Required when Steps holds an output step.
	Outputs io.Writer
}

// RunLauncher is a Launcher that runs a whole run in one isolated task and
// keeps state outside B. The Terraform runner asks it for the backend block
// to write, so neither the rendered files nor the run decide where state goes.
type RunLauncher interface {
	Launcher
	// StateBackend returns the whole file that configures the deployment's
	// state backend.
	StateBackend(deploymentID string) ([]byte, error)
	LaunchRun(ctx context.Context, r Run) error
}

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// EnvList turns env into the sorted NAME=value list a process takes. It
// refuses a name that is not a plain identifier. The result is never nil, so
// a process given it never inherits its parent's environment.
func EnvList(env map[string]string) ([]string, error) {
	out := make([]string, 0, len(env))
	for k, v := range env {
		if !envNamePattern.MatchString(k) {
			return nil, fmt.Errorf("environment name %q is not valid", k)
		}
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out, nil
}

// Local runs each task as a child process of B.
type Local struct {
	// Grace is how long a cancelled task has to stop after an interrupt
	// before it is killed. Zero means 30 seconds.
	Grace time.Duration
}

// Launch implements Launcher.
func (l Local) Launch(ctx context.Context, t Task) error {
	if t.Path == "" || t.Dir == "" {
		return errors.New("a task needs a program and a working directory")
	}
	env, err := EnvList(t.Env)
	if err != nil {
		return err
	}
	out := t.Output
	if out == nil {
		out = io.Discard
	}
	cmd := exec.CommandContext(ctx, t.Path, t.Args...)
	cmd.Dir = t.Dir
	// Never nil: exec.Cmd with a nil Env inherits B's environment, which holds
	// the database string and the credential key.
	cmd.Env = env
	cmd.Stdin = nil
	cmd.Stdout = out
	cmd.Stderr = out
	if t.Stderr != nil {
		cmd.Stderr = t.Stderr
	}
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = l.Grace
	if cmd.WaitDelay == 0 {
		cmd.WaitDelay = 30 * time.Second
	}
	return cmd.Run()
}
