package internal

import (
	"context"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func GetAllOrgsDataForUser(userId string) ([]OrgOverviewData, error) {

	// Connect to PostgreSQL
	// Externally used for iterating through orgs by row
	ctx := context.Background()
	conn, err := generic.PsqlConnect()
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	// Get all organization's data
	// Input:
	// <$1> = member ID
	queryAllOrgDataForUser := `
		SELECT
			o.id,
			o.created_by,
			o.name,
			o.description,
			o.created_at,
			COUNT(DISTINCT p.id) AS total_projects,
			COUNT(DISTINCT d.id) AS total_diagrams,
			COUNT(DISTINCT om.member_id) AS total_members,
			COALESCE(ARRAY_AGG(DISTINCT om.member_id), '{}') AS members
		FROM organizations o
		LEFT JOIN projects p ON p.organization_id = o.id
		LEFT JOIN diagrams d ON d.project_id = p.id
		LEFT JOIN organization_members om ON om.organization_id = o.id
		WHERE o.id IN (
			SELECT organization_id
			FROM organization_members
			WHERE member_id = $1
		)
		GROUP BY o.id, o.created_by, o.name, o.description
		ORDER BY MIN(om.joined_at) ASC
	`

	// Query all org data into rows
	rows, err := conn.Query(ctx, queryAllOrgDataForUser, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Create empty Org OverviewData slice to append to in the iteration process
	orgs := []OrgOverviewData{}

	// Iterate through each org
	for rows.Next() {
		// Initialize variables for org struct and members slice
		var org OrgOverviewData

		// Scan from row to org parameters and members slice
		if err := rows.Scan(
			&org.Id,
			&org.CreatedBy,
			&org.Name,
			&org.Description,
			&org.CreatedAt,
			&org.TotalProjects,
			&org.TotalDiagrams,
			&org.TotalMembers,
			&org.Members,
		); err != nil {
			return nil, err
		}

		// Append full org data to orgs slice
		orgs = append(orgs, org)
	}

	return orgs, nil

}
