package main

import (
	"context"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
)

// The output step's stdout holds sensitive values: it goes to its own object,
// never to the log B copies to the user.
func TestTheOutputStepNeverReachesTheLog(t *testing.T) {
	steps := [][]string{
		{"init", "-input=false", "-no-color"},
		{"apply", "-input=false", "-no-color", "-auto-approve", "-lock-timeout=60s"},
		{runtask.OutputStep, "-json", "-no-color"},
	}
	w, m, _ := setup(t, runtask.RunSpec{Steps: steps, Dir: "/b/data/work/" + testRun})
	if code := w.Run(context.Background()); code != 0 {
		t.Fatalf("exit %d, status %s", code, w.Status)
	}
	keys := runtask.KeysFor(testRun)
	log, _ := m.Object(testBucket, keys.Log)
	if !strings.Contains(string(log), "terraform says apply") || strings.Contains(string(log), "terraform says output") {
		t.Fatalf("log %q", log)
	}
	out, ok := m.Object(testBucket, keys.Outputs)
	if !ok || strings.TrimSpace(string(out)) != "terraform says output" {
		t.Fatalf("outputs %q (%v)", out, ok)
	}
}
