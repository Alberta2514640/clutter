// Package translate turns a saved diagram document into a building-block graph.
//
// It owns the shape of a building-block diagram document, the `data` column of
// A's diagrams table:
//
//	{
//	  "engine": "blocks",
//	  "nodes": [{"id": "n1", "type": "vm", "data": {"name": "web", "fields": {"size": "small"}}}],
//	  "edges": [{"id": "e1", "source": "n1", "target": "n2"}]
//	}
//
// A node's type is its block kind and data.fields holds its common fields.
// Other keys the canvas keeps (position, measured, other data keys) are ignored.
// Per-cloud settings live on the deployment, not in the document.
package translate

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// Engine is the marker a building-block diagram carries. A owns it, since A's
// generator must skip such a diagram.
const Engine = generic.EngineBlocks

// Limits on one document.
const (
	MaxNodes = 500
	MaxEdges = 2000
)

// NodeIDPattern is the shape of a node id. Node ids address blocks inside
// generated code, so they are held to a safe alphabet here.
var NodeIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ErrNotBlocks means the document is A's, not a building-block diagram.
var ErrNotBlocks = errors.New("the diagram is not a building-block diagram")

// Error is a document that cannot be translated. Its message is safe to show.
type Error struct{ Message string }

func (e *Error) Error() string { return e.Message }

type document struct {
	Engine string `json:"engine"`
	Nodes  []node `json:"nodes"`
	Edges  []edge `json:"edges"`
}

type node struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Name   string         `json:"name"`
		Fields map[string]any `json:"fields"`
	} `json:"data"`
}

type edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Translate parses a diagram document. It returns ErrNotBlocks for a document
// without the blocks marker, and an *Error for a malformed one.
func Translate(raw []byte) (connectors.Graph, error) {
	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return connectors.Graph{}, &Error{Message: "The diagram could not be read."}
	}
	if doc.Engine != Engine {
		return connectors.Graph{}, ErrNotBlocks
	}
	if len(doc.Nodes) > MaxNodes {
		return connectors.Graph{}, &Error{Message: fmt.Sprintf("The diagram has more than %d blocks.", MaxNodes)}
	}
	if len(doc.Edges) > MaxEdges {
		return connectors.Graph{}, &Error{Message: fmt.Sprintf("The diagram has more than %d connections between blocks.", MaxEdges)}
	}

	g := connectors.Graph{Blocks: make(map[string]connectors.Block, len(doc.Nodes))}
	for _, n := range doc.Nodes {
		if !NodeIDPattern.MatchString(n.ID) {
			return connectors.Graph{}, &Error{Message: "A block has an id that is not valid."}
		}
		if _, dup := g.Blocks[n.ID]; dup {
			return connectors.Graph{}, &Error{Message: fmt.Sprintf("Two blocks share the id %q.", n.ID)}
		}
		if !connectors.KindPattern.MatchString(n.Type) {
			return connectors.Graph{}, &Error{Message: fmt.Sprintf("Block %q has a kind that is not valid.", n.ID)}
		}
		fields := n.Data.Fields
		if fields == nil {
			fields = map[string]any{}
		}
		g.Blocks[n.ID] = connectors.Block{
			ID:     n.ID,
			Kind:   connectors.BlockKind(n.Type),
			Name:   n.Data.Name,
			Fields: fields,
		}
	}

	seen := map[connectors.Edge]bool{}
	for _, e := range doc.Edges {
		if _, ok := g.Blocks[e.Source]; !ok {
			return connectors.Graph{}, &Error{Message: "A connection starts at a block that does not exist."}
		}
		if _, ok := g.Blocks[e.Target]; !ok {
			return connectors.Graph{}, &Error{Message: "A connection ends at a block that does not exist."}
		}
		if e.Source == e.Target {
			return connectors.Graph{}, &Error{Message: fmt.Sprintf("Block %q is connected to itself.", e.Source)}
		}
		ce := connectors.Edge{Source: e.Source, Target: e.Target}
		if seen[ce] {
			continue
		}
		seen[ce] = true
		g.Edges = append(g.Edges, ce)
	}
	return g, nil
}
