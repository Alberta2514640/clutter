// Package store reads and writes B's tables (backend/storage-sql/adapter-tables.sql).
//
// Every query a request reaches that takes a diagram, connection, deployment
// or run id also takes the caller's organization id and filters on it, so a
// request can never read or change another organization's row. Deployments
// and runs are scoped twice: through their connection and through their
// diagram's project. The job runner's writes (RunStarted, RunFinished,
// RunParts, SetDeploymentStatus) and FailInterrupted take no organization:
// they act only on runs B itself queued, after the request that queued them
// was scoped, and no request reaches them.
package store

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/jobs"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Errors the API maps to responses.
var (
	ErrNotFound  = errors.New("not found")
	ErrExists    = errors.New("already exists")
	ErrLive      = errors.New("has live deployments")
	ErrRunActive = errors.New("a run is already queued or running")
	ErrOrgBusy   = errors.New("the organization has too many runs queued or running")
)

// Store is B's Postgres access.
type Store struct {
	Pool *pgxpool.Pool
}

// schemaLockKey serialises schema applies across B processes.
const schemaLockKey = 7_514_640_003

// ApplySchema runs B's schema file in one transaction, under an advisory lock.
func (s *Store) ApplySchema(ctx context.Context, sql string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(schemaLockKey)); err != nil {
		return err
	}
	// No arguments, so pgx sends the file over the simple protocol, which
	// accepts many statements in one call.
	if _, err := tx.Exec(ctx, sql); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Connection is a connections row.
type Connection struct {
	ID                  string
	OrganizationID      string
	ConnectorID         string // column connector_type
	Name                string
	Region              string
	Status              string
	StatusDetail        *string
	SecretRef           string
	RoleTemplateVersion *string
	CreatedBy           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	VerifiedAt          *time.Time
	// Diagrams is how many diagrams have a deployment that builds on the
	// connection, as its own or through a placement. Deleting the connection
	// removes those deployments.
	Diagrams int
}

// Connection statuses.
const (
	ConnectionWorking  = "working"
	ConnectionFailed   = "failed"
	ConnectionDeleting = "deleting"
)

// connectionColumns are read from the connections table, unaliased, in every
// statement that uses them. The diagram count reaches only deployments on
// this connection, which belong to its organization (CreateDeployment,
// insertPlacements).
const connectionColumns = `id, organization_id, connector_type, name, region, status, status_detail,
	secret_ref, role_template_version, COALESCE(created_by::text, ''), created_at, updated_at, verified_at,
	(SELECT count(DISTINCT dd.diagram_id) FROM deployments dd
		WHERE dd.connection_id = connections.id OR EXISTS (
			SELECT 1 FROM deployment_placements ddp WHERE ddp.deployment_id = dd.id AND ddp.connection_id = connections.id))`

func scanConnection(row pgx.Row) (Connection, error) {
	var c Connection
	err := row.Scan(&c.ID, &c.OrganizationID, &c.ConnectorID, &c.Name, &c.Region, &c.Status, &c.StatusDetail,
		&c.SecretRef, &c.RoleTemplateVersion, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &c.VerifiedAt, &c.Diagrams)
	if errors.Is(err, pgx.ErrNoRows) {
		return Connection{}, ErrNotFound
	}
	return c, err
}

