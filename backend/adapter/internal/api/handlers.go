package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/examples"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/jobs"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/split"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/store"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/translate"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/google/uuid"
)

// ---- catalog ----

type catalogEntry struct {
	Manifest    connectors.Manifest       `json:"manifest"`
	Credentials connectors.CredentialSpec `json:"credentials"`
	// HasLinkedAccounts is true when the connector offers linked accounts
	// (connectors.Linker), so a connection can be made from one and the
	// linked-accounts route lists them.
	HasLinkedAccounts bool `json:"has_linked_accounts"`
}

func (a *API) catalog(w http.ResponseWriter, _ *http.Request) {
	out := []catalogEntry{}
	for _, c := range a.Registry.All() {
		_, hasLinked := c.(connectors.Linker)
		out = append(out, catalogEntry{Manifest: c.Manifest(), Credentials: c.Credentials(), HasLinkedAccounts: hasLinked})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// examples lists the built-in example diagrams this build can deploy: each
// keeps only the clouds the registry offers.
func (a *API) examples(w http.ResponseWriter, r *http.Request) {
	all, err := examples.All()
	if err != nil {
		a.internal(w, r, "read the examples", err)
		return
	}
	items := examples.Fitting(all, func(id string) bool {
		_, ok := a.Registry.Get(id)
		return ok
	})
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// listLinkedAccounts lists the accounts the organization linked outside the
// adapter layer for one connector, by name only.
func (a *API) listLinkedAccounts(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	c, found := a.Registry.Get(r.PathValue("connectorID"))
	if !found {
		notFound(w)
		return
	}
	l, ok := c.(connectors.Linker)
	if !ok {
		writeError(w, http.StatusNotImplemented, "no_linking", "This cloud has no linked accounts.")
		return
	}
	accounts, err := l.Linked(r.Context(), orgID)
	if err != nil {
		a.internal(w, r, "list linked accounts", err)
		return
	}
	if accounts == nil {
		accounts = []connectors.LinkedAccount{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": accounts})
}

// ---- connections ----

type connectionView struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	Connector      string     `json:"connector"`
	Name           string     `json:"name"`
	Region         string     `json:"region"`
	Status         string     `json:"status"`
	StatusDetail   *string    `json:"status_detail"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	VerifiedAt     *time.Time `json:"verified_at"`
	// Diagrams is how many diagrams deploy to it; deleting it removes those
	// deployments.
	Diagrams int `json:"diagrams"`
}

// connectionOut never carries the secret reference or any credential.
func connectionOut(c store.Connection) connectionView {
	return connectionView{
		ID: c.ID, OrganizationID: c.OrganizationID, Connector: c.ConnectorID, Name: c.Name, Region: c.Region,
		Status: c.Status, StatusDetail: c.StatusDetail, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, VerifiedAt: c.VerifiedAt,
		Diagrams: c.Diagrams,
	}
}

func (a *API) listConnections(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	cs, err := a.Store.ListConnections(r.Context(), orgID)
	if err != nil {
		a.internal(w, r, "list connections", err)
		return
	}
	out := make([]connectionView, 0, len(cs))
	for _, c := range cs {
		out = append(out, connectionOut(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// createConnectionBody carries either pasted credentials or the id of a
// linked account (connectors.Linker), never both.
type createConnectionBody struct {
	Connector   string            `json:"connector"`
	Name        string            `json:"name"`
	Region      string            `json:"region"`
	Credentials map[string]string `json:"credentials"`
	LinkedID    string            `json:"linked_id"`
}

func (a *API) verify(ctx context.Context, c connectors.Connector, conn connectors.Connection) error {
	t := a.VerifyTimeout
	if t == 0 {
		t = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()
	return c.VerifyConnection(ctx, conn)
}

const verifyFailed = "The cloud refused these credentials. Check them and try again."

func (a *API) createConnection(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	var body createConnectionBody
	if !decode(w, r, &body) {
		return
	}
	c, found := a.Registry.Get(body.Connector)
	if !found {
		writeError(w, http.StatusUnprocessableEntity, "unknown_connector", "This cloud is not available.")
		return
	}
	name, ok := cleanName(body.Name)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "invalid_name", "The name must be 1 to 64 characters of plain text.")
		return
	}
	if !c.Manifest().AllowsRegion(body.Region) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_region", "This cloud has no such region.")
		return
	}
	spec := c.Credentials()
	// A linked-only cloud never takes pasted credentials: they could be
	// another organization's link (ADR-019).
	if spec.LinkedOnly && (body.LinkedID == "" || len(body.Credentials) > 0) {
		writeError(w, http.StatusBadRequest, "linked_only",
			"This cloud connects only through an account linked to this organization. Link the account first, then choose it here.")
		return
	}
	supplied := body.Credentials
	var templateVersion *string
	if body.LinkedID != "" {
		if len(body.Credentials) > 0 {
			writeError(w, http.StatusUnprocessableEntity, "invalid_credentials", "Send either credentials or a linked account, not both.")
			return
		}
		l, hasLinked := c.(connectors.Linker)
		if !hasLinked {
			writeError(w, http.StatusUnprocessableEntity, "no_linking", "This cloud has no linked accounts.")
			return
		}
		lc, err := l.LinkedSecrets(r.Context(), orgID, body.LinkedID)
		if errors.Is(err, connectors.ErrLinkedNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "This linked account was not found.")
			return
		}
		if err != nil {
			a.internal(w, r, "read linked account", err)
			return
		}
		supplied = lc.Secrets
		if v := lc.TemplateVersion; v != "" && len(v) <= connectors.MaxTemplateVersion {
			templateVersion = &v
		}
	}
	if err := spec.CheckSupplied(supplied); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_credentials", connectors.UserMessage(err, "The credentials are not complete."))
		return
	}
	secrets := connectors.Secrets(supplied)
	id := uuid.NewString()
	conn := connectors.Connection{ID: id, ConnectorID: c.ID(), Region: body.Region, Credentials: secrets}
	if err := a.verify(r.Context(), c, conn); err != nil {
		a.Logger.Info("connection verification refused", "connector", c.ID(), "org", orgID)
		writeError(w, http.StatusUnprocessableEntity, "verification_failed", connectors.UserMessage(err, verifyFailed))
		return
	}
	ref, err := a.Credentials.Put(r.Context(), id, secrets)
	if err != nil {
		a.internal(w, r, "store connection credentials", err)
		return
	}
	row, err := a.Store.CreateConnection(r.Context(), store.Connection{
		ID: id, OrganizationID: orgID, ConnectorID: c.ID(), Name: name, Region: body.Region,
		Status: store.ConnectionWorking, SecretRef: ref, CreatedBy: userID, RoleTemplateVersion: templateVersion,
	})
	if err != nil {
		if derr := a.Credentials.Delete(context.WithoutCancel(r.Context()), ref); derr != nil {
			a.Logger.Error("could not remove the credentials of an unsaved connection", "connection_id", id, "err", derr)
		}
		if errors.Is(err, store.ErrExists) {
			writeError(w, http.StatusConflict, "name_taken", "A connection with this name already exists.")
			return
		}
		a.internal(w, r, "create connection", err)
		return
	}
	writeJSON(w, http.StatusCreated, connectionOut(row))
}

func (a *API) verifyConnection(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "connectionID")
	if !ok {
		return
	}
	row, err := a.Store.GetConnection(r.Context(), orgID, id)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		a.internal(w, r, "get connection", err)
		return
	}
	c, found := a.Registry.Get(row.ConnectorID)
	if !found {
		writeError(w, http.StatusUnprocessableEntity, "unknown_connector", "This cloud is not available.")
		return
	}
	secrets, err := a.Credentials.Get(r.Context(), row.SecretRef)
	if err != nil {
		a.internal(w, r, "read connection credentials", err)
		return
	}
	status, detail := store.ConnectionWorking, ""
	if err := a.verify(r.Context(), c, connectors.Connection{ID: row.ID, ConnectorID: c.ID(), Region: row.Region, Credentials: secrets}); err != nil {
		status, detail = store.ConnectionFailed, connectors.UserMessage(err, verifyFailed)
	}
	row, err = a.Store.SetConnectionStatus(context.WithoutCancel(r.Context()), orgID, id, status, detail)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		a.internal(w, r, "record verification", err)
		return
	}
	writeJSON(w, http.StatusOK, connectionOut(row))
}

func (a *API) deleteConnection(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "connectionID")
	if !ok {
		return
	}
	row, err := a.Store.MarkConnectionDeleting(r.Context(), orgID, id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(w)
		return
	case errors.Is(err, store.ErrLive):
		writeError(w, http.StatusConflict, "live_deployments", "This connection has live deployments. Destroy them first.")
		return
	case err != nil:
		a.internal(w, r, "mark connection deleting", err)
		return
	}
	// The connection is now hidden. If a step below fails, the tombstone keeps
	// the secret reference so the clean-up can finish later.
	ctx := context.WithoutCancel(r.Context())
	if err := a.Credentials.Delete(ctx, row.SecretRef); err != nil {
		a.internal(w, r, "delete connection credentials", err)
		return
	}
	if err := a.Store.DeleteConnection(ctx, orgID, id); err != nil {
		a.internal(w, r, "delete connection", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- deployments ----

type deploymentView struct {
	ID           string         `json:"id"`
	DiagramID    string         `json:"diagram_id"`
	ConnectionID string         `json:"connection_id"`
	Settings     map[string]any `json:"settings"`
	Exclusions   []string       `json:"exclusions"`
	// Placements are the other clouds some of its blocks sit on (ADR-020).
	Placements []placementView `json:"placements"`
	Status     string          `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func deploymentOut(d store.Deployment) deploymentView {
	return deploymentView{ID: d.ID, DiagramID: d.DiagramID, ConnectionID: d.ConnectionID, Settings: d.Settings,
		Exclusions: d.Exclusions, Placements: placementsOut(d.Placements), Status: d.Status, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

// createDeploymentBody is the body of a new deployment. Its diagram is the one
// in the path.
type createDeploymentBody struct {
	ConnectionID string          `json:"connection_id"`
	Settings     map[string]any  `json:"settings"`
	Exclusions   []string        `json:"exclusions"`
	Placements   []placementBody `json:"placements"`
}

const maxExclusions = translate.MaxNodes

func (a *API) createDeployment(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	var body createDeploymentBody
	if !decode(w, r, &body) {
		return
	}
	// The diagram belongs to a project of this organization.
	ids, err := generic.CheckOwnedIDsPSQL(r.Context(), a.Store.Pool, userID, orgID, "", r.PathValue("diagramID"))
	if err != nil {
		a.authError(w, r, err)
		return
	}
	diagramID := ids.DiagramID
	connectionID, ok := generic.CanonicalUuid(body.ConnectionID)
	if !ok {
		notFound(w)
		return
	}
	exclusions, ok := cleanExclusions(body.Exclusions)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "invalid_exclusions", "Exclusions must be a list of block ids.")
		return
	}
	doc, _, err := a.Store.DiagramDocument(r.Context(), orgID, diagramID)
	if err != nil {
		a.internal(w, r, "read diagram", err)
		return
	}
	g, err := translate.Translate(doc)
	if err != nil {
		a.translateError(w, r, err)
		return
	}
	// The settings are checked against the deployment's own cloud, as a
	// change to them is (updateDeployment). Another organization's
	// connection, or none, is not found.
	conn, err := a.Store.GetConnection(r.Context(), orgID, connectionID)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		a.internal(w, r, "get connection", err)
		return
	}
	c, found := a.Registry.Get(conn.ConnectorID)
	if !found {
		writeError(w, http.StatusUnprocessableEntity, "unknown_connector", "This cloud is not available.")
		return
	}
	settings, problems := cleanSettings(c.Manifest(), body.Settings)
	if len(problems) > 0 {
		writeProblems(w, http.StatusUnprocessableEntity, "invalid_settings", "Some settings are not valid for this cloud.", problems)
		return
	}
	placements, rf, err := a.cleanPlacements(r.Context(), orgID, connectionID, body.Placements, g)
	if err != nil {
		a.internal(w, r, "check placements", err)
		return
	}
	if rf != nil {
		rf.write(w)
		return
	}
	// The inserts prove again that the diagram and every connection belong to
	// this organization.
	d, err := a.Store.CreateDeployment(r.Context(), orgID, userID, store.Deployment{
		ID: uuid.NewString(), DiagramID: diagramID, ConnectionID: connectionID,
		Settings: settings, Exclusions: exclusions, Placements: placements,
	})
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(w)
	case errors.Is(err, store.ErrExists):
		writeError(w, http.StatusConflict, "deployment_exists", "This diagram already has a deployment on this connection.")
	case err != nil:
		a.internal(w, r, "create deployment", err)
	default:
		writeJSON(w, http.StatusCreated, deploymentOut(d))
	}
}

func cleanExclusions(in []string) ([]string, bool) {
	if len(in) > maxExclusions {
		return nil, false
	}
	seen := map[string]bool{}
	out := []string{}
	for _, x := range in {
		if !translate.NodeIDPattern.MatchString(x) {
			return nil, false
		}
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out, true
}

func (a *API) translateError(w http.ResponseWriter, r *http.Request, err error) {
	var te *translate.Error
	switch {
	case errors.Is(err, translate.ErrNotBlocks):
		writeError(w, http.StatusUnprocessableEntity, "not_blocks", "This diagram is not drawn in building blocks.")
	case errors.As(err, &te):
		writeError(w, http.StatusUnprocessableEntity, "invalid_diagram", te.Message)
	default:
		a.internal(w, r, "translate diagram", err)
	}
}

// loaded is a deployment with its connector and, unless asked not to, its
// graph. split is set for a deployment with blocks placed on other clouds.
type loaded struct {
	row       store.Deployment
	connector connectors.Connector
	graph     connectors.Graph
	version   *int
	dep       connectors.Deployment
	split     *split.Plan
}

func (a *API) loadDeployment(w http.ResponseWriter, r *http.Request, orgID string, withGraph bool) (loaded, bool) {
	id, ok := pathID(w, r, "deploymentID")
	if !ok {
		return loaded{}, false
	}
	row, err := a.Store.GetDeployment(r.Context(), orgID, id)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return loaded{}, false
	}
	if err != nil {
		a.internal(w, r, "get deployment", err)
		return loaded{}, false
	}
	c, found := a.Registry.Get(row.ConnectorID)
	if !found {
		writeError(w, http.StatusUnprocessableEntity, "unknown_connector", "This cloud is not available.")
		return loaded{}, false
	}
	l := loaded{row: row, connector: c, dep: connectors.Deployment{
		ID: row.ID, Prefix: connectors.PrefixFor(row.ID),
		Connection: connectors.Connection{ID: row.ConnectionID, ConnectorID: row.ConnectorID, Region: row.Region},
		Settings:   row.Settings, Exclusions: row.Exclusions,
	}}
	if withGraph {
		doc, version, err := a.Store.DiagramDocument(r.Context(), orgID, row.DiagramID)
		if err != nil {
			a.internal(w, r, "read diagram", err)
			return loaded{}, false
		}
		g, err := translate.Translate(doc)
		if err != nil {
			a.translateError(w, r, err)
			return loaded{}, false
		}
		if err := a.attachCode(r.Context(), orgID, row.DiagramID, &g); err != nil {
			a.internal(w, r, "read function code", err)
			return loaded{}, false
		}
		l.graph, l.version = g, version
	}
	if len(row.Placements) > 0 {
		plan, rf := a.splitOf(row, c, l.graph)
		if rf != nil {
			rf.write(w)
			return loaded{}, false
		}
		l.split = plan
	}
	return l, true
}

// compatibilityOf is the capability check plus the connector's own checks,
// for every cloud of a split deployment.
func compatibilityOf(l loaded) compatibilityView {
	if l.split != nil {
		return splitCompatibility(l)
	}
	return compatibilityView{Report: capability.CheckConnector(l.connector, l.graph, l.dep)}
}

func (a *API) compatibility(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	l, ok := a.loadDeployment(w, r, orgID, true)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, compatibilityOf(l))
}

// deploymentListItem is one row of a diagram's deploy panel.
type deploymentListItem struct {
	deploymentView
	Connector      string   `json:"connector"`
	ConnectionName string   `json:"connection_name"`
	Region         string   `json:"region"`
	LatestRun      *runView `json:"latest_run"`
	// LastApply is its latest completed apply, whose results "What was built"
	// shows, or null.
	LastApply *runView `json:"last_apply"`
	// DiagramVersion is the version of the diagram as saved now; newer than
	// LastApply's diagram_version means changes not deployed yet.
	DiagramVersion *int `json:"diagram_version"`
}

func (a *API) listDiagramDeployments(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	// The diagram belongs to a project of this organization.
	ids, err := generic.CheckOwnedIDsPSQL(r.Context(), a.Store.Pool, userID, orgID, "", r.PathValue("diagramID"))
	if err != nil {
		a.authError(w, r, err)
		return
	}
	ds, err := a.Store.ListDeployments(r.Context(), orgID, ids.DiagramID)
	if err != nil {
		a.internal(w, r, "list deployments", err)
		return
	}
	out := make([]deploymentListItem, 0, len(ds))
	for _, d := range ds {
		item := deploymentListItem{deploymentView: deploymentOut(d.Deployment), Connector: d.ConnectorID,
			ConnectionName: d.Deployment.ConnectionName, Region: d.Region, DiagramVersion: d.DiagramVersion}
		if d.LatestRun != nil {
			run := runOut(*d.LatestRun)
			item.LatestRun = &run
		}
		if d.LastApply != nil {
			run := runOut(*d.LastApply)
			item.LastApply = &run
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// updateDeploymentBody replaces whichever of the three it holds.
type updateDeploymentBody struct {
	Settings   *map[string]any  `json:"settings"`
	Exclusions *[]string        `json:"exclusions"`
	Placements *[]placementBody `json:"placements"`
}

// maxSettings caps how many settings one request may name.
const maxSettings = 64

// cleanSettings checks settings against a manifest's deployment settings. A
// null value unsets a setting. Problems name the setting, never echo a value.
func cleanSettings(m connectors.Manifest, in map[string]any) (map[string]any, []connectors.Problem) {
	if len(in) > maxSettings {
		return nil, []connectors.Problem{{Code: connectors.CodeInvalidValue, Message: "There are too many settings."}}
	}
	specs := map[string]connectors.FieldSpec{}
	for _, f := range m.Settings {
		specs[f.Name] = f
	}
	names := make([]string, 0, len(in))
	for name := range in {
		names = append(names, name)
	}
	sort.Strings(names)
	out := map[string]any{}
	var problems []connectors.Problem
	for _, name := range names {
		f, known := specs[name]
		if !known {
			p := connectors.Problem{Code: capability.CodeUnknownField, Message: "This setting is not used by this cloud."}
			if connectors.FieldNamePattern.MatchString(name) {
				p.Field = name
			}
			problems = append(problems, p)
			continue
		}
		v := in[name]
		if v == nil {
			continue
		}
		if err := connectors.CheckValue(f, v); err != nil {
			problems = append(problems, connectors.Problem{Field: name, Code: connectors.CodeInvalidValue,
				Message: fmt.Sprintf("%s %s.", f.Title, err.Error())})
			continue
		}
		out[name] = v
	}
	return out, problems
}

func (a *API) updateDeployment(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	var body updateDeploymentBody
	if !decode(w, r, &body) {
		return
	}
	// The graph is needed only to check exclusions and placements against
	// the diagram.
	l, ok := a.loadDeployment(w, r, orgID, body.Exclusions != nil || body.Placements != nil)
	if !ok {
		return
	}
	var placements *[]store.Placement
	if body.Placements != nil {
		ps, rf, err := a.cleanPlacements(r.Context(), orgID, l.row.ConnectionID, *body.Placements, l.graph)
		if err != nil {
			a.internal(w, r, "check placements", err)
			return
		}
		if rf != nil {
			rf.write(w)
			return
		}
		placements = &ps
	}
	settings, exclusions := l.row.Settings, l.row.Exclusions
	if body.Settings != nil {
		s, problems := cleanSettings(l.connector.Manifest(), *body.Settings)
		if len(problems) > 0 {
			writeProblems(w, http.StatusUnprocessableEntity, "invalid_settings", "Some settings are not valid for this cloud.", problems)
			return
		}
		settings = s
	}
	if body.Exclusions != nil {
		x, ok := cleanExclusions(*body.Exclusions)
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "invalid_exclusions", "Exclusions must be a list of block ids.")
			return
		}
		for _, id := range x {
			if _, found := l.graph.Blocks[id]; !found {
				writeError(w, http.StatusUnprocessableEntity, "unknown_exclusion", "A left-out block is not in the diagram. Reload the diagram and try again.")
				return
			}
		}
		exclusions = x
	}
	d, err := a.Store.UpdateDeployment(r.Context(), orgID, l.row.ID, settings, exclusions, placements)
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(w)
	case errors.Is(err, store.ErrRunActive):
		writeError(w, http.StatusConflict, "run_active", "This deployment has a run in progress. Wait for it to finish, then try again.")
	case errors.Is(err, store.ErrLive):
		writeError(w, http.StatusConflict, "deployment_live",
			"Blocks move between clouds only while the deployment holds nothing. Destroy it first, then change where its blocks run.")
	case err != nil:
		a.internal(w, r, "update deployment", err)
	default:
		writeJSON(w, http.StatusOK, deploymentOut(d))
	}
}

// deleteDeployment removes a deployment that holds nothing in the user's
// cloud (new or destroyed, with no run queued or running), so its connection
// is free for a new deployment of the same diagram.
func (a *API) deleteDeployment(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "deploymentID")
	if !ok {
		return
	}
	err := a.Store.DeleteDeployment(r.Context(), orgID, id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(w)
	case errors.Is(err, store.ErrRunActive):
		writeError(w, http.StatusConflict, "run_active", "This deployment has a run in progress. Wait for it to finish, then try again.")
	case errors.Is(err, store.ErrLive):
		writeError(w, http.StatusConflict, "deployment_live", "This deployment may still have resources in your cloud. Destroy it first.")
	case err != nil:
		a.internal(w, r, "delete deployment", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// previewTimeout bounds one Preview call.
const previewTimeout = 30 * time.Second

func (a *API) preview(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	l, ok := a.loadDeployment(w, r, orgID, true)
	if !ok {
		return
	}
	p, ok := l.connector.(connectors.Previewer)
	if !ok {
		writeError(w, http.StatusNotImplemented, "no_preview", "This cloud does not offer a Terraform preview.")
		return
	}
	if rep := compatibilityOf(l); !rep.Ready {
		writeProblems(w, http.StatusUnprocessableEntity, "not_ready",
			"This cloud cannot build the diagram as it is. Fix the problems or leave those blocks out.", rep.Problems)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), previewTimeout)
	defer cancel()
	if l.split != nil {
		files, rf, err := splitPreview(ctx, l)
		switch {
		case rf != nil:
			rf.write(w)
		case err != nil:
			var ue *connectors.UserError
			if errors.As(err, &ue) {
				writeError(w, http.StatusUnprocessableEntity, "preview_failed", ue.Message)
				return
			}
			a.internal(w, r, "render preview", err)
		default:
			writeJSON(w, http.StatusOK, map[string]any{"files": files})
		}
		return
	}
	// loadDeployment never reads the secret; this keeps it that way if it ever does.
	dep := l.dep
	dep.Connection.Credentials = nil
	files, err := p.Preview(ctx, l.graph, dep)
	if err != nil {
		var ue *connectors.UserError
		if errors.As(err, &ue) {
			writeError(w, http.StatusUnprocessableEntity, "preview_failed", ue.Message)
			return
		}
		a.internal(w, r, "render preview", err)
		return
	}
	out := make(map[string]string, len(files))
	for name, b := range files {
		out[name] = string(b)
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": out})
}

// ---- runs ----

type runView struct {
	ID             string     `json:"id"`
	DeploymentID   string     `json:"deployment_id"`
	Action         string     `json:"action"`
	Status         string     `json:"status"`
	DiagramVersion *int       `json:"diagram_version"`
	Failure        *string    `json:"failure"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	// Parts is where each cloud of a split run stands, in the order the run
	// takes them; null for a run on one cloud.
	Parts []jobs.PartProgress `json:"parts"`
}

func runOut(r store.Run) runView {
	return runView{ID: r.ID, DeploymentID: r.DeploymentID, Action: r.Action, Status: r.Status, DiagramVersion: r.DiagramVersion,
		Failure: r.Failure, CreatedAt: r.CreatedAt, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt, Parts: r.Parts}
}

type createRunBody struct {
	Action connectors.Action `json:"action"`
}

const queueBusy = "Clutter is busy. Try again in a minute."

func (a *API) createRun(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	var body createRunBody
	if !decode(w, r, &body) {
		return
	}
	if !body.Action.Valid() {
		writeError(w, http.StatusUnprocessableEntity, "invalid_action", "The action must be plan, apply or destroy.")
		return
	}
	// A destroy does not read the diagram, so a later broken edit cannot stop it.
	needGraph := body.Action != connectors.ActionDestroy
	l, ok := a.loadDeployment(w, r, orgID, needGraph)
	if !ok {
		return
	}
	if needGraph {
		if rep := compatibilityOf(l); !rep.Ready {
			writeProblems(w, http.StatusUnprocessableEntity, "not_ready",
				"This cloud cannot build the diagram as it is. Fix the problems or leave those blocks out.", rep.Problems)
			return
		}
	}
	runID := uuid.NewString()
	run, err := a.Store.CreateRun(r.Context(), orgID, userID, store.Run{
		ID: runID, DeploymentID: l.row.ID, Action: string(body.Action), DiagramVersion: l.version, LogKey: logstore.KeyForRun(runID),
	}, a.maxRunsPerOrg())
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(w)
		return
	case errors.Is(err, store.ErrOrgBusy):
		writeError(w, http.StatusTooManyRequests, "too_many_runs",
			fmt.Sprintf("This organization already has %d runs queued or running. Wait for one to finish, then try again.", a.maxRunsPerOrg()))
		return
	case errors.Is(err, store.ErrRunActive):
		writeError(w, http.StatusConflict, "run_active", "This deployment already has a run in progress.")
		return
	case err != nil:
		a.internal(w, r, "create run", err)
		return
	}
	job := jobs.Job{
		RunID: run.ID, Action: body.Action, Graph: l.graph, Deployment: l.dep,
		SecretRef: l.row.SecretRef, LogKey: run.LogKey, Prior: jobs.DeploymentStatus(l.row.Status),
	}
	if l.split != nil {
		job.Parts = l.split.Parts
	}
	err = a.Jobs.Enqueue(job)
	if err != nil {
		ctx := context.WithoutCancel(r.Context())
		if ferr := a.Store.RunFinished(ctx, run.ID, jobs.StatusFailed, queueBusy, time.Now()); ferr != nil {
			a.Logger.Error("could not fail an unqueued run", "run_id", run.ID, "err", ferr)
		}
		a.Logger.Warn("could not queue a run", "run_id", run.ID, "err", err)
		writeError(w, http.StatusServiceUnavailable, "busy", queueBusy)
		return
	}
	writeJSON(w, http.StatusAccepted, runOut(run))
}

func (a *API) loadRun(w http.ResponseWriter, r *http.Request) (store.Run, bool) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return store.Run{}, false
	}
	id, ok := pathID(w, r, "runID")
	if !ok {
		return store.Run{}, false
	}
	run, err := a.Store.GetRun(r.Context(), orgID, id)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return store.Run{}, false
	}
	if err != nil {
		a.internal(w, r, "get run", err)
		return store.Run{}, false
	}
	return run, true
}

func (a *API) getRun(w http.ResponseWriter, r *http.Request) {
	run, ok := a.loadRun(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, runOut(run))
}

func (a *API) runLogs(w http.ResponseWriter, r *http.Request) {
	run, ok := a.loadRun(w, r)
	if !ok {
		return
	}
	rc, err := a.Logs.Open(run.LogKey)
	if errors.Is(err, logstore.ErrNotFound) {
		if run.Status == string(jobs.StatusQueued) {
			rc = io.NopCloser(http.NoBody)
		} else {
			notFound(w)
			return
		}
	} else if err != nil {
		a.internal(w, r, "open run log", err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		a.Logger.Warn("could not send a run log", "run_id", run.ID, "err", err)
	}
}
