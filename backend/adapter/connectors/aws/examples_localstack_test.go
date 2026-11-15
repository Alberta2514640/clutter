package aws

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/examples"
)

var (
	addedLine     = regexp.MustCompile(`Apply complete! Resources: (\d+) added`)
	destroyedLine = regexp.MustCompile(`Destroy complete! Resources: (\d+) destroyed`)
)

// TestExamplesOnLocalStack applies and destroys, for real on LocalStack
// through B's job runner, every built-in example that AWS builds alone. Each
// apply must report every block built, and each destroy must remove what the
// apply added. It runs only with CLUTTER_LOCALSTACK=1 (see newLSCloud).
func TestExamplesOnLocalStack(t *testing.T) {
	l := newLSCloud(t)
	list, err := examples.All()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range list {
		if e.Split() || !slices.Contains(e.Clouds, ID) {
			continue
		}
		t.Run(e.ID, func(t *testing.T) {
			g, err := e.Graph()
			if err != nil {
				t.Fatal(err)
			}
			d := testDeployment(lsRegion)
			d.Connection.Credentials = nil
			destroyed := false
			l.destroyAtEnd(t, g, d, func() bool { return destroyed })

			applyLog := l.run(t, connectors.ActionApply, g, d)
			want := fmt.Sprintf("Apply complete: %d blocks built.", len(g.Blocks))
			if len(g.Blocks) == 1 {
				want = "Apply complete: 1 block built."
			}
			if !strings.Contains(applyLog, want) {
				t.Errorf("not every block was reported built (want %q):\n%s", want, applyLog)
			}
			added := addedLine.FindStringSubmatch(applyLog)
			destroyLog := l.run(t, connectors.ActionDestroy, g, d)
			destroyed = true
			gone := destroyedLine.FindStringSubmatch(destroyLog)
			if added == nil || gone == nil || added[1] != gone[1] {
				t.Errorf("added %v, destroyed %v", added, gone)
			}
			if added != nil && gone != nil {
				t.Logf("%s on AWS (LocalStack): %d blocks, %d links, %s resources added, %s destroyed", e.ID, len(g.Blocks), len(g.Edges), added[1], gone[1])
			}
		})
	}
}
