package runtask

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask/runtasktest"
	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func TestCheckStepsAllowsOnlyTheJSONOutputStep(t *testing.T) {
	if err := CheckSteps([][]string{{OutputStep, "-json", "-no-color"}}); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][][]string{{{OutputStep, "-state=/x"}}, {{"plan", "-json"}}, {{OutputStep, "-raw"}}} {
		if CheckSteps(bad) == nil {
			t.Errorf("%v accepted", bad)
		}
	}
}

func TestLaunchRunHandsBackOutputsOutsideTheLog(t *testing.T) {
	var m *runtasktest.S3
	keys := KeysFor(tRun)
	e, _, m, _, r := newTestECS(t, func() ecstypes.Task {
		m.Set("clutter-adapter-runs-x", keys.Log, []byte("apply done\n"))
		m.Set("clutter-adapter-runs-x", keys.Outputs, []byte(`{"cross_x0123456789_secret_access_key":{"sensitive":true,"value":"s3cr3t"}}`))
		return stopped(aws.Int32(0), "Essential container in task exited")()
	})
	r.Steps = append(r.Steps, []string{OutputStep, "-json", "-no-color"})
	var outputs bytes.Buffer
	r.Outputs = &outputs
	if err := e.LaunchRun(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(outputs.String(), "s3cr3t") {
		t.Fatalf("outputs %q", outputs.String())
	}
	if strings.Contains(r.Output.(*bytes.Buffer).String(), "s3cr3t") {
		t.Fatal("the outputs reached the log")
	}
	if left := m.Keys(); len(left) != 0 {
		t.Fatalf("objects left after the run: %v", left)
	}

	// A run that prints outputs needs somewhere to put them.
	e, _, _, _, r = newTestECS(t)
	r.Steps = append(r.Steps, []string{OutputStep, "-json"})
	if err := e.LaunchRun(context.Background(), r); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("no outputs writer: %v", err)
	}
}
