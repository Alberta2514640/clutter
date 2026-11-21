package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/organization/update/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

// dbConn is the part of a Postgres connection this handler uses: the shared
// Conn plus a transaction for the name and description updates.
type dbConn interface {
	generic.Conn
	Begin(ctx context.Context) (pgx.Tx, error)
}

// connect opens the database. Tests replace it with a mock.
var connect = func() (dbConn, error) {
	conn, err := generic.PsqlConnect()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func main() {

	lambda.Start(handler)

}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Get org ID from path paremeter
	orgId := request.PathParameters["organizationId"]
	// Check if org ID is a valid UUID
	isValidUuid := generic.IsValidUuid(orgId)
	if !isValidUuid {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "inputted 'organizationId' path parameter is not a valid UUID.",
		})
	}

	// The authorizer context retains data about the user extracted from the JWT inside the authorizer Lambda function
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from authorizer context"},
		)
	}
	// Get user ID from user authorizer data
	userId := userData.Id

	// Process body from API Gateway request
	var body generic.OrgRequestBody
	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request"})
	}
	// Get org name and/or description from request body
	newOrgName := body.OrganizationName
	newDescription := body.Description
	// The website's rule, checked here too, so a bad name is a 400 and not a database error.
	// An empty name means the name does not change.
	if newOrgName != "" && !generic.OrganizationNameOK(newOrgName) {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": generic.OrganizationNameProblem})
	}

	// Connect to PostgreSQL
	ctx := context.Background()
	conn, err := connect()
	if err != nil {
		log.Printf("organization update: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// Check membership of user to organization
	err = generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, orgId)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	// Fetch current org data
	var currentOrgName, currentDescription string

	queryCurrentData := `
		SELECT name, description
		FROM organizations
		WHERE id = $1
	`

	row := conn.QueryRow(ctx, queryCurrentData, orgId)
	if err := row.Scan(&currentOrgName, &currentDescription); err != nil {
		log.Printf("organization update: read the organization: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch current organization data",
		})
	}

	// Check if anything changed
	nameUnchanged := newOrgName == "" || newOrgName == currentOrgName
	descriptionUnchanged := newDescription == "" || newDescription == currentDescription

	if nameUnchanged && descriptionUnchanged {
		return generic.Response(http.StatusOK, generic.Json{
			"message": "no changes detected, organization not updated",
		})
	}

	// Begin transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Printf("organization update: begin: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to start transaction",
		})
	}
	defer tx.Rollback(ctx)

	// Update organization name only if new and current name don't match
	if newOrgName != "" && newOrgName != currentOrgName {

		err = internal.UpdateOrgName(tx, ctx, newOrgName, orgId)

		if generic.IsUniqueViolation(err) {
			return generic.Response(http.StatusConflict, generic.Json{
				"message": "organization name already exists for this user",
			})
		} else if err != nil {
			log.Printf("organization update: name: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to update organization name",
			})
		}

	}

	// Update organization description only if new and current description don't match
	if newDescription != "" && newDescription != currentDescription {

		err = internal.UpdateOrgDescription(tx, ctx, newDescription, orgId)

		if err != nil {
			log.Printf("organization update: description: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to update organization description",
			})
		}

	}

	// Query all data for updated org
	querySingleOrg := fmt.Sprintf(
		generic.QueryOrgData,
		"WHERE o.id = $1 AND om.member_id = $2",
	)

	// Initialize empty orgData struct to populate in scan
	var orgData generic.OrgOverviewData
	// Initialize variable to hold projects json as bytes
	var projectsJson []byte

	// Get data of new updated organization for use by the frontend
	row = tx.QueryRow(ctx, querySingleOrg, orgId, userId)
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
		log.Printf("organization update: read the overview: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch organization overview",
		})
	}

	// Unmarshal projects json as bytes into Projects slice
	if err := json.Unmarshal(projectsJson, &orgData.Projects); err != nil {
		log.Printf("organization update: parse projects: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to parse projects data",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Printf("organization update: commit: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to commit organization update transaction",
		})
	}

	// Organization update process completed successfully and new data can be returned
	return generic.Response(http.StatusOK, generic.Json{
		"message": "organization updated successfully",
		"data":    orgData,
	})
}
