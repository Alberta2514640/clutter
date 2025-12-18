package internal

import (
	"context"
	"encoding/json"
	"fmt"

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
	querySingleOrg := fmt.Sprintf(
		QueryOrgData,
		"WHERE o.id = $1 AND om.member_id = $2",
	)

	// Initialize empty orgData struct to populate in scan
	var orgData OrgOverviewData
	// Initialize variable to hold projects json as bytes
	var projectsJson []byte

	// Query for single org row
	row := conn.QueryRow(ctx, querySingleOrg, orgId, userId)
	err = row.Scan(
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
	)
	if err != nil {
		return OrgOverviewData{}, err
	}

	// unmarshal projects json as bytes into Projects slice
	if err := json.Unmarshal(projectsJson, &orgData.Projects); err != nil {
		return OrgOverviewData{}, err
	}

	return orgData, nil

}
