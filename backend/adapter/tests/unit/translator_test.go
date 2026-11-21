// Package unit tests System B's components one at a time: the translator,
// manifest handling, the capability check, the connector registry and the
// job runner's status tracking. Nothing here needs Docker or a cloud.
package unit

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/translate"
)

func TestTranslatorReadsABlocksDiagram(t *testing.T) {
	doc := `{"engine":"blocks","nodes":[
	  {"id":"n1","type":"vm","position":{"x":10,"y":20},"measured":{"w":1},"data":{"name":"web","label":"ignored","fields":{"size":"small","public":true}}},
	  {"id":"n2","type":"network","data":{"name":"net"}}],
	 "edges":[{"id":"e1","source":"n1","target":"n2"},{"id":"e2","source":"n1","target":"n2"}]}`
	g, err := translate.Translate([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Blocks) != 2 {
		t.Fatalf("blocks: %d", len(g.Blocks))
	}
	vm := g.Blocks["n1"]
	if vm.Kind != connectors.KindVM || vm.Name != "web" || vm.Fields["size"] != "small" || vm.Fields["public"] != true {
		t.Fatalf("vm block: %+v", vm)
	}
	if g.Blocks["n2"].Fields == nil {
		t.Fatal("a node without fields must get an empty map, not nil")
	}
	if len(g.Edges) != 1 {
		t.Fatalf("a repeated edge must be kept once, got %d", len(g.Edges))
	}
}

func TestTranslatorSaysAnADiagramIsNotABlocksDiagram(t *testing.T) {
	for _, doc := range []string{`{"nodes":[]}`, `{"engine":"aws","nodes":[]}`} {
		if _, err := translate.Translate([]byte(doc)); !errors.Is(err, translate.ErrNotBlocks) {
			t.Fatalf("%s: got %v, want ErrNotBlocks", doc, err)
		}
	}
}

func TestTranslatorRefusesMalformedDocuments(t *testing.T) {
	many := func(n int, kind string) string {
		var parts []string
		for i := 0; i < n; i++ {
			parts = append(parts, fmt.Sprintf(`{"id":"n%d","type":%q,"data":{"name":"b%d"}}`, i, kind, i))
		}
		return strings.Join(parts, ",")
	}
	manyEdges := func(n int) string {
		var parts []string
		for i := 0; i < n; i++ {
			parts = append(parts, `{"source":"a","target":"b"}`)
		}
		return strings.Join(parts, ",")
	}
	cases := map[string]string{
		"not json":                    `{"engine":"blocks",`,
		"too many blocks":             `{"engine":"blocks","nodes":[` + many(translate.MaxNodes+1, "vm") + `]}`,
		"too many edges":              `{"engine":"blocks","nodes":[{"id":"a","type":"vm"},{"id":"b","type":"vm"}],"edges":[` + manyEdges(translate.MaxEdges+1) + `]}`,
		"id with a quote":             `{"engine":"blocks","nodes":[{"id":"n\"1","type":"vm"}]}`,
		"id with a template sequence": `{"engine":"blocks","nodes":[{"id":"${x}","type":"vm"}]}`,
		"empty id":                    `{"engine":"blocks","nodes":[{"id":"","type":"vm"}]}`,
		"duplicate ids":               `{"engine":"blocks","nodes":[{"id":"a","type":"vm"},{"id":"a","type":"network"}]}`,
		"kind with capitals":          `{"engine":"blocks","nodes":[{"id":"a","type":"VM"}]}`,
		"kind with a dot":             `{"engine":"blocks","nodes":[{"id":"a","type":"aws.ec2"}]}`,
		"edge from nowhere":           `{"engine":"blocks","nodes":[{"id":"a","type":"vm"}],"edges":[{"source":"zz","target":"a"}]}`,
		"edge to nowhere":             `{"engine":"blocks","nodes":[{"id":"a","type":"vm"}],"edges":[{"source":"a","target":"zz"}]}`,
		"edge to itself":              `{"engine":"blocks","nodes":[{"id":"a","type":"vm"}],"edges":[{"source":"a","target":"a"}]}`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := translate.Translate([]byte(doc))
			var te *translate.Error
			if !errors.As(err, &te) {
				t.Fatalf("got %v, want a *translate.Error", err)
			}
			if te.Message == "" || strings.Contains(te.Message, "json:") {
				t.Fatalf("the message must be safe to show: %q", te.Message)
			}
		})
	}
}

func TestTranslatorAcceptsTheLimits(t *testing.T) {
	var parts []string
	for i := 0; i < translate.MaxNodes; i++ {
		parts = append(parts, fmt.Sprintf(`{"id":"n%d","type":"vm","data":{"name":"b%d"}}`, i, i))
	}
	g, err := translate.Translate([]byte(`{"engine":"blocks","nodes":[` + strings.Join(parts, ",") + `]}`))
	if err != nil || len(g.Blocks) != translate.MaxNodes {
		t.Fatalf("exactly %d blocks: %v", translate.MaxNodes, err)
	}
}
