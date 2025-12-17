package internal

import (
	"context"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func GetSingleOrgDataWithId(userId string, orgId string) (OrgOverviewData, error) {

	// Connect to PostgreSQL
	// Externally used for iterating through orgs by row
	ctx := context.Background()
	conn, err := generic.PsqlConnect()
	if err != nil {
		return OrgOverviewData{}, err
	}
	defer conn.Close(ctx)

	// Query all data for single org given ID
	querySingleOrg := `
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
		WHERE o.id = $1 AND om.member_id = $2
		GROUP BY o.id, o.created_by, o.name, o.description
	`

	// Initialize empty orgData struct to populate in scan
	var orgData OrgOverviewData

	// Query for single org row
	row := conn.QueryRow(ctx, querySingleOrg, orgId, userId)
	err = row.Scan(
		&orgData.Id,
		&orgData.CreatedBy,
		&orgData.Name,
		&orgData.Description,
		&orgData.CreatedAt,
		&orgData.TotalProjects,
		&orgData.TotalDiagrams,
		&orgData.TotalMembers,
		&orgData.Members,
	)
	if err != nil {
		return OrgOverviewData{}, err
	}

	return orgData, nil

}
