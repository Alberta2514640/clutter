package terraform

import (
	"io"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

func TestLineWriterHandsOnWholeLines(t *testing.T) {
	var lines []string
	w := NewLineWriter(func(s string) error { lines = append(lines, s); return nil })
	for _, chunk := range []string{"one\ntw", "o\n", "", "three\nlast"} {
		if _, err := io.WriteString(w, chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(lines, "|"); got != "one\n|two\n|three\n|last" {
		t.Fatalf("got %q", got)
	}
}

func TestLineWriterLeavesOutALongLineWhole(t *testing.T) {
	var lines []string
	w := NewLineWriter(func(s string) error { lines = append(lines, s); return nil })
	long := strings.Repeat("x", MaxLine+10)
	for _, chunk := range []string{"a\n", long[:MaxLine/2], long[MaxLine/2:], "tail\nb\n", long + "\nc\n"} {
		if _, err := io.WriteString(w, chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	want := []string{"a\n", LongLine, "b\n", LongLine, "c\n"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q", lines)
	}
	// A line of exactly MaxLine bytes is kept.
	lines = nil
	if _, err := io.WriteString(w, strings.Repeat("y", MaxLine)+"\n"); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0]) != MaxLine+1 {
		t.Fatalf("a line at the cap: %d lines", len(lines))
	}
}

func TestPlanSummary(t *testing.T) {
	cases := map[string]struct {
		res connectors.PlanResult
		ok  bool
	}{
		"Plan: 2 to add, 1 to change, 3 to destroy.\n":                  {connectors.PlanResult{Add: 2, Change: 1, Destroy: 3}, true},
		"  Plan: 1 to import, 4 to add, 0 to change, 0 to destroy.":     {connectors.PlanResult{Add: 4}, true},
		"No changes. Your infrastructure matches the configuration.":    {connectors.PlanResult{}, true},
		`  + description = "Plan: 9 to add, 9 to change, 9 to destroy"`: {connectors.PlanResult{}, false},
		"Apply complete! Resources: 1 added, 0 changed, 0 destroyed.":   {connectors.PlanResult{}, false},
	}
	for line, want := range cases {
		res, ok := PlanSummary(line)
		if ok != want.ok || res != want.res {
			t.Errorf("%q: %+v %v", line, res, ok)
		}
	}
}
