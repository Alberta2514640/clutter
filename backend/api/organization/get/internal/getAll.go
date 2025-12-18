package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func GetAllOrgsDataForUser(userId string) ([]generic.OrgOverviewData, error) {

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
	queryAllOrgDataForUser := fmt.Sprintf(
		generic.QueryOrgData,
		`WHERE o.id IN (
			SELECT organization_id
			FROM organization_members
			WHERE member_id = $1
		)`,
	)

	// Query all org data into rows
	rows, err := conn.Query(ctx, queryAllOrgDataForUser, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Create empty Org OverviewData slice to append to in the iteration process
	orgs := []generic.OrgOverviewData{}

	// Iterate through each org
	for rows.Next() {
		// Initialize variables for org struct and members slice
		var orgData generic.OrgOverviewData
		// Initialize variable to hold projects json as bytes
		var projectsJson []byte

		// Scan from row to org parameters and members slice
		if err := rows.Scan(
			&orgData.Id,
			&orgData.CreatedBy,
			&orgData.Name,
			&orgData.Description,
			&orgData.CreatedAt,
			&orgData.TotalMembers,
			&orgData.Members,
			&orgData.TotalProjects,
			&projectsJson,
			&orgData.TotalDiagrams,
		); err != nil {
			return nil, err
		}

		// unmarshal projects json as bytes into Projects slice
		if err := json.Unmarshal(projectsJson, &orgData.Projects); err != nil {
			return nil, err
		}

		// Append full org data to orgs slice
		orgs = append(orgs, orgData)
	}

	return orgs, nil

}