// CreateConnection inserts a verified connection. A nil or empty
// RoleTemplateVersion is stored as NULL (not known).
func (s *Store) CreateConnection(ctx context.Context, c Connection) (Connection, error) {
	var version *string
	if c.RoleTemplateVersion != nil && *c.RoleTemplateVersion != "" {
		version = c.RoleTemplateVersion
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO connections (id, organization_id, connector_type, name, region, status, secret_ref, created_by, verified_at, role_template_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), $9)
		RETURNING `+connectionColumns,
		c.ID, c.OrganizationID, c.ConnectorID, c.Name, c.Region, c.Status, c.SecretRef, c.CreatedBy, version)
	out, err := scanConnection(row)
	if generic.IsUniqueViolation(err) {
		return Connection{}, ErrExists
	}
	return out, err
}

// ListConnections returns an organization's connections, oldest first.
func (s *Store) ListConnections(ctx context.Context, orgID string) ([]Connection, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+connectionColumns+` FROM connections
		WHERE organization_id = $1 AND status <> 'deleting' ORDER BY created_at, id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Connection{}
	for rows.Next() {
		c, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetConnection returns one of an organization's connections.
func (s *Store) GetConnection(ctx context.Context, orgID, id string) (Connection, error) {
	return scanConnection(s.Pool.QueryRow(ctx, `SELECT `+connectionColumns+` FROM connections
		WHERE id = $1 AND organization_id = $2 AND status <> 'deleting'`, id, orgID))
}

// SetConnectionStatus records the outcome of a verification.
func (s *Store) SetConnectionStatus(ctx context.Context, orgID, id, status, detail string) (Connection, error) {
	var d *string
	if detail != "" {
		d = &detail
	}
	return scanConnection(s.Pool.QueryRow(ctx, `UPDATE connections
		SET status = $3, status_detail = $4, verified_at = now()
		WHERE id = $1 AND organization_id = $2 AND status <> 'deleting'
		RETURNING `+connectionColumns, id, orgID, status, d))
}

// MarkConnectionDeleting turns a connection into a tombstone and returns it.
// It refuses with ErrLive while any of its deployments is live. Creating a
// deployment or a run takes a share lock on the connection, so none can start
// between the check and the mark.
func (s *Store) MarkConnectionDeleting(ctx context.Context, orgID, id string) (Connection, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Connection{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit
	c, err := scanConnection(tx.QueryRow(ctx, `SELECT `+connectionColumns+` FROM connections
		WHERE id = $1 AND organization_id = $2 AND status <> 'deleting' FOR UPDATE`, id, orgID))
	if err != nil {
		return Connection{}, err
	}
	// Live on this connection as its primary, or through blocks placed on it.
	var live bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM deployments d
		WHERE (d.connection_id = $1 OR EXISTS (
			SELECT 1 FROM deployment_placements dp WHERE dp.deployment_id = d.id AND dp.connection_id = $1))
		AND public.deployment_is_live(d.id, d.status))`, id).Scan(&live); err != nil {
		return Connection{}, err
	}
	if live {
		return Connection{}, ErrLive
	}
	if _, err := tx.Exec(ctx, `UPDATE connections SET status = 'deleting' WHERE id = $1`, id); err != nil {
		return Connection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Connection{}, err
	}
	c.Status = ConnectionDeleting
	return c, nil
}

// DeleteConnection removes one of an organization's tombstoned connections
// and, by cascade, its deployments and runs. The database refuses it while a
// deployment is live.
func (s *Store) DeleteConnection(ctx context.Context, orgID, id string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM connections WHERE id = $1 AND organization_id = $2 AND status = 'deleting'`, id, orgID)
	if generic.IsLiveDeploymentDelete(err) {
		return ErrLive
	}
	return err
}

// Deployment is a deployments row, with what a run needs from its connection.
type Deployment struct {
	ID           string
	DiagramID    string
	ConnectionID string
	Settings     map[string]any
	Exclusions   []string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time

	ConnectorID string // the connection's connector_type
	Region      string
	SecretRef   string
	// ConnectionName is the primary connection's name.
	ConnectionName string
	// Placements are the other connections some of its blocks are placed on,
	// oldest first (ADR-020).
	Placements []Placement
}

// Placement is a deployment_placements row, with what a run needs from its
// connection, which belongs to the deployment's organization.
type Placement struct {
	ConnectionID string
	NodeIDs      []string
	Settings     map[string]any

	ConnectorID    string // the connection's connector_type
	Region         string
	SecretRef      string
	ConnectionName string
}

// q is what both a pool and a transaction offer.
type q interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// insertPlacements adds a deployment's placements. Each insert proves its
// connection belongs to orgID and is not being deleted, and takes a share
// lock on it; ErrNotFound means one does not.
func insertPlacements(ctx context.Context, tx q, orgID, deploymentID string, ps []Placement) error {
	for _, p := range ps {
		if p.Settings == nil {
			p.Settings = map[string]any{}
		}
		if p.NodeIDs == nil {
			p.NodeIDs = []string{}
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO deployment_placements (deployment_id, connection_id, node_ids, settings)
			SELECT $1, c.id, $3, $4
			FROM connections c
			WHERE c.id = $2 AND c.organization_id = $5 AND c.status <> 'deleting'
			FOR SHARE`,
			deploymentID, p.ConnectionID, p.NodeIDs, p.Settings, orgID)
		if generic.IsUniqueViolation(err) {
			return ErrExists
		}
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrNotFound
		}
	}
	return nil
}

// samePlaces reports whether next puts the same blocks on the same
// connections as current, whatever their settings.
func samePlaces(current, next []Placement) bool {
	if len(current) != len(next) {
		return false
	}
	have := map[string]string{}
	for _, p := range current {
		ids := append([]string(nil), p.NodeIDs...)
		sort.Strings(ids)
		have[p.ConnectionID] = strings.Join(ids, ",")
	}
	for _, p := range next {
		ids := append([]string(nil), p.NodeIDs...)
		sort.Strings(ids)
		if got, ok := have[p.ConnectionID]; !ok || got != strings.Join(ids, ",") {
			return false
		}
	}
	return true
}

// placementsOf returns the placements of these deployments, by deployment
// id, scoped to orgID through each placement's connection.
func placementsOf(ctx context.Context, db q, orgID string, deploymentIDs []string) (map[string][]Placement, error) {
	out := map[string][]Placement{}
	if len(deploymentIDs) == 0 {
		return out, nil
	}
	rows, err := db.Query(ctx, `
		SELECT dp.deployment_id, dp.connection_id, dp.node_ids, dp.settings, c.connector_type, c.region, c.secret_ref, c.name
		FROM deployment_placements dp
		JOIN connections c ON c.id = dp.connection_id
		WHERE dp.deployment_id = ANY($1::uuid[]) AND c.organization_id = $2 AND c.status <> 'deleting'
		ORDER BY dp.created_at, dp.connection_id`, deploymentIDs, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var depID string
		var p Placement
		if err := rows.Scan(&depID, &p.ConnectionID, &p.NodeIDs, &p.Settings, &p.ConnectorID, &p.Region, &p.SecretRef, &p.ConnectionName); err != nil {
			return nil, err
		}
		out[depID] = append(out[depID], p)
	}
	return out, rows.Err()
}

// CreateDeployment binds a diagram to one of the organization's connections,
// with its blocks placed on others. The caller has proved the diagram belongs
// to orgID; the inserts prove every connection does. It returns ErrNotFound
// for another organization's connection and ErrExists for a second deployment
// of the same pair.
func (s *Store) CreateDeployment(ctx context.Context, orgID, createdBy string, d Deployment) (Deployment, error) {
	if d.Settings == nil {
		d.Settings = map[string]any{}
	}
	if d.Exclusions == nil {
		d.Exclusions = []string{}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Deployment{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit
	row := tx.QueryRow(ctx, `
		INSERT INTO deployments (id, diagram_id, connection_id, settings, exclusions, status, created_by)
		SELECT $1, $2, c.id, $4, $5, 'new', $6
		FROM connections c
		WHERE c.id = $3 AND c.organization_id = $7 AND c.status <> 'deleting'
		FOR SHARE
		RETURNING status, created_at, updated_at`,
		d.ID, d.DiagramID, d.ConnectionID, d.Settings, d.Exclusions, createdBy, orgID)
	err = row.Scan(&d.Status, &d.CreatedAt, &d.UpdatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Deployment{}, ErrNotFound
	case generic.IsUniqueViolation(err):
		return Deployment{}, ErrExists
	case err != nil:
		return Deployment{}, err
	}
	if err := insertPlacements(ctx, tx, orgID, d.ID, d.Placements); err != nil {
		return Deployment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Deployment{}, err
	}
	return s.GetDeployment(ctx, orgID, d.ID)
}

// orgScope joins a deployment d to its connection c and its diagram's project
// p, and requires both to belong to the organization $2.
const orgScope = `
	JOIN connections c ON c.id = d.connection_id
	JOIN diagrams g ON g.id = d.diagram_id
	JOIN projects p ON p.id = g.project_id
	WHERE c.organization_id = $2 AND p.organization_id = $2`

// GetDeployment returns one of an organization's deployments.
func (s *Store) GetDeployment(ctx context.Context, orgID, id string) (Deployment, error) {
	var d Deployment
	err := s.Pool.QueryRow(ctx, `
		SELECT d.id, d.diagram_id, d.connection_id, d.settings, d.exclusions, d.status, d.created_at, d.updated_at,
		       c.connector_type, c.region, c.secret_ref, c.name
		FROM deployments d`+orgScope+` AND d.id = $1 AND c.status <> 'deleting'`, id, orgID).
		Scan(&d.ID, &d.DiagramID, &d.ConnectionID, &d.Settings, &d.Exclusions, &d.Status, &d.CreatedAt, &d.UpdatedAt,
			&d.ConnectorID, &d.Region, &d.SecretRef, &d.ConnectionName)
	if errors.Is(err, pgx.ErrNoRows) {
		return Deployment{}, ErrNotFound
	}
	if err != nil {
		return Deployment{}, err
	}
	ps, err := placementsOf(ctx, s.Pool, orgID, []string{d.ID})
	if err != nil {
		return Deployment{}, err
	}
	d.Placements = ps[d.ID]
	return d, nil
}

// DeploymentSummary is a deployment as the deploy panel lists it: with its
// latest run, if any.
type DeploymentSummary struct {
	Deployment
	LatestRun *Run
	// LastApply is the deployment's latest apply that completed, or nil.
	LastApply *Run
	// DiagramVersion is the version of the diagram as saved now (the version
	// DiagramDocument reads), so it can be compared with LastApply's.
	DiagramVersion *int
}

// nullableRun is a run read through a LEFT JOIN, whose columns are all null
// when the deployment has no such run.
type nullableRun struct {
	r                                      Run
	id, deployment, action, status, logKey *string
	created                                *time.Time
}

// run returns the run, or nil when the join found none.
func (n nullableRun) run() *Run {
	if n.id == nil || n.deployment == nil || n.action == nil || n.status == nil || n.logKey == nil || n.created == nil {
		return nil
	}
	r := n.r
	r.ID, r.DeploymentID, r.Action, r.Status, r.LogKey, r.CreatedAt = *n.id, *n.deployment, *n.action, *n.status, *n.logKey, *n.created
	return &r
}

// ListDeployments returns one diagram's deployments in an organization, oldest
// first. The caller has proved the diagram belongs to orgID; the query still
// scopes each row through its connection and its diagram's project.
func (s *Store) ListDeployments(ctx context.Context, orgID, diagramID string) ([]DeploymentSummary, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT d.id, d.diagram_id, d.connection_id, d.settings, d.exclusions, d.status, d.created_at, d.updated_at,
		       c.connector_type, c.region, c.name,
		       lr.id, lr.deployment_id, lr.action, lr.status, lr.diagram_version, lr.failure, lr.log_key,
		       lr.created_at, lr.started_at, lr.finished_at, lr.parts,
		       la.id, la.deployment_id, la.action, la.status, la.diagram_version, la.failure, la.log_key,
		       la.created_at, la.started_at, la.finished_at, la.parts,
		       (SELECT COALESCE(max(h.version), 0) + 1 FROM diagram_history h WHERE h.diagram_id = d.diagram_id)
		FROM deployments d
		JOIN connections c ON c.id = d.connection_id
		JOIN diagrams g ON g.id = d.diagram_id
		JOIN projects p ON p.id = g.project_id
		LEFT JOIN LATERAL (
			SELECT r.* FROM runs r WHERE r.deployment_id = d.id ORDER BY r.created_at DESC, r.id DESC LIMIT 1
		) lr ON true
		LEFT JOIN LATERAL (
			SELECT r.* FROM runs r WHERE r.deployment_id = d.id AND r.action = 'apply' AND r.status = 'completed'
			ORDER BY r.created_at DESC, r.id DESC LIMIT 1
		) la ON true
		WHERE d.diagram_id = $1 AND c.organization_id = $2 AND p.organization_id = $2 AND c.status <> 'deleting'
		ORDER BY d.created_at, d.id`, diagramID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeploymentSummary{}
	for rows.Next() {
		var d DeploymentSummary
		var latest, last nullableRun
		if err := rows.Scan(&d.ID, &d.DiagramID, &d.ConnectionID, &d.Settings, &d.Exclusions, &d.Status, &d.CreatedAt, &d.UpdatedAt,
			&d.ConnectorID, &d.Region, &d.ConnectionName,
			&latest.id, &latest.deployment, &latest.action, &latest.status, &latest.r.DiagramVersion, &latest.r.Failure, &latest.logKey,
			&latest.created, &latest.r.StartedAt, &latest.r.FinishedAt, &latest.r.Parts,
			&last.id, &last.deployment, &last.action, &last.status, &last.r.DiagramVersion, &last.r.Failure, &last.logKey,
			&last.created, &last.r.StartedAt, &last.r.FinishedAt, &last.r.Parts,
			&d.DiagramVersion); err != nil {
			return nil, err
		}
		d.LatestRun = latest.run()
		d.LastApply = last.run()
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	ids := make([]string, len(out))
	for i, d := range out {
		ids[i] = d.ID
	}
	ps, err := placementsOf(ctx, s.Pool, orgID, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Placements = ps[out[i].ID]
	}
	return out, nil
}

// UpdateDeployment replaces one of an organization's deployments' settings and
// exclusions, and its placements when placements is not nil. It returns
// ErrRunActive while a run of the deployment is queued or running, so a run
// never sees its deployment change under it, and ErrLive for placements that
// move a block while the deployment may hold resources: a cloud's state must
// not be left without its blocks. Placements that change only a placed cloud's
// settings are allowed then. The row lock also holds back a run being inserted (its
// foreign key needs a share lock on the deployment) until the change commits.
// A placement on another organization's connection is ErrNotFound.
func (s *Store) UpdateDeployment(ctx context.Context, orgID, id string, settings map[string]any, exclusions []string, placements *[]Placement) (Deployment, error) {
	if settings == nil {
		settings = map[string]any{}
	}
	if exclusions == nil {
		exclusions = []string{}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Deployment{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit
	var found string
	var live bool
	err = tx.QueryRow(ctx, `SELECT d.id, public.deployment_is_live(d.id, d.status) FROM deployments d`+orgScope+` AND d.id = $1 AND c.status <> 'deleting'
		FOR UPDATE OF d`, id, orgID).Scan(&found, &live)
	if errors.Is(err, pgx.ErrNoRows) {
		return Deployment{}, ErrNotFound
	}
	if err != nil {
		return Deployment{}, err
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM runs WHERE deployment_id = $1 AND status IN ('queued', 'running'))`,
		found).Scan(&active); err != nil {
		return Deployment{}, err
	}
	if active {
		return Deployment{}, ErrRunActive
	}
	if _, err := tx.Exec(ctx, `UPDATE deployments SET settings = $2, exclusions = $3 WHERE id = $1`, found, settings, exclusions); err != nil {
		return Deployment{}, err
	}
	if placements != nil {
		current, err := placementsOf(ctx, tx, orgID, []string{found})
		if err != nil {
			return Deployment{}, err
		}
		switch {
		case samePlaces(current[found], *placements):
			// Only settings change, which a live deployment allows, as it
			// does for its own connection's settings.
			for _, p := range *placements {
				if p.Settings == nil {
					p.Settings = map[string]any{}
				}
				if _, err := tx.Exec(ctx, `UPDATE deployment_placements SET settings = $3 WHERE deployment_id = $1 AND connection_id = $2`,
					found, p.ConnectionID, p.Settings); err != nil {
					return Deployment{}, err
				}
			}
		case live:
			return Deployment{}, ErrLive
		default:
			if _, err := tx.Exec(ctx, `DELETE FROM deployment_placements WHERE deployment_id = $1`, found); err != nil {
				return Deployment{}, err
			}
			if err := insertPlacements(ctx, tx, orgID, found, *placements); err != nil {
				return Deployment{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Deployment{}, err
	}
	return s.GetDeployment(ctx, orgID, found)
}

// DeleteDeployment removes one of an organization's deployments and, by
// cascade, its runs. Only a deployment that holds nothing in the user's cloud
// can go: it returns ErrLive unless the deployment is new or destroyed, and
// ErrRunActive while a run of it is queued or running. The row lock holds back
// a run being inserted (its foreign key needs a share lock on the deployment)
// until the delete commits; the database's own trigger refuses a live delete too.
func (s *Store) DeleteDeployment(ctx context.Context, orgID, id string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit
	var found string
	var live bool
	err = tx.QueryRow(ctx, `SELECT d.id, public.deployment_is_live(d.id, d.status) FROM deployments d`+orgScope+` AND d.id = $1 AND c.status <> 'deleting'
		FOR UPDATE OF d`, id, orgID).Scan(&found, &live)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM runs WHERE deployment_id = $1 AND status IN ('queued', 'running'))`,
		found).Scan(&active); err != nil {
		return err
	}
	if active {
		return ErrRunActive
	}
	if live {
		return ErrLive
	}
	if _, err := tx.Exec(ctx, `DELETE FROM deployments WHERE id = $1`, found); err != nil {
		if generic.IsLiveDeploymentDelete(err) {
			return ErrLive
		}
		return err
	}
	return tx.Commit(ctx)
}

// DiagramDocument returns the saved document of a diagram in one of the
// organization's projects, and the version of that document. A's save writes
// the document it replaces to diagram_history as the next version, so history
// row N holds the document before save N, and the saved document is one past
// the latest row (1 for a diagram never saved again): the row it becomes on
// the next save.
func (s *Store) DiagramDocument(ctx context.Context, orgID, diagramID string) ([]byte, *int, error) {
	var data []byte
	var version *int
	err := s.Pool.QueryRow(ctx, `SELECT g.data::text, (SELECT COALESCE(max(version), 0) + 1 FROM diagram_history WHERE diagram_id = $1)
		FROM diagrams g JOIN projects p ON p.id = g.project_id
		WHERE g.id = $1 AND p.organization_id = $2`, diagramID, orgID).Scan(&data, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	return data, version, err
}

// Run is a runs row.
type Run struct {
	ID             string
	DeploymentID   string
	Action         string
	Status         string
	DiagramVersion *int
	Failure        *string
	LogKey         string
	CreatedAt      time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
	// Parts is where each cloud of a split run stands, or nil.
	Parts []jobs.PartProgress
}

const runColumns = `r.id, r.deployment_id, r.action, r.status, r.diagram_version, r.failure, r.log_key,
	r.created_at, r.started_at, r.finished_at, r.parts`

func scanRun(row pgx.Row) (Run, error) {
	var r Run
	err := row.Scan(&r.ID, &r.DeploymentID, &r.Action, &r.Status, &r.DiagramVersion, &r.Failure, &r.LogKey,
		&r.CreatedAt, &r.StartedAt, &r.FinishedAt, &r.Parts)
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	return r, err
}

// runLockClass is the first key of the advisory lock that makes one
// organization's run creations take turns (the second is the org id's hash).
const runLockClass int32 = 75_146_401

// CreateRun records a queued run of one of the organization's deployments.
// It returns ErrOrgBusy while the organization already has maxActive runs
// queued or running, and ErrRunActive while another run of the deployment is
// queued or running. An advisory lock per organization, held to the end of
// the transaction, stops two requests from both passing the cap.
func (s *Store) CreateRun(ctx context.Context, orgID, createdBy string, r Run, maxActive int) (Run, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, runLockClass, orgID); err != nil {
		return Run{}, err
	}
	var active int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM runs r
		JOIN deployments d ON d.id = r.deployment_id
		JOIN connections c ON c.id = d.connection_id
		WHERE c.organization_id = $1 AND r.status IN ('queued', 'running')`, orgID).Scan(&active); err != nil {
		return Run{}, err
	}
	if active >= maxActive {
		return Run{}, ErrOrgBusy
	}
	// The connections the deployment's blocks are placed on cannot be marked
	// for deletion while the run is created, as its own connection cannot.
	var placedGone bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM deployment_placements dp WHERE dp.deployment_id = $1 AND NOT EXISTS (
			SELECT 1 FROM connections c WHERE c.id = dp.connection_id AND c.organization_id = $2 AND c.status <> 'deleting'))`,
		r.DeploymentID, orgID).Scan(&placedGone); err != nil {
		return Run{}, err
	}
	if placedGone {
		return Run{}, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `SELECT 1 FROM connections c JOIN deployment_placements dp ON dp.connection_id = c.id
		WHERE dp.deployment_id = $1 AND c.organization_id = $2 FOR SHARE OF c`, r.DeploymentID, orgID); err != nil {
		return Run{}, err
	}
	row := tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO runs (id, deployment_id, action, status, diagram_version, log_key, created_by)
			SELECT $1, d.id, $3, 'queued', $4, $5, $6
			FROM deployments d`+orgScope+` AND d.id = $7 AND c.status <> 'deleting'
			FOR SHARE OF c
			RETURNING *
		)
		SELECT `+runColumns+` FROM ins r`,
		r.ID, orgID, r.Action, r.DiagramVersion, r.LogKey, createdBy, r.DeploymentID)
	out, err := scanRun(row)
	if generic.IsUniqueViolation(err) {
		return Run{}, ErrRunActive
	}
	if err != nil {
		return Run{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	return out, nil
}

// GetRun returns one of an organization's runs.
func (s *Store) GetRun(ctx context.Context, orgID, id string) (Run, error) {
	return scanRun(s.Pool.QueryRow(ctx, `SELECT `+runColumns+`
		FROM runs r JOIN deployments d ON d.id = r.deployment_id`+orgScope+` AND r.id = $1`, id, orgID))
}

// RunStarted implements jobs.Store.
func (s *Store) RunStarted(ctx context.Context, runID string, at time.Time) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE runs SET status = 'running', started_at = $2 WHERE id = $1 AND status = 'queued'`, runID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("the run is not queued")
	}
	return nil
}

// RunFinished implements jobs.Store.
func (s *Store) RunFinished(ctx context.Context, runID string, status jobs.Status, failure string, at time.Time) error {
	var f *string
	if failure != "" {
		f = &failure
	}
	_, err := s.Pool.Exec(ctx, `UPDATE runs SET status = $2, failure = $3, finished_at = $4 WHERE id = $1`,
		runID, string(status), f, at)
	return err
}

// RunParts implements jobs.PartRecorder.
func (s *Store) RunParts(ctx context.Context, runID string, parts []jobs.PartProgress) error {
	_, err := s.Pool.Exec(ctx, `UPDATE runs SET parts = $2 WHERE id = $1`, runID, parts)
	return err
}

var _ jobs.PartRecorder = (*Store)(nil)

// SetDeploymentStatus implements jobs.Store.
func (s *Store) SetDeploymentStatus(ctx context.Context, deploymentID string, status jobs.DeploymentStatus) error {
	_, err := s.Pool.Exec(ctx, `UPDATE deployments SET status = $2 WHERE id = $1`, deploymentID, string(status))
	return err
}

// FailInterrupted fails every run a stopped process left queued or running.
// A deployment whose apply or destroy was cut short counts as failed, since it
// may hold resources. It returns how many runs it failed.
func (s *Store) FailInterrupted(ctx context.Context, failure string) (int64, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit
	if _, err := tx.Exec(ctx, `UPDATE deployments SET status = 'failed' WHERE id IN (
		SELECT deployment_id FROM runs WHERE status = 'running' AND action IN ('apply', 'destroy'))`); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `UPDATE runs SET status = 'failed', failure = $1, finished_at = now()
		WHERE status IN ('queued', 'running')`, failure)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), tx.Commit(ctx)
}
