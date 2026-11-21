// Package split turns a deployment whose blocks are placed on several clouds
// into parts: one per cloud, each with the blocks on it, the cross-cloud edges
// it exports and imports, and the order the parts are applied in (ADR-020,
// docs/architecture/system-b-adapter-layer.md § Split deployments).
//
// It never names a connector: it asks each part's connector, through
// connectors.CrossCloud, what it can serve.
package split

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/google/uuid"
)

// Problem codes.
const (
	CodeCrossEdge    = "cross_cloud_edge"
	CodeCycle        = "cross_cloud_cycle"
	CodeNoSplit      = "cloud_cannot_split"
	CodeCrossAddress = "cross_cloud_address"
	CodeCrossKey     = "cross_cloud_key"
)

// Cloud is one connection of a deployment: the primary, or one with blocks
// placed on it.
type Cloud struct {
	ConnectionID string
	// Name is the connection's name, for the run's log and its failure.
	Name      string
	Connector connectors.Connector
	Region    string
	Settings  map[string]any
	SecretRef string
	// Nodes are the blocks placed on this cloud. The primary holds every
	// block no other cloud holds, and its Nodes is ignored.
	Nodes   []string
	Primary bool
}

// Part is one cloud of a split deployment, ready to run. Its Deployment has no
// credentials: the job runner's broker adds them for that part's step only.
type Part struct {
	Cloud
	Deployment connectors.Deployment
}

// Plan is a split deployment's parts in the order they are applied (a cloud
// whose blocks another cloud's blocks depend on comes first) and its
// cross-cloud edges. Problems stop a run; Notes do not.
type Plan struct {
	Parts    []Part
	Edges    []connectors.CrossEdge
	Problems []connectors.Problem
	Notes    []connectors.Problem
	// Owner maps each block's node id to its part's index in Parts.
	Owner map[string]int
}

// PartID is the id of the part of deployment deploymentID on one connection.
// It keys the part's Terraform state. The primary's part keeps the
// deployment's own id, so a deployment's state stays where it was before any
// block was placed elsewhere.
func PartID(deploymentID, connectionID string, primary bool) string {
	if primary {
		return deploymentID
	}
	ns, err := uuid.Parse(deploymentID)
	if err != nil {
		return ""
	}
	return uuid.NewSHA1(ns, []byte("part:"+connectionID)).String()
}

// StepRunID is the run id of one step of a split run: one part's plan, apply,
// destroy or roll-back. It keys the step's run folder and its task.
func StepRunID(runID, connectionID, step string) string {
	ns, err := uuid.Parse(runID)
	if err != nil {
		return ""
	}
	return uuid.NewSHA1(ns, []byte(step+":"+connectionID)).String()
}

// Build plans deployment deploymentID of graph g over clouds (the primary
// first). exclusions are the deployment's own left-out blocks.
func Build(g connectors.Graph, deploymentID string, exclusions []string, clouds []Cloud) Plan {
	p := Plan{Owner: map[string]int{}}
	excluded := map[string]bool{}
	for _, x := range exclusions {
		excluded[x] = true
	}

	// Who holds each block: its placement, or the primary.
	primary := 0
	for i, c := range clouds {
		if c.Primary {
			primary = i
		}
	}
	owner := map[string]int{}
	for i, c := range clouds {
		if c.Primary {
			continue
		}
		for _, n := range c.Nodes {
			if _, ok := g.Blocks[n]; !ok {
				p.Notes = append(p.Notes, connectors.Problem{NodeID: n, Code: "unknown_placement",
					Message: fmt.Sprintf("A block placed on %s is no longer in the diagram.", c.Name)})
				continue
			}
			owner[n] = i
		}
	}
	for id := range g.Blocks {
		if _, ok := owner[id]; !ok {
			owner[id] = primary
		}
	}

	// The cross-cloud edges, between blocks this deployment includes.
	type link struct{ from, to int }
	var links []link
	for _, e := range g.Edges {
		if excluded[e.Source] || excluded[e.Target] {
			continue
		}
		from, to := owner[e.Source], owner[e.Target]
		if from == to {
			continue
		}
		s, t := g.Blocks[e.Source], g.Blocks[e.Target]
		class := connectors.CrossClass(s.Kind, t.Kind)
		if class == "" {
			p.Problems = append(p.Problems, connectors.Problem{EdgeSource: e.Source, EdgeTarget: e.Target, Code: CodeCrossEdge,
				Message: crossRefusal(s, t)})
			continue
		}
		p.Edges = append(p.Edges, connectors.NewCrossEdge(e.Source, e.Target, class))
		links = append(links, link{from: from, to: to})
		p.Notes = append(p.Notes, crossNote(class, s, t, clouds[from].Name, clouds[to].Name))
	}

	// Every cloud of a split must be able to take part in one.
	if len(clouds) > 1 {
		for _, c := range clouds {
			if _, ok := c.Connector.(connectors.CrossCloud); !ok {
				p.Problems = append(p.Problems, connectors.Problem{Code: CodeNoSplit,
					Message: fmt.Sprintf("%s (%s) cannot be one of several clouds in a deployment. Keep this deployment's blocks on one cloud.", c.Name, c.Connector.Manifest().Title)})
			}
		}
	}

	// Targets first: a part runs after every part whose values it imports.
	order, cyclic := topo(len(clouds), primary, func(add func(from, to int)) {
		for _, l := range links {
			add(l.to, l.from)
		}
	})
	if cyclic {
		var names []string
		for _, c := range clouds {
			names = append(names, c.Name)
		}
		p.Problems = append(p.Problems, connectors.Problem{Code: CodeCycle,
			Message: fmt.Sprintf("The clouds of this deployment (%s) hand values to each other in a circle. Place the blocks so values flow one way between clouds.", strings.Join(names, ", "))})
	}

	index := map[int]int{}
	prefix := connectors.PrefixFor(deploymentID)
	for pos, i := range order {
		index[i] = pos
		c := clouds[i]
		var x []string
		x = append(x, exclusions...)
		for _, id := range sortedIDs(g) {
			if owner[id] != i && !excluded[id] {
				x = append(x, id)
			}
		}
		p.Parts = append(p.Parts, Part{Cloud: c, Deployment: connectors.Deployment{
			ID:         PartID(deploymentID, c.ConnectionID, c.Primary),
			Prefix:     prefix,
			Connection: connectors.Connection{ID: c.ConnectionID, ConnectorID: c.Connector.ID(), Region: c.Region},
			Settings:   c.Settings,
			Exclusions: x,
			Cross:      &connectors.Cross{},
		}})
	}
	for id, i := range owner {
		p.Owner[id] = index[i]
	}
	for _, e := range p.Edges {
		src, tgt := &p.Parts[p.Owner[e.Source]], &p.Parts[p.Owner[e.Target]]
		tgt.Deployment.Cross.Exports = append(tgt.Deployment.Cross.Exports, e)
		src.Deployment.Cross.Imports = append(src.Deployment.Cross.Imports, connectors.CrossImport{CrossEdge: e})
	}
	return p
}

