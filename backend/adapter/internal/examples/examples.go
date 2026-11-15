// Package examples owns Clutter's built-in example diagrams. Each one is a
// JSON file in this folder, embedded in the build, holding a building-block
// document (the shape internal/translate owns) and the clouds it fits. The
// website offers them under "New diagram"; B API serves them (GET
// /v1/examples). The tests prove each example passes the capability check
// and its connector's own checks on every cloud it names, and renders.
//
// An example names its clouds by connector id, as data. No code here or in
// the core branches on those ids: the API keeps an example only while every
// cloud it names is in the registry.
package examples

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"sync"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/translate"
)

//go:embed *.json
var files embed.FS

// Example is one built-in example diagram.
type Example struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Level is how much the example asks of a new user (Levels). The files
	// are ordered from easy to complex, each adding one idea.
	Level       string `json:"level"`
	Description string `json:"description"`
	// Clouds are the connectors the example fits, by id. Without
	// Placements, each one alone builds the whole example. With Placements,
	// the example is one split deployment over all of them: the first holds
	// every block no placement names.
	Clouds     []string    `json:"clouds"`
	Placements []Placement `json:"placements,omitempty"`
	// Document is the diagram document the new diagram is saved with.
	Document json.RawMessage `json:"document"`
}

// Placement puts some of a split example's blocks on another of its clouds.
type Placement struct {
	Connector string   `json:"connector"`
	NodeIDs   []string `json:"node_ids"`
}

// Split reports whether the example is one deployment over several clouds.
func (e Example) Split() bool { return len(e.Placements) > 0 }

// Graph is the example's document as a building-block graph.
func (e Example) Graph() (connectors.Graph, error) { return translate.Translate(e.Document) }

// Limits an example is held to, so the website can show and save it.
var (
	idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,40}$`)
)

const (
	// maxName is A's cap on a diagram's name, which the new diagram takes.
	maxName = 32
	// maxDescription keeps a description to one short line.
	maxDescription = 100
)

// Levels are the example levels, from easy to complex. The website labels
// each one (examples/types.ts copies this list).
var Levels = []string{"starter", "intermediate", "advanced"}

func levelIndex(level string) int {
	for i, l := range Levels {
		if l == level {
			return i
		}
	}
	return -1
}

// All returns every example, in the order of their file names. It reads the
// embedded files once.
func All() ([]Example, error) { return load() }

var load = sync.OnceValues(func() ([]Example, error) {
	names, err := fs.Glob(files, "*.json")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	out := make([]Example, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		raw, err := files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var e Example
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, fmt.Errorf("example %s: %w", name, err)
		}
		if err := e.check(); err != nil {
			return nil, fmt.Errorf("example %s: %w", name, err)
		}
		if seen[e.ID] {
			return nil, fmt.Errorf("example %s: the id %q is used twice", name, e.ID)
		}
		if len(out) > 0 && levelIndex(e.Level) < levelIndex(out[len(out)-1].Level) {
			return nil, fmt.Errorf("example %s: it comes after a harder example", name)
		}
		seen[e.ID] = true
		out = append(out, e)
	}
	return out, nil
})

// check holds an example to the shape the website and B expect.
func (e Example) check() error {
	switch {
	case !idPattern.MatchString(e.ID):
		return fmt.Errorf("the id %q is not valid", e.ID)
	case e.Name == "" || len([]rune(e.Name)) > maxName:
		return fmt.Errorf("the name must be 1 to %d characters", maxName)
	case levelIndex(e.Level) < 0:
		return fmt.Errorf("the level %q is not one of %v", e.Level, Levels)
	case e.Description == "" || len([]rune(e.Description)) > maxDescription:
		return fmt.Errorf("the description must be 1 to %d characters", maxDescription)
	case len(e.Clouds) == 0:
		return fmt.Errorf("it names no cloud")
	}
	clouds := map[string]bool{}
	for _, c := range e.Clouds {
		if !connectors.ConnectorIDPattern.MatchString(c) || clouds[c] {
			return fmt.Errorf("the cloud %q is not valid or appears twice", c)
		}
		clouds[c] = true
	}
	g, err := e.Graph()
	if err != nil {
		return fmt.Errorf("the document: %w", err)
	}
	placed := map[string]bool{}
	for _, p := range e.Placements {
		if !clouds[p.Connector] || p.Connector == e.Clouds[0] || len(p.NodeIDs) == 0 {
			return fmt.Errorf("a placement must name one of the example's clouds other than the first, and at least one block")
		}
		for _, id := range p.NodeIDs {
			if _, ok := g.Blocks[id]; !ok || placed[id] {
				return fmt.Errorf("the placed block %q is not in the document or is placed twice", id)
			}
			placed[id] = true
		}
	}
	if e.Split() && len(e.Placements) != len(e.Clouds)-1 {
		return fmt.Errorf("a split example places blocks on every cloud after the first")
	}
	return nil
}

// Fitting returns the examples this build can deploy, given has, which
// reports whether a connector is offered. An example keeps only its offered
// clouds and is left out when none is; a split example is left out unless
// every one of its clouds is offered.
func Fitting(all []Example, has func(connector string) bool) []Example {
	out := []Example{}
	for _, e := range all {
		var clouds []string
		for _, c := range e.Clouds {
			if has(c) {
				clouds = append(clouds, c)
			}
		}
		if len(clouds) == 0 || (e.Split() && len(clouds) < len(e.Clouds)) {
			continue
		}
		e.Clouds = clouds
		out = append(out, e)
	}
	return out
}
