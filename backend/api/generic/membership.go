package generic

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// CheckOrganizationMembershipPSQL verifies if a user is a member of an organization using PostgreSQL
// Returns nil if authorized, error otherwise
func CheckOrganizationMembershipPSQL(
	ctx context.Context,
	conn *pgx.Conn,
	userID string,
	orgID string,
) error {
	if userID == "" {
		return NewForbiddenError("User does not have access to this organization")
	}
	if orgID == "" {
		return NewForbiddenError("User does not have access to this organization")
	}

	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM organization_members 
			WHERE organization_id = $1 AND member_id = $2
		)
	`
	err := conn.QueryRow(ctx, query, orgID, userID).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return NewForbiddenError("User does not have access to this organization")
	}

	return nil
}
