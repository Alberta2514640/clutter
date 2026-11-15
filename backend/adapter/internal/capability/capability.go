// Package capability compares a building-block graph with a connector's
// manifest, minus one deployment's exclusions, before anything runs.
package capability

import (
	"fmt"
	"sort"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
)

// Problem and note codes the check alone reports. A wrong or missing value is
// connectors.CodeInvalidValue or connectors.CodeMissingValue, as a connector
// reports it.
const (
	CodeUnsupportedBlock = "unsupported_block"
	CodeUnsupportedEdge  = "unsupported_edge"
	CodeInvalidName      = "invalid_name"
	CodeUnknownField     = "unknown_field"
	CodeEdgeDropped      = "edge_dropped"
	CodeUnknownExclusion = "unknown_exclusion"
)

// Report is the compatibility of one deployment. Problems stop a run; notes do not.
type Report struct {
	Ready    bool                 `json:"ready"`
	Problems []connectors.Problem `json:"problems"`
	Notes    []connectors.Problem `json:"notes"`
}

// Check compares g with m for deployment d.
func Check(m connectors.Manifest, g connectors.Graph, d connectors.Deployment) Report {
	r := Report{Problems: []connectors.Problem{}, Notes: []connectors.Problem{}}

	for _, x := range d.Exclusions {
		if _, ok := g.Blocks[x]; !ok {
			r.Notes = append(r.Notes, connectors.Problem{NodeID: x, Code: CodeUnknownExclusion,
				Message: "This left-out block is no longer in the diagram."})
		}
	}

	for _, b := range g.Blocks {
		if d.Excluded(b.ID) {
			continue
		}
		spec, ok := m.Block(b.Kind)
		if !ok || spec.Treatment == connectors.TreatmentRemoved {
			r.Problems = append(r.Problems, connectors.Problem{NodeID: b.ID, Code: CodeUnsupportedBlock,
				Message: fmt.Sprintf("%s cannot build a %s block. Remove it from the diagram or leave it out of this deployment.", m.Title, kindTitle(m, b.Kind))})
			continue
		}
		if err := hclsafe.CheckName(b.Name); err != nil {
			r.Problems = append(r.Problems, connectors.Problem{NodeID: b.ID, Field: "name", Code: CodeInvalidName,
				Message: "The block's name must start with a lowercase letter and use only lowercase letters, digits, hyphens and underscores (at most 64)."})
		}
		r.Problems = append(r.Problems, checkFields(b.ID, spec.Fields, b.Fields)...)
	}

	for _, e := range g.Edges {
		if d.Excluded(e.Source) || d.Excluded(e.Target) {
			r.Notes = append(r.Notes, connectors.Problem{EdgeSource: e.Source, EdgeTarget: e.Target, Code: CodeEdgeDropped,
				Message: "This connection is dropped from this deployment because one of its blocks is left out."})
			continue
		}
		s, t := g.Blocks[e.Source], g.Blocks[e.Target]
		if !m.AllowsEdge(s.Kind, t.Kind) {
			r.Problems = append(r.Problems, connectors.Problem{EdgeSource: e.Source, EdgeTarget: e.Target, Code: CodeUnsupportedEdge,
				Message: fmt.Sprintf("%s cannot connect a %s to a %s.", m.Title, kindTitle(m, s.Kind), kindTitle(m, t.Kind))})
		}
	}

	r.Problems = append(r.Problems, checkFields("", m.Settings, d.Settings)...)

	sortProblems(r.Problems)
	sortProblems(r.Notes)
	r.Ready = len(r.Problems) == 0
	return r
}

// CheckConnector is every check one deployment passes before it runs or is
// previewed: Check against c's manifest, then c's own checks (Validate). It
// is the only place B calls Validate, so a connector never repeats Check.
func CheckConnector(c connectors.Connector, g connectors.Graph, d connectors.Deployment) Report {
	r := Check(c.Manifest(), g, d)
	r.Problems = Merge(r.Problems, c.Validate(g, d))
	r.Ready = len(r.Problems) == 0
	return r
}

type problemKey struct{ node, src, tgt, field, code string }

func keyOf(p connectors.Problem) problemKey {
	return problemKey{p.NodeID, p.EdgeSource, p.EdgeTarget, p.Field, p.Code}
}

// Merge appends ps to list, leaving out a problem with the same block or
// edge, field and code as one already there: a value both Check and a
// connector refuse is shown once.
func Merge(list []connectors.Problem, ps []connectors.Problem) []connectors.Problem {
	seen := map[problemKey]bool{}
	for _, p := range list {
		seen[keyOf(p)] = true
	}
	for _, p := range ps {
		if !seen[keyOf(p)] {
			seen[keyOf(p)] = true
			list = append(list, p)
		}
	}
	return list
}

func checkFields(nodeID string, specs []connectors.FieldSpec, values map[string]any) []connectors.Problem {
	var out []connectors.Problem
	known := map[string]connectors.FieldSpec{}
	for _, f := range specs {
		known[f.Name] = f
		v, set := values[f.Name]
		if !set || v == nil {
			if f.Required && f.Default == nil {
				out = append(out, connectors.Problem{NodeID: nodeID, Field: f.Name, Code: connectors.CodeMissingValue,
					Message: fmt.Sprintf("%s is required.", f.Title)})
			}
			continue
		}
		if err := connectors.CheckValue(f, v); err != nil {
			out = append(out, connectors.Problem{NodeID: nodeID, Field: f.Name, Code: connectors.CodeInvalidValue,
				Message: fmt.Sprintf("%s %s.", f.Title, err.Error())})
		}
	}
	for name := range values {
		if _, ok := known[name]; !ok {
			out = append(out, connectors.Problem{NodeID: nodeID, Field: name, Code: CodeUnknownField,
				Message: "This field is not used by this cloud."})
		}
	}
	return out
}

func kindTitle(m connectors.Manifest, k connectors.BlockKind) string {
	if s, ok := m.Block(k); ok && s.Title != "" {
		return s.Title
	}
	return string(k)
}

func sortProblems(ps []connectors.Problem) {
	sort.SliceStable(ps, func(i, j int) bool {
		a, b := ps[i], ps[j]
		if a.NodeID != b.NodeID {
			return a.NodeID < b.NodeID
		}
		if a.EdgeSource != b.EdgeSource {
			return a.EdgeSource < b.EdgeSource
		}
		if a.EdgeTarget != b.EdgeTarget {
			return a.EdgeTarget < b.EdgeTarget
		}
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		return a.Code < b.Code
	})
}
