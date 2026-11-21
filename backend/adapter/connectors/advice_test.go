package connectors

import (
	"reflect"
	"testing"
)

func TestClosedPublicVMs(t *testing.T) {
	vm := func(id string, public bool) Block {
		return Block{ID: id, Kind: KindVM, Name: id, Fields: map[string]any{"public": public}}
	}
	fw := func(id, ingress string) Block {
		return Block{ID: id, Kind: KindFirewall, Name: id, Fields: map[string]any{"ingress": ingress}}
	}
	g := Graph{
		Blocks: map[string]Block{
			"open":     vm("open", true),
			"closed":   vm("closed", true),
			"bare":     vm("bare", true),
			"private":  vm("private", false),
			"left":     vm("left", true),
			"badrules": vm("badrules", true),
			"web-fw":   fw("web-fw", "tcp/80"),
			"empty-fw": fw("empty-fw", " , "),
			"bad-fw":   fw("bad-fw", "tcp/99999"),
		},
		Edges: []Edge{
			{Source: "web-fw", Target: "open"},
			{Source: "empty-fw", Target: "open"},
			{Source: "empty-fw", Target: "closed"},
			{Source: "bad-fw", Target: "badrules"},
		},
	}
	got := ClosedPublicVMs(g, Deployment{Exclusions: []string{"left"}})
	want := []Problem{
		{NodeID: "bare", Code: CodeClosedPublicVM, Message: MsgClosedPublicVM},
		{NodeID: "closed", Code: CodeClosedPublicVM, Message: MsgClosedPublicVM},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ClosedPublicVMs = %+v, want %+v", got, want)
	}
	// A firewall left out of the deployment lets nothing in there.
	if got := ClosedPublicVMs(g, Deployment{Exclusions: []string{"web-fw", "left", "bare", "closed", "badrules"}}); len(got) != 1 || got[0].NodeID != "open" {
		t.Fatalf("with its firewall left out = %+v", got)
	}
}