// topo orders n nodes so every edge's from comes before its to, keeping the
// lowest index first among ready nodes, but first always first when it is
// ready. It reports a cycle, and then still returns every node.
func topo(n, first int, edges func(add func(from, to int))) ([]int, bool) {
	in := make([]int, n)
	next := make([][]int, n)
	edges(func(from, to int) {
		next[from] = append(next[from], to)
		in[to]++
	})
	done := make([]bool, n)
	var out []int
	for len(out) < n {
		pick := -1
		if !done[first] && in[first] == 0 {
			pick = first
		} else {
			for i := 0; i < n; i++ {
				if !done[i] && in[i] == 0 {
					pick = i
					break
				}
			}
		}
		if pick < 0 {
			// A cycle: keep the rest in their own order.
			for i := 0; i < n; i++ {
				if !done[i] {
					out = append(out, i)
				}
			}
			return out, true
		}
		done[pick] = true
		out = append(out, pick)
		for _, t := range next[pick] {
			in[t]--
		}
	}
	return out, false
}

func sortedIDs(g connectors.Graph) []string {
	ids := make([]string, 0, len(g.Blocks))
	for id := range g.Blocks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// crossRefusal says why an edge cannot cross clouds.
func crossRefusal(s, t connectors.Block) string {
	switch {
	case s.Kind == connectors.KindVM && t.Kind == connectors.KindNetwork,
		s.Kind == connectors.KindVolume && t.Kind == connectors.KindVM,
		s.Kind == connectors.KindFirewall && t.Kind == connectors.KindVM:
		return fmt.Sprintf("%s and %s are wired together inside one cloud. Place both on the same cloud.", s.Name, t.Name)
	case s.Kind == connectors.KindFunction:
		return fmt.Sprintf("A function reads only on its own cloud. Place %s and %s on the same cloud, or connect a VM to %s instead.", s.Name, t.Name, t.Name)
	}
	return fmt.Sprintf("Clutter cannot connect %s to %s across clouds. Place both on the same cloud.", s.Name, t.Name)
}

// crossNote says what a cross-cloud edge will do, for the deploy panel.
func crossNote(class connectors.EdgeClass, s, t connectors.Block, sourceCloud, targetCloud string) connectors.Problem {
	if class == connectors.EdgePermission {
		return connectors.Problem{EdgeSource: s.ID, EdgeTarget: t.ID, Code: CodeCrossKey,
			Message: fmt.Sprintf("Clutter creates a key that can only read %s and puts it on %s. The key is deleted when this deployment is destroyed.", t.Name, s.Name)}
	}
	return connectors.Problem{EdgeSource: s.ID, EdgeTarget: t.ID, Code: CodeCrossAddress,
		Message: fmt.Sprintf("%s (%s) gets the public address of %s (%s) when it first starts.", s.Name, sourceCloud, t.Name, targetCloud)}
}
