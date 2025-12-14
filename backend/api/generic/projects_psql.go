package generic

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

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
