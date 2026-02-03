package generic

import (
	"context"
	"fmt"

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

	// Check if org exists
	queryOrgExists := `
		SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1)
	`
	var orgExists bool
	if err := conn.QueryRow(ctx, queryOrgExists, orgID).Scan(&orgExists); err != nil {
		return err
	}

	if !orgExists {
		return NewNotFoundError(fmt.Sprintf("organization with ID %s not found", orgID))
	}

	var userIsMemberOfOrg bool
	queryUserIsMember := `
		SELECT EXISTS(
			SELECT 1 
			FROM organization_members 
			WHERE organization_id = $1 AND member_id = $2
		)
	`
	err := conn.QueryRow(ctx, queryUserIsMember, orgID, userID).Scan(&userIsMemberOfOrg)
	if err != nil {
		return err
	}

	if !userIsMemberOfOrg {
		return NewForbiddenError(fmt.Sprintf("user with id '%s' does not have access to organization with id '%s'", userID, orgID))
	}

	return nil
}
