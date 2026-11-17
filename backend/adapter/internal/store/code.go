package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// FunctionCode is a function_code row: the zip uploaded for one function
// block of a diagram.
type FunctionCode struct {
	DiagramID      string
	NodeID         string
	OrganizationID string
	ObjectKey      string
	FileName       string
	Size           int64
	SHA256         string
	UploadedBy     string
	UploadedAt     time.Time
}

const functionCodeColumns = `fc.diagram_id, fc.node_id, fc.organization_id, fc.object_key, fc.file_name,
	fc.size_bytes, fc.sha256, COALESCE(fc.uploaded_by::text, ''), fc.uploaded_at`

// codeScope joins a function_code row fc to its diagram's project p and
// requires both to belong to the organization $1.
const codeScope = `
	JOIN diagrams g ON g.id = fc.diagram_id
	JOIN projects p ON p.id = g.project_id
	WHERE fc.organization_id = $1 AND p.organization_id = $1`

func scanFunctionCode(row pgx.Row) (FunctionCode, error) {
	var c FunctionCode
	err := row.Scan(&c.DiagramID, &c.NodeID, &c.OrganizationID, &c.ObjectKey, &c.FileName,
		&c.Size, &c.SHA256, &c.UploadedBy, &c.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return FunctionCode{}, ErrNotFound
	}
	return c, err
}

// ListFunctionCode returns the code uploaded for one of an organization's
// diagrams, by node id order.
func (s *Store) ListFunctionCode(ctx context.Context, orgID, diagramID string) ([]FunctionCode, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+functionCodeColumns+` FROM function_code fc`+codeScope+`
		AND fc.diagram_id = $2 ORDER BY fc.node_id`, orgID, diagramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FunctionCode{}
	for rows.Next() {
		c, err := scanFunctionCode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// codeLockClass is the first key of the advisory lock that makes the code
// writes of one node take turns (the second is the node's hash).
const codeLockClass int32 = 75_146_402

// lockCodeNode takes the node's advisory lock until tx ends.
func lockCodeNode(ctx context.Context, tx pgx.Tx, diagramID, nodeID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, codeLockClass, diagramID+"/"+nodeID)
	return err
}

// The code objects and their rows stay consistent through the node's lock:
// PutFunctionCode calls ensure, which makes sure c's object is in the code
// store, under the lock and before the row commits, and
// DeleteUnreferencedCode deletes an object only under the same lock and only
// when no row names it. So once a row names an object, no delete removes it,
// and a delete that ran between an upload's put and its record is undone by
// ensure. The upload itself is stored before the lock, so a transaction never
// stays open while 50 MB go to the store.

// PutFunctionCode records c as a node's code, replacing any earlier upload,
// and returns the object key it replaced ("" when there was none, or when it
// is c's own key). The insert proves the diagram belongs to the
// organization: it returns ErrNotFound otherwise, without calling ensure.
// An error from ensure is returned as it is, and nothing is recorded.
func (s *Store) PutFunctionCode(ctx context.Context, c FunctionCode, ensure func(context.Context) error) (FunctionCode, string, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return FunctionCode{}, "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit
	if err := lockCodeNode(ctx, tx, c.DiagramID, c.NodeID); err != nil {
		return FunctionCode{}, "", err
	}
	var previous string
	err = tx.QueryRow(ctx, `SELECT fc.object_key FROM function_code fc`+codeScope+`
		AND fc.diagram_id = $2 AND fc.node_id = $3`, c.OrganizationID, c.DiagramID, c.NodeID).Scan(&previous)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return FunctionCode{}, "", err
	}
	var uploadedBy *string
	if c.UploadedBy != "" {
		uploadedBy = &c.UploadedBy
	}
	row := tx.QueryRow(ctx, `
		WITH up AS (
			INSERT INTO function_code (diagram_id, node_id, organization_id, object_key, file_name, size_bytes, sha256, uploaded_by, uploaded_at)
			SELECT g.id, $3, $1, $4, $5, $6, $7, $8, now()
			FROM diagrams g JOIN projects p ON p.id = g.project_id
			WHERE g.id = $2 AND p.organization_id = $1
			ON CONFLICT (diagram_id, node_id) DO UPDATE
				SET object_key = EXCLUDED.object_key, file_name = EXCLUDED.file_name, size_bytes = EXCLUDED.size_bytes,
				    sha256 = EXCLUDED.sha256, uploaded_by = EXCLUDED.uploaded_by, uploaded_at = EXCLUDED.uploaded_at
				WHERE function_code.organization_id = EXCLUDED.organization_id
			RETURNING *
		)
		SELECT `+functionCodeColumns+` FROM up fc`,
		c.OrganizationID, c.DiagramID, c.NodeID, c.ObjectKey, c.FileName, c.Size, c.SHA256, uploadedBy)
	out, err := scanFunctionCode(row)
	if err != nil {
		return FunctionCode{}, "", err
	}
	if err := ensure(ctx); err != nil {
		return FunctionCode{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return FunctionCode{}, "", err
	}
	if previous == out.ObjectKey {
		previous = ""
	}
	return out, previous, nil
}

// DeleteUnreferencedCode calls del to delete the object at key, of the
// diagram's node, under the node's lock, when no function_code row names
// key. It reports whether it called del; an error from del is returned and
// leaves the object to the next try. It reads every organization's rows on
// purpose: an object any row names must stay, and the answer leaves B only
// as that yes or no.
func (s *Store) DeleteUnreferencedCode(ctx context.Context, diagramID, nodeID, key string, del func(context.Context) error) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit
	if err := lockCodeNode(ctx, tx, diagramID, nodeID); err != nil {
		return false, err
	}
	var used bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM function_code WHERE object_key = $1)`, key).Scan(&used); err != nil {
		return false, err
	}
	if used {
		return false, tx.Commit(ctx)
	}
	if err := del(ctx); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// UnreferencedCode returns the keys among keys that no function_code row
// names, in their order. Like DeleteUnreferencedCode it reads every
// organization's rows, and answers only which keys are unused.
func (s *Store) UnreferencedCode(ctx context.Context, keys []string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT k FROM unnest($1::text[]) WITH ORDINALITY AS u(k, i)
		WHERE NOT EXISTS (SELECT 1 FROM function_code fc WHERE fc.object_key = u.k)
		ORDER BY u.i`, keys)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteFunctionCode removes a node's code record from one of an
// organization's diagrams and returns it, so the caller can delete the
// object with DeleteUnreferencedCode. It returns ErrNotFound when there is
// none.
func (s *Store) DeleteFunctionCode(ctx context.Context, orgID, diagramID, nodeID string) (FunctionCode, error) {
	return scanFunctionCode(s.Pool.QueryRow(ctx, `
		WITH gone AS (
			DELETE FROM function_code d
			USING diagrams g, projects p
			WHERE d.diagram_id = $2 AND d.node_id = $3 AND d.organization_id = $1
			  AND g.id = d.diagram_id AND p.id = g.project_id AND p.organization_id = $1
			RETURNING d.*
		)
		SELECT `+functionCodeColumns+` FROM gone fc`, orgID, diagramID, nodeID))
}
