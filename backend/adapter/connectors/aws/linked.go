package aws

import (
	"context"
	"errors"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrLinkedRoleNotFound means the organization has no completed linked role
// with that id.
var ErrLinkedRoleNotFound = errors.New("linked AWS role not found")

// Querier is the part of a pgx pool LinkedRoles uses.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// LinkedRoles reads the AWS roles an organization linked through A's
// CloudFormation flow (A's aws_account_access_roles table), so B can offer
// them as AWS connections. It only reads, and every query filters on the
// organization. The composition root hands it to the connector
// (WithLinkedRoles), which offers the roles through connectors.Linker.
type LinkedRoles struct{ DB Querier }

// LinkedRole is one completed role link. It holds no secret: the external id
// is read only by Credentials.
type LinkedRole struct {
	ID          string
	AccountName string
	RoleARN     string
	CreatedAt   time.Time
}

// List returns the organization's completed role links, by account name.
func (l LinkedRoles) List(ctx context.Context, orgID string) ([]LinkedRole, error) {
	if _, err := uuid.Parse(orgID); err != nil {
		return nil, errors.New("organization id is not valid")
	}
	rows, err := l.DB.Query(ctx, `
		SELECT id::text, account_name, role_arn, created_at
		FROM public.aws_account_access_roles
		WHERE organization_id = $1 AND status = 'complete' AND role_arn IS NOT NULL
		ORDER BY account_name, id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LinkedRole{}
	for rows.Next() {
		var r LinkedRole
		if err := rows.Scan(&r.ID, &r.AccountName, &r.RoleARN, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Credentials returns one completed role link's credentials, in the shape
// this connector's CredentialSpec asks for, to create a connection from it.
// It filters on the organization, so another organization's link, even with
// a known external id, is ErrLinkedRoleNotFound.
func (l LinkedRoles) Credentials(ctx context.Context, orgID, roleID string) (connectors.Secrets, error) {
	if _, err := uuid.Parse(orgID); err != nil {
		return nil, ErrLinkedRoleNotFound
	}
	if _, err := uuid.Parse(roleID); err != nil {
		return nil, ErrLinkedRoleNotFound
	}
	var arn, ext string
	err := l.DB.QueryRow(ctx, `
		SELECT role_arn, external_id
		FROM public.aws_account_access_roles
		WHERE id = $1 AND organization_id = $2 AND status = 'complete' AND role_arn IS NOT NULL`,
		roleID, orgID).Scan(&arn, &ext)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLinkedRoleNotFound
	}
	if err != nil {
		return nil, err
	}
	return connectors.Secrets{CredRoleARN: arn, CredExternalID: ext}, nil
}
