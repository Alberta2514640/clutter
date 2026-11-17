package generalopenstack

import (
	"bytes"
	"context"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/examples"
	"github.com/google/uuid"
)

var (
	addedLine     = regexp.MustCompile(`Apply complete! Resources: (\d+) added`)
	destroyedLine = regexp.MustCompile(`Destroy complete! Resources: (\d+) destroyed`)
)

// TestExamplesOnRealCloud applies and destroys, on the real OpenStack cloud
// TestRealCloud uses (such as DevStack), every built-in example OpenStack
// builds alone. Each must pass the core's checks, every block must come back
// built, and the destroy must remove what the apply added. It runs only when
// the CLUTTER_OPENSTACK_* variables are set.
func TestExamplesOnRealCloud(t *testing.T) {
	rc := newRealCloud(t)
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
			d := rc.deployment()
			if rep := capability.CheckConnector(rc.c, g, d); !rep.Ready {
				t.Fatalf("problems: %+v", rep.Problems)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
			defer cancel()
			var log bytes.Buffer
			run := func(a connectors.Action) connectors.Run {
				return connectors.Run{ID: uuid.NewString(), Action: a, Log: &log}
			}
			destroyed := false
			destroy := func() string {
				log.Reset()
				dctx, dcancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer dcancel()
				if _, err := rc.c.Destroy(dctx, run(connectors.ActionDestroy), d); err != nil {
					t.Errorf("destroy: %v\n%s", err, rc.redact(log.String()))
				}
				destroyed = true
				return log.String()
			}
			t.Cleanup(func() {
				if !destroyed {
					destroy()
				}
			})

			res, err := rc.c.Apply(ctx, run(connectors.ActionApply), g, d)
			applyLog := log.String()
			if err != nil {
				t.Fatalf("apply: %v\n%s", err, rc.redact(applyLog))
			}
			for id := range g.Blocks {
				if res.Resources[id] == "" {
					t.Errorf("block %s was not reported built: %v", id, res.Resources)
				}
			}
			added := addedLine.FindStringSubmatch(applyLog)
			gone := destroyedLine.FindStringSubmatch(destroy())
			if added == nil || gone == nil || added[1] != gone[1] {
				t.Errorf("added %v, destroyed %v", added, gone)
				return
			}
			t.Logf("%s on OpenStack: %d blocks, %d links, %d blocks built, %s resources added, %s destroyed",
				e.ID, len(g.Blocks), len(g.Edges), len(res.Resources), added[1], gone[1])
		})
	}
}
