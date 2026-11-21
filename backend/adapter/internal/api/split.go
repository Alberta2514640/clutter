package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/split"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/store"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/translate"
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// placementBody places some of a deployment's blocks on another of the
// organization's connections, with that cloud's own settings (ADR-020).
type placementBody struct {
	ConnectionID string         `json:"connection_id"`
	NodeIDs      []string       `json:"node_ids"`
	Settings     map[string]any `json:"settings"`
}

// placementView is a placement as the deploy panel reads it. It never carries
// the connection's secret reference.
type placementView struct {
	ConnectionID   string         `json:"connection_id"`
	ConnectionName string         `json:"connection_name"`
	Connector      string         `json:"connector"`
	Region         string         `json:"region"`
	NodeIDs        []string       `json:"node_ids"`
	Settings       map[string]any `json:"settings"`
}

func placementsOut(ps []store.Placement) []placementView {
	out := make([]placementView, 0, len(ps))
	for _, p := range ps {
		out = append(out, placementView{ConnectionID: p.ConnectionID, ConnectionName: p.ConnectionName, Connector: p.ConnectorID,
			Region: p.Region, NodeIDs: p.NodeIDs, Settings: p.Settings})
	}
	return out
}

// maxPlacements caps how many other clouds one deployment's blocks may sit on.
const maxPlacements = 8

// refusal is an answer a check gives instead of a value.
type refusal struct {
	status   int
	code     string
	message  string
	problems []connectors.Problem
}

func (rf *refusal) write(w http.ResponseWriter) {
	writeProblems(w, rf.status, rf.code, rf.message, rf.problems)
}

// cleanPlacements checks placements for a deployment on connection primary of
// graph g: each names one of the organization's connections (another
// organization's is not found), never the primary and never twice; each holds
// at least one of the diagram's blocks, and no block is placed twice; and its
// settings are checked against that cloud's manifest.
func (a *API) cleanPlacements(ctx context.Context, orgID, primary string, in []placementBody, g connectors.Graph) ([]store.Placement, *refusal, error) {
	invalid := func(msg string) *refusal {
		return &refusal{status: http.StatusUnprocessableEntity, code: "invalid_placements", message: msg}
	}
	if len(in) > maxPlacements {
		return nil, invalid(fmt.Sprintf("A deployment can place blocks on at most %d other clouds.", maxPlacements)), nil
	}
	seenConn := map[string]bool{}
	seenNode := map[string]bool{}
	out := make([]store.Placement, 0, len(in))
	for _, p := range in {
		id, ok := generic.CanonicalUuid(p.ConnectionID)
		if !ok {
			return nil, &refusal{status: http.StatusNotFound, code: "not_found", message: "Not found."}, nil
		}
		if id == primary {
			return nil, invalid("A block placed on another cloud must name a connection other than the deployment's own."), nil
		}
		if seenConn[id] {
			return nil, invalid("Each cloud appears once in a deployment's placements."), nil
		}
		seenConn[id] = true
		if len(p.NodeIDs) == 0 || len(p.NodeIDs) > translate.MaxNodes {
			return nil, invalid("A cloud in a deployment's placements needs at least one block."), nil
		}
		nodes := make([]string, 0, len(p.NodeIDs))
		for _, n := range p.NodeIDs {
			if !translate.NodeIDPattern.MatchString(n) {
				return nil, invalid("Placements must list block ids."), nil
			}
			if _, found := g.Blocks[n]; !found {
				return nil, &refusal{status: http.StatusUnprocessableEntity, code: "unknown_block",
					message: "A placed block is not in the diagram. Reload the diagram and try again."}, nil
			}
			if seenNode[n] {
				return nil, invalid("A block can be placed on one cloud only."), nil
			}
			seenNode[n] = true
			nodes = append(nodes, n)
		}
		sort.Strings(nodes)
		conn, err := a.Store.GetConnection(ctx, orgID, id)
		if errors.Is(err, store.ErrNotFound) {
			// Another organization's connection, or none: the same answer.
			return nil, &refusal{status: http.StatusNotFound, code: "not_found", message: "Not found."}, nil
		}
		if err != nil {
			return nil, nil, err
		}
		c, found := a.Registry.Get(conn.ConnectorID)
		if !found {
			return nil, &refusal{status: http.StatusUnprocessableEntity, code: "unknown_connector", message: "This cloud is not available."}, nil
		}
		settings, problems := cleanSettings(c.Manifest(), p.Settings)
		if len(problems) > 0 {
			return nil, &refusal{status: http.StatusUnprocessableEntity, code: "invalid_settings",
				message: fmt.Sprintf("Some settings are not valid for %s.", conn.Name), problems: problems}, nil
		}
		out = append(out, store.Placement{ConnectionID: id, NodeIDs: nodes, Settings: settings})
	}
	return out, nil, nil
}

