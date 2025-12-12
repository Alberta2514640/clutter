package internal

import (
	"context"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func GetAllOrgsDataForUser(userId string) ([]OrgOverviewData, error) {

	// Connect to PostgreSQL
	// Externally used for iterating through orgs by row
	ctx := context.Background()
	externalConn, err := generic.PsqlConnect()
	if err != nil {
		return nil, err
	}
	defer externalConn.Close(ctx)

	// SQL Queries

	// Get all organization's data
	// Input:
	// <$1> = member ID
	queryAllOrgs := `
		SELECT orgs.id, orgs.created_by, orgs.name, orgs.description
		FROM organizations orgs
		WHERE orgs.id IN (
			SELECT organization_id
			FROM organization_members
			WHERE member_id = $1
		)
	`
	// Get number of total projects in org
	// Input:
	// <$1> = organization ID
	queryProjectCount := `
		SELECT COUNT(*) FROM projects WHERE organization_id = $1
	`
	// Get number of total diagrams in org
	// Input:
	// <$1> = organization ID
	queryDiagramCount := `
		SELECT COUNT(*)
		FROM diagrams d
		JOIN projects p ON p.id = d.project_id
		WHERE p.organization_id = $1
	`
	// Get members part of the organization by their ID
	// Input:
	// <$1> = organization ID
	queryMemberIds := `
		SELECT member_id FROM organization_members WHERE organization_id = $1
	`

	// Create an empty list of organization overview data
	orgs := []OrgOverviewData{}

	// Query all orgs a user is part of as rows
	orgRows, err := externalConn.Query(ctx, queryAllOrgs, userId)
	if err != nil {
		return nil, err
	}
	defer orgRows.Close()

	// Iterate through each org
	for orgRows.Next() {

		// Create an empty OrgOverviewData struct
		var org OrgOverviewData

		// First scan the org data from organizations table
		if err = orgRows.Scan(&org.Id, &org.CreatedBy, &org.Name, &org.Description); err != nil {
			return nil, err
		}

		// Now that you have org.ID, start querying for rest of overview data

		// Connect to PostgreSQL again
		// Internally used for querying individual org data
		ctx := context.Background()
		internalConn, err := generic.PsqlConnect()
		if err != nil {
			return nil, err
		}

		// Query total projects
		if err = internalConn.QueryRow(ctx, queryProjectCount, org.Id).Scan(&org.TotalProjects); err != nil {
			return nil, err
		}

		// Query total diagrams
		if err = internalConn.QueryRow(ctx, queryDiagramCount, org.Id).Scan(&org.TotalDiagrams); err != nil {
			return nil, err
		}

		// Query member IDs
		memberRows, err := internalConn.Query(ctx, queryMemberIds, org.Id)
		if err != nil {
			return nil, err
		}
		// Created empty list for members
		members := []string{}
		// Iterate through each member in org
		for memberRows.Next() {
			var memberId string
			if err := memberRows.Scan(&memberId); err != nil {
				return nil, err
			}
			members = append(members, memberId)
		}
		// Close rows opened for member query
		memberRows.Close()
		// Close internalConn since it is no longer needed
		internalConn.Close(ctx)
		// Assign member values acquired to orgOverviewData struct parameters
		org.Members = members
		org.TotalMembers = len(members)

		// append this completed org data to orgs
		orgs = append(orgs, org)

	}

	return orgs, nil

}
