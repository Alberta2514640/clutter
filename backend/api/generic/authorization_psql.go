package generic

import (
	"context"
	"errors"

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

// GetProjectOrganizationPSQL fetches a project and extracts its organization ID using PostgreSQL
// Returns orgID and error
func GetProjectOrganizationPSQL(
	ctx context.Context,
	conn *pgx.Conn,
	projectID string,
) (string, error) {
	if projectID == "" {
		return "", NewNotFoundError("Project not found")
	}

	var orgID string
	query := `
		SELECT organization_id 
		FROM projects 
		WHERE id = $1
	`
	err := conn.QueryRow(ctx, query, projectID).Scan(&orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", NewNotFoundError("Project not found")
		}
		return "", err
	}

	return orgID, nil
}

// GetDiagramProjectPSQL fetches a diagram and extracts its project ID using PostgreSQL
// Returns projectID and error
func GetDiagramProjectPSQL(
	ctx context.Context,
	conn *pgx.Conn,
	diagramID string,
) (string, error) {
	if diagramID == "" {
		return "", NewNotFoundError("Diagram not found")
	}

	var projectID string
	query := `
		SELECT project_id 
		FROM diagrams 
		WHERE id = $1
	`
	err := conn.QueryRow(ctx, query, diagramID).Scan(&projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", NewNotFoundError("Diagram not found")
		}
		return "", err
	}

	return projectID, nil
}