// splitOf plans a deployment with placements over its clouds. A destroy
// plans over an empty graph: it needs only each cloud's state.
func (a *API) splitOf(row store.Deployment, primary connectors.Connector, g connectors.Graph) (*split.Plan, *refusal) {
	clouds := []split.Cloud{{ConnectionID: row.ConnectionID, Name: row.ConnectionName, Connector: primary, Region: row.Region,
		Settings: row.Settings, SecretRef: row.SecretRef, Primary: true}}
	for _, p := range row.Placements {
		c, found := a.Registry.Get(p.ConnectorID)
		if !found {
			return nil, &refusal{status: http.StatusUnprocessableEntity, code: "unknown_connector", message: "One of this deployment's clouds is not available."}
		}
		clouds = append(clouds, split.Cloud{ConnectionID: p.ConnectionID, Name: p.ConnectionName, Connector: c, Region: p.Region,
			Settings: p.Settings, SecretRef: p.SecretRef, Nodes: p.NodeIDs})
	}
	if g.Blocks == nil {
		g.Blocks = map[string]connectors.Block{}
	}
	plan := split.Build(g, row.ID, row.Exclusions, clouds)
	return &plan, nil
}

// cloudReport is one cloud's compatibility in a split deployment.
type cloudReport struct {
	ConnectionID   string `json:"connection_id"`
	ConnectionName string `json:"connection_name"`
	Connector      string `json:"connector"`
	// Position is the cloud's place in the order a run applies the clouds.
	Position int `json:"position"`
	capability.Report
}

// compatibilityView is a deployment's compatibility; Clouds is set only for a
// split deployment, whose Problems and Notes then gather every cloud's.
type compatibilityView struct {
	capability.Report
	Clouds []cloudReport `json:"clouds,omitempty"`
}

// splitCompatibility is the capability check, the connector's own checks and
// its cross-cloud checks for every cloud of a split deployment.
func splitCompatibility(l loaded) compatibilityView {
	real := map[string]bool{}
	for _, x := range l.row.Exclusions {
		real[x] = true
	}
	out := compatibilityView{Report: capability.Report{Problems: []connectors.Problem{}, Notes: []connectors.Problem{}}}
	out.Problems = capability.Merge(out.Problems, l.split.Problems)
	out.Notes = capability.Merge(out.Notes, l.split.Notes)
	for i, p := range l.split.Parts {
		rep := capability.CheckConnector(p.Connector, l.graph, p.Deployment)
		// An edge to a block on another cloud is a cross-cloud edge, which
		// the split's own problems and notes describe, not a dropped one; an
		// unknown exclusion is noted once, on the primary.
		var notes []connectors.Problem
		for _, n := range rep.Notes {
			switch {
			case n.Code == capability.CodeEdgeDropped && !real[n.EdgeSource] && !real[n.EdgeTarget]:
			case n.Code == capability.CodeUnknownExclusion && !p.Primary:
			default:
				notes = append(notes, n)
			}
		}
		rep.Notes = notes
		if rep.Notes == nil {
			rep.Notes = []connectors.Problem{}
		}
		if cc, ok := p.Connector.(connectors.CrossCloud); ok {
			rep.Problems = capability.Merge(rep.Problems, cc.CrossCheck(l.graph, p.Deployment))
		}
		rep.Ready = len(rep.Problems) == 0
		out.Clouds = append(out.Clouds, cloudReport{ConnectionID: p.ConnectionID, ConnectionName: p.Name, Connector: p.Connector.ID(), Position: i, Report: rep})
		out.Problems = capability.Merge(out.Problems, rep.Problems)
		out.Notes = capability.Merge(out.Notes, rep.Notes)
	}
	out.Ready = len(out.Problems) == 0
	return out
}

// splitPreview renders every cloud's Terraform, each file named after its
// cloud's connection ("<connection>/<file>"). No credential and no value
// another cloud would hand a VM is in it.
func splitPreview(ctx context.Context, l loaded) (map[string]string, *refusal, error) {
	out := map[string]string{}
	for _, p := range l.split.Parts {
		pv, ok := p.Connector.(connectors.Previewer)
		if !ok {
			return nil, &refusal{status: http.StatusNotImplemented, code: "no_preview",
				message: fmt.Sprintf("%s does not offer a Terraform preview.", p.Connector.Manifest().Title)}, nil
		}
		dep := p.Deployment
		dep.Connection.Credentials = nil
		files, err := pv.Preview(ctx, l.graph, dep)
		if err != nil {
			return nil, nil, err
		}
		for name, b := range files {
			out[p.Name+"/"+name] = string(b)
		}
	}
	return out, nil, nil
}
