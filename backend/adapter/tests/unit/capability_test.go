package unit

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/Alberta2514640/clutter/backend/adapter/tests/testkit"
	"github.com/google/uuid"
)

func mockDeployment(settings map[string]any, exclusions ...string) connectors.Deployment {
	id := uuid.NewString()
	return connectors.Deployment{ID: id, Prefix: connectors.PrefixFor(id),
		Connection: connectors.Connection{ConnectorID: "mock", Region: "mock-1"}, Settings: settings, Exclusions: exclusions}
}

func block(id string, kind connectors.BlockKind, name string, fields map[string]any) connectors.Block {
	if fields == nil {
		fields = map[string]any{}
	}
	return connectors.Block{ID: id, Kind: kind, Name: name, Fields: fields}
}

func graph(edges []connectors.Edge, bs ...connectors.Block) connectors.Graph {
	g := connectors.Graph{Blocks: map[string]connectors.Block{}, Edges: edges}
	for _, b := range bs {
		g.Blocks[b.ID] = b
	}
	return g
}

func TestCapabilityReadyGraph(t *testing.T) {
	g := graph([]connectors.Edge{{Source: "n1", Target: "n2"}},
		block("n1", connectors.KindVM, "web", map[string]any{"size": "small"}),
		block("n2", connectors.KindNetwork, "net", nil))
	r := capability.Check(testkit.NewMock().Manifest(), g, mockDeployment(nil))
	if !r.Ready || len(r.Problems) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestCapabilityRules(t *testing.T) {
	m := testkit.NewMock().Manifest()
	cases := []struct {
		name  string
		g     connectors.Graph
		d     connectors.Deployment
		node  string
		code  string
		note  bool
		ready bool
	}{
		{"a removed block is unsupported", graph(nil, block("b", connectors.KindBucket, "files", nil)), mockDeployment(nil), "b", capability.CodeUnsupportedBlock, false, false},
		{"an unknown kind is unsupported", graph(nil, block("g", "gpu", "gpu", nil)), mockDeployment(nil), "g", capability.CodeUnsupportedBlock, false, false},
		{"a left-out block is not checked", graph(nil, block("b", connectors.KindBucket, "files", nil)), mockDeployment(nil, "b"), "", "", false, true},
		{"a left-out id that is gone is a note", graph(nil, block("n1", connectors.KindVM, "web", nil)), mockDeployment(nil, "gone"), "gone", capability.CodeUnknownExclusion, true, true},
		{"a name with capitals", graph(nil, block("n1", connectors.KindVM, "Web", nil)), mockDeployment(nil), "n1", capability.CodeInvalidName, false, false},
		{"a name with a template sequence", graph(nil, block("n1", connectors.KindVM, "${x}", nil)), mockDeployment(nil), "n1", capability.CodeInvalidName, false, false},
		{"a field the cloud does not use", graph(nil, block("n1", connectors.KindVM, "web", map[string]any{"gpu": true})), mockDeployment(nil), "n1", capability.CodeUnknownField, false, false},
		{"a value outside the allowed list", graph(nil, block("n1", connectors.KindVM, "web", map[string]any{"size": "huge"})), mockDeployment(nil), "n1", connectors.CodeInvalidValue, false, false},
		{"a value of the wrong type", graph(nil, block("v", connectors.KindVolume, "data", map[string]any{"size_gb": "ten"})), mockDeployment(nil), "v", connectors.CodeInvalidValue, false, false},
		{"a setting outside its allowed list", graph(nil, block("n1", connectors.KindVM, "web", nil)), mockDeployment(map[string]any{"image": "windows-11"}), "", connectors.CodeInvalidValue, false, false},
		{"an unknown setting", graph(nil, block("n1", connectors.KindVM, "web", nil)), mockDeployment(map[string]any{"flavor": "x"}), "", capability.CodeUnknownField, false, false},
		{"an edge the cloud cannot wire", graph([]connectors.Edge{{Source: "n2", Target: "n1"}},
			block("n1", connectors.KindVM, "web", nil), block("n2", connectors.KindNetwork, "net", nil)), mockDeployment(nil), "", capability.CodeUnsupportedEdge, false, false},
		{"an edge to a left-out block is dropped", graph([]connectors.Edge{{Source: "n1", Target: "b"}},
			block("n1", connectors.KindVM, "web", nil), block("b", connectors.KindBucket, "files", nil)), mockDeployment(nil, "b"), "", capability.CodeEdgeDropped, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := capability.Check(m, c.g, c.d)
			if r.Ready != c.ready {
				t.Fatalf("ready %v, want %v: %+v", r.Ready, c.ready, r)
			}
			if c.code == "" {
				return
			}
			list := r.Problems
			if c.note {
				list = r.Notes
			}
			if !contract.HasProblem(list, c.node, c.code) {
				t.Fatalf("no %s on %q in %+v", c.code, c.node, list)
			}
		})
	}
}

func TestCapabilityOrderIsStable(t *testing.T) {
	g := graph(nil, block("b", connectors.KindBucket, "files", nil), block("a", connectors.KindTable, "t", nil), block("c", connectors.KindFunction, "f", nil))
	first := capability.Check(testkit.NewMock().Manifest(), g, mockDeployment(nil))
	for i := 0; i < 20; i++ {
		again := capability.Check(testkit.NewMock().Manifest(), g, mockDeployment(nil))
		for j := range first.Problems {
			if first.Problems[j].NodeID != again.Problems[j].NodeID {
				t.Fatal("problems came back in a different order")
			}
		}
	}
	if first.Problems[0].NodeID != "a" {
		t.Fatalf("problems are not sorted by block: %+v", first.Problems)
	}
}

func TestCapabilityMergeShowsAProblemOnce(t *testing.T) {
	p := connectors.Problem{NodeID: "n1", Field: "size", Code: connectors.CodeInvalidValue, Message: "one"}
	q := p
	q.Message = "the same problem, said by the connector"
	out := capability.Merge([]connectors.Problem{p}, []connectors.Problem{q, {NodeID: "n2", Code: "x"}})
	if len(out) != 2 {
		t.Fatalf("%+v", out)
	}
}

func TestCheckConnectorAddsTheConnectorsOwnRules(t *testing.T) {
	os := testkit.NewOpenStack(t, &testkit.FakeTerraform{})
	id := uuid.NewString()
	d := connectors.Deployment{ID: id, Prefix: connectors.PrefixFor(id),
		Connection: connectors.Connection{ConnectorID: os.ID(), Region: testkit.OpenStackRegion}}
	g := graph([]connectors.Edge{{Source: "n1", Target: "n2"}},
		block("n1", connectors.KindVM, "web", nil),
		block("n2", connectors.KindNetwork, "net", map[string]any{"cidr": "10.0.0.0/30"}))
	r := capability.CheckConnector(os, g, d)
	if r.Ready || !contract.HasProblem(r.Problems, "n2", connectors.CodeInvalidValue) {
		t.Fatalf("a /30 network passes the manifest but not OpenStack's own rule: %+v", r)
	}
}
