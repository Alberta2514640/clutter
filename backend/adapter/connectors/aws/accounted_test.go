package aws

import (
	"bytes"
	"context"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/google/uuid"
)

// threeNetworksGraph is a diagram a user deployed to AWS: three networks, two
// VMs and a volume, with one link, from vm-2 to vm. Its node ids are the
// editor's (UUIDs).
func threeNetworksGraph() connectors.Graph {
	const (
		network  = "6ab5608d-b7b0-479d-a33b-81ac98307ff4"
		network2 = "7c7ebac8-0fd0-406c-8404-04f3171c541c"
		network3 = "b9d47782-447b-40e2-9e69-92aa9efcea0c"
		vm       = "ad0fd8e7-42ba-4c68-98ef-bb1269e3c3e9"
		vm2      = "e7295367-5fb6-40cc-bcf5-6ac92f2e6478"
		volume   = "17371524-b1b0-4647-8884-c16bc4221acb"
	)
	g := connectors.Graph{Blocks: map[string]connectors.Block{}}
	for _, b := range []connectors.Block{
		{ID: network, Kind: connectors.KindNetwork, Name: "network", Fields: map[string]any{"cidr": "10.0.0.0/24"}},
		{ID: network2, Kind: connectors.KindNetwork, Name: "network-2", Fields: map[string]any{"cidr": "10.0.0.0/24"}},
		{ID: network3, Kind: connectors.KindNetwork, Name: "network-3", Fields: map[string]any{"cidr": "10.0.0.0/24"}},
		{ID: vm, Kind: connectors.KindVM, Name: "vm", Fields: map[string]any{"size": "small", "public": false}},
		{ID: vm2, Kind: connectors.KindVM, Name: "vm-2", Fields: map[string]any{"size": "small", "public": false}},
		{ID: volume, Kind: connectors.KindVolume, Name: "volume", Fields: map[string]any{"size_gb": float64(10)}},
	} {
		g.Blocks[b.ID] = b
	}
	g.Edges = []connectors.Edge{{Source: vm2, Target: vm}}
	return g
}

// accountFor checks, for a run that started, that every block of g is
// exactly one of: built (in res), left out of d, or named by a problem in
// rep. A block that is none of them would be dropped without the deploy
// panel saying so. (A report with a problem stops the run before anything is
// built.)
func accountFor(t *testing.T, g connectors.Graph, d connectors.Deployment, rep capability.Report, res connectors.RunResult) {
	t.Helper()
	named := map[string]bool{}
	for _, p := range rep.Problems {
		named[p.NodeID] = true
	}
	for id, b := range g.Blocks {
		built := res.Resources[id] != ""
		n := 0
		for _, ok := range []bool{built, d.Excluded(id), named[id]} {
			if ok {
				n++
			}
		}
		if n != 1 {
			t.Errorf("block %s (%s): built %t, left out %t, named in a problem %t", b.Name, b.Kind, built, d.Excluded(id), named[id])
		}
	}
}

// TestEveryBlockIsBuiltOrNamed proves an AWS deployment never builds fewer
// blocks than its diagram without the deploy panel saying which and why.
func TestEveryBlockIsBuiltOrNamed(t *testing.T) {
	c, _, _ := newTestConnector(t)
	apply := func(t *testing.T, g connectors.Graph, d connectors.Deployment) connectors.RunResult {
		t.Helper()
		res, err := c.Apply(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &bytes.Buffer{}}, g, d)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	volume := "17371524-b1b0-4647-8884-c16bc4221acb"
	vm2 := "e7295367-5fb6-40cc-bcf5-6ac92f2e6478"

	t.Run("as drawn, the loose volume stops the deploy and is named", func(t *testing.T) {
		g, d := threeNetworksGraph(), testDeployment("ca-central-1")
		rep := capability.CheckConnector(c, g, d)
		if rep.Ready || len(rep.Problems) != 1 || rep.Problems[0].NodeID != volume || rep.Problems[0].Code != CodeVolumeNeedsVM {
			t.Fatalf("the panel shows %+v", rep.Problems)
		}
		// A run cannot start: the API refuses it (not_ready), and the
		// connector refuses to render it.
		if _, err := c.Apply(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &bytes.Buffer{}}, g, d); err == nil {
			t.Fatal("the connector applied a diagram with a problem")
		}
	})

	t.Run("with the volume on vm-2, all six blocks are built", func(t *testing.T) {
		g, d := threeNetworksGraph(), testDeployment("ca-central-1")
		g.Edges = append(g.Edges, connectors.Edge{Source: volume, Target: vm2})
		rep := capability.CheckConnector(c, g, d)
		if !rep.Ready {
			t.Fatalf("problems: %+v", rep.Problems)
		}
		res := apply(t, g, d)
		if len(res.Resources) != len(g.Blocks) {
			t.Errorf("built %d of %d blocks: %v", len(res.Resources), len(g.Blocks), res.Resources)
		}
		accountFor(t, g, d, rep, res)
	})

	t.Run("with the volume left out, five are built and the sixth is an exclusion", func(t *testing.T) {
		g, d := threeNetworksGraph(), testDeployment("ca-central-1")
		d.Exclusions = []string{volume}
		rep := capability.CheckConnector(c, g, d)
		if !rep.Ready {
			t.Fatalf("problems: %+v", rep.Problems)
		}
		res := apply(t, g, d)
		if len(res.Resources) != len(g.Blocks)-1 {
			t.Errorf("built %d blocks: %v", len(res.Resources), res.Resources)
		}
		accountFor(t, g, d, rep, res)
	})

	t.Run("the four blocks that were saved when the run started are all built", func(t *testing.T) {
		// The diagram as saved before vm-2 and the volume were: the run
		// reported "4 blocks built", which is every block it was given.
		g, d := threeNetworksGraph(), testDeployment("ca-central-1")
		delete(g.Blocks, vm2)
		delete(g.Blocks, volume)
		g.Edges = nil
		rep := capability.CheckConnector(c, g, d)
		res := apply(t, g, d)
		if !rep.Ready || len(res.Resources) != 4 {
			t.Fatalf("ready %t, built %d", rep.Ready, len(res.Resources))
		}
		accountFor(t, g, d, rep, res)
	})
}
