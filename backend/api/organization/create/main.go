package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/organization/create/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// dbConn is the part of a Postgres connection this handler uses: the shared
// Conn plus a transaction for the organization and its first member.
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

func handler(request events.APIGatewayProxyRequest) (resp events.APIGatewayProxyResponse, err error) {

	// The authorizer context retains data about the user extracted from the JWT inside the authorizer Lambda function
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from authorizer context"},
		)
	}

	// Process body from API Gateway request
	var body generic.OrgRequestBody

	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request"})
	}
	organizationName := body.OrganizationName
	description := body.Description
	// The website's rule, checked here too, so a bad name is a 400 and not a database error.
	if !generic.OrganizationNameOK(organizationName) {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": generic.OrganizationNameProblem})
	}

	// Generate required fields for organization
	organizationId := uuid.NewString()

	// Build organizationData struct
	organizationData := internal.OrganizationData{
		Id:          organizationId,
		CreatedBy:   userData.Id,
		Name:        organizationName,
		Description: description,
	}

	// SQL queries for insertion
	// Insert new organization
	queryInsertOrg := `
		INSERT INTO organizations (id, created_by, name, description)
		VALUES ($1, $2, $3, $4)
	`
	// Insert member (by default the creator)
	queryInsertMember := `
		INSERT INTO organization_members (organization_id, member_id)
		VALUES ($1, $2)
	`

	// Connect to PostgreSQL
	ctx := context.Background()
	conn, err := connect()
	if err != nil {
		log.Printf("organization create: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": "failed to connect to database"})
	}
	defer conn.Close(ctx)

	// Begin organization creation transaction since we will be making 2 seperate inserts
	transaction, err := conn.Begin(ctx)
	if err != nil {
		log.Printf("organization create: begin: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to start transaction",
		})
	}
	defer transaction.Rollback(ctx) // If any of the queries fail, undo all changes

	// Add organization insertion to transaction
	if _, err := transaction.Exec(
		ctx,
		queryInsertOrg,
		organizationData.Id,
		organizationData.CreatedBy,
		organizationData.Name,
		organizationData.Description,
	); err != nil {
		// If err is of a unique violation it means user attempted to create a duplicate org so respond with conflict error
		if generic.IsUniqueViolation(err) {
			return generic.Response(http.StatusConflict, generic.Json{
				"message": "organization name already exists for this user",
			})
		}
		// Otherwise it is an unexpected error and thus respond with internal server error
		log.Printf("organization create: insert the organization: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to create the organization",
		})
	}

	// Add member -> organization link insertion to transaction
	if _, err := transaction.Exec(ctx, queryInsertMember, organizationData.Id, organizationData.CreatedBy); err != nil {
		log.Printf("organization create: insert the member: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to create the organization",
		})
	}

	// Create query for getting data of newly created org
	queryOrgOverviewData := fmt.Sprintf(
		generic.QueryOrgData,
		"WHERE o.id = $1",
	)

	// Initialize required variables
	var orgData generic.OrgOverviewData
	var projectsJson []byte

	// Query the organization which was just created for overview data to be used by frontend
	row := transaction.QueryRow(ctx, queryOrgOverviewData, organizationData.Id)
	if err := row.Scan(
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
		log.Printf("organization create: read the overview: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch organization overview",
		})
	}

	// unmarshal projects json as bytes into Projects slice
	if err := json.Unmarshal(projectsJson, &orgData.Projects); err != nil {
		log.Printf("organization create: parse projects: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to parse projects data",
		})
	}

	// Commit the transaction
	if err := transaction.Commit(ctx); err != nil {
		log.Printf("organization create: commit: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to create the organization",
		})
	}

	return generic.Response(
		http.StatusOK,
		generic.Json{"message": "new organization created successfully", "data": orgData},
	)

}
