package translate

import (
	"errors"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

func TestTranslate(t *testing.T) {
	doc := `{"engine":"blocks","nodes":[
	  {"id":"n1","type":"vm","position":{"x":1,"y":2},"data":{"name":"web","label":"ignored","fields":{"size":"small"}}},
	  {"id":"n2","type":"network","data":{"name":"net"}}],
	  "edges":[{"id":"e1","source":"n1","target":"n2"},{"id":"e2","source":"n1","target":"n2"}]}`
	g, err := Translate([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Blocks) != 2 || g.Blocks["n1"].Kind != connectors.KindVM || g.Blocks["n1"].Name != "web" || g.Blocks["n1"].Fields["size"] != "small" {
		t.Fatalf("blocks: %+v", g.Blocks)
	}
	if g.Blocks["n2"].Fields == nil {
		t.Fatal("a block without fields has nil fields")
	}
	if len(g.Edges) != 1 {
		t.Fatalf("duplicate edge kept: %+v", g.Edges)
	}
}

func TestTranslateRefuses(t *testing.T) {
	if _, err := Translate([]byte(`{"nodes":[],"edges":[]}`)); !errors.Is(err, ErrNotBlocks) {
		t.Fatalf("A's document: %v", err)
	}
	bad := map[string]string{
		"not json":       `{`,
		"bad node id":    `{"engine":"blocks","nodes":[{"id":"a b","type":"vm"}]}`,
		"injected id":    `{"engine":"blocks","nodes":[{"id":"x\"}","type":"vm"}]}`,
		"duplicate id":   `{"engine":"blocks","nodes":[{"id":"a","type":"vm"},{"id":"a","type":"vm"}]}`,
		"bad kind":       `{"engine":"blocks","nodes":[{"id":"a","type":"VM!"}]}`,
		"dangling edge":  `{"engine":"blocks","nodes":[{"id":"a","type":"vm"}],"edges":[{"source":"a","target":"b"}]}`,
		"self edge":      `{"engine":"blocks","nodes":[{"id":"a","type":"vm"}],"edges":[{"source":"a","target":"a"}]}`,
		"edge from none": `{"engine":"blocks","nodes":[{"id":"a","type":"vm"}],"edges":[{"source":"z","target":"a"}]}`,
	}
	for name, doc := range bad {
		_, err := Translate([]byte(doc))
		var te *Error
		if !errors.As(err, &te) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
