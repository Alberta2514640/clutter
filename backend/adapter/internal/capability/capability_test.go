package capability_test

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
)

func codes(ps []connectors.Problem) map[string]string {
	out := map[string]string{}
	for _, p := range ps {
		key := p.NodeID
		if p.EdgeSource != "" {
			key = p.EdgeSource + "->" + p.EdgeTarget
		}
		if p.Field != "" {
			key += "." + p.Field
		}
		out[key] = p.Code
	}
	return out
}

func TestCheck(t *testing.T) {
	m := mock.New().Manifest()
	g := connectors.Graph{
		Blocks: map[string]connectors.Block{
			"vm":  {ID: "vm", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"size": "huge", "colour": "red"}},
			"vol": {ID: "vol", Kind: connectors.KindVolume, Name: "Data Disk", Fields: map[string]any{"size_gb": 1.5}},
			"net": {ID: "net", Kind: connectors.KindNetwork, Name: "net", Fields: map[string]any{}},
			"fn":  {ID: "fn", Kind: connectors.KindFunction, Name: "fn", Fields: map[string]any{}},
			"gpu": {ID: "gpu", Kind: "gpu_node", Name: "gpu", Fields: map[string]any{}},
		},
		Edges: []connectors.Edge{{Source: "net", Target: "vm"}, {Source: "fn", Target: "vm"}},
	}
	d := connectors.Deployment{Settings: map[string]any{"image": "windows"}, Exclusions: []string{"gone"}}
	r := capability.Check(m, g, d)
	if r.Ready {
		t.Fatal("ready with problems")
	}
	want := map[string]string{
		"vm.size":     connectors.CodeInvalidValue,
		"vm.colour":   capability.CodeUnknownField,
		"vol.name":    capability.CodeInvalidName,
		"vol.size_gb": connectors.CodeInvalidValue,
		"fn":          capability.CodeUnsupportedBlock,
		"gpu":         capability.CodeUnsupportedBlock,
		"net->vm":     capability.CodeUnsupportedEdge,
		"fn->vm":      capability.CodeUnsupportedEdge,
		".image":      connectors.CodeInvalidValue,
	}
	got := codes(r.Problems)
	for k, c := range want {
		if got[k] != c {
			t.Errorf("%s: got %q, want %q", k, got[k], c)
		}
	}
	if len(got) != len(want) {
		t.Errorf("problems %v", got)
	}
	if codes(r.Notes)["gone"] != capability.CodeUnknownExclusion {
		t.Errorf("notes %v", r.Notes)
	}

	// Leaving out the function drops its edge with a note and clears its problems.
	d.Exclusions = []string{"fn"}
	r = capability.Check(m, g, d)
	got = codes(r.Problems)
	if _, ok := got["fn"]; ok {
		t.Error("an excluded block is still a problem")
	}
	if _, ok := got["fn->vm"]; ok {
		t.Error("an edge to an excluded block is still a problem")
	}
	if codes(r.Notes)["fn->vm"] != capability.CodeEdgeDropped {
		t.Errorf("notes %v", r.Notes)
	}
}

// A problem names a block kind by its title, in a sentence's case (an
// acronym stays), and offers running it on another cloud first.
func TestCheckWordsKindsPlainly(t *testing.T) {
	m := mock.New().Manifest()
	g := connectors.Graph{
		Blocks: map[string]connectors.Block{
			"files": {ID: "files", Kind: connectors.KindBucket, Name: "files", Fields: map[string]any{}},
			"api":   {ID: "api", Kind: connectors.KindHTTPAPI, Name: "api", Fields: map[string]any{}},
		},
	}
	msgs := map[string]string{}
	for _, p := range capability.Check(m, g, connectors.Deployment{}).Problems {
		msgs[p.NodeID] = p.Message
	}
	if want := "Mock cloud cannot build this bucket block. Run it on another cloud, leave it out of this deployment, or remove it from the diagram."; msgs["files"] != want {
		t.Errorf("files: %q, want %q", msgs["files"], want)
	}
	if want := "Mock cloud cannot build this HTTP API block."; len(msgs["api"]) < len(want) || msgs["api"][:len(want)] != want {
		t.Errorf("api: %q", msgs["api"])
	}
}

// A connector that names a cost beyond the blocks (connectors.Advisor) adds
// it as a note, which never stops a run.
type costly struct{ *mock.Connector }

func (costly) Advice(connectors.Graph, connectors.Deployment) []connectors.Problem {
	return []connectors.Problem{{NodeID: "net", Code: connectors.CodeCost, Message: "A gateway costs money."}}
}

func TestCheckConnectorAddsCostNotes(t *testing.T) {
	g := connectors.Graph{Blocks: map[string]connectors.Block{
		"vm": {ID: "vm", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{}},
	}}
	r := capability.CheckConnector(costly{mock.New()}, g, connectors.Deployment{})
	if !r.Ready || codes(r.Notes)["net"] != connectors.CodeCost {
		t.Fatalf("report %+v", r)
	}
	if r = capability.CheckConnector(mock.New(), g, connectors.Deployment{}); len(r.Notes) != 0 {
		t.Fatalf("a connector with no advice has notes: %+v", r.Notes)
	}
}

func TestCheckReady(t *testing.T) {
	m := mock.New().Manifest()
	g := connectors.Graph{Blocks: map[string]connectors.Block{
		"vm": {ID: "vm", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{}},
	}}
	if r := capability.Check(m, g, connectors.Deployment{}); !r.Ready {
		t.Fatalf("a required field with a default is reported missing: %+v", r.Problems)
	}
}
