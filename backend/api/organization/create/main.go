package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/organization/create/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
)

func main() {

	lambda.Start(handler)

}

func handler(request events.APIGatewayProxyRequest) (resp events.APIGatewayProxyResponse, err error) {

	// The authorizer context retains data about the user extracted from the JWT inside the authorizer Lambda function
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from autorizer context", "error": err.Error()},
		)
	}

	// Process body from API Gateway request
	var body internal.RequestBody

	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request", "error": err.Error()})
	}
	organizationName := body.OrganizationName
	description := body.Description

	// Generate required fields for organization
	organizationId := uuid.NewString()
	// Initialize createdAt as a time.Time to be returned by table upon creation of organization
	var createdAt time.Time

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
		RETURNING created_at
	`
	// Insert member (by default the creator)
	queryInsertMember := `
		INSERT INTO organization_members (organization_id, member_id)
		VALUES ($1, $2)
	`

	// Connect to PostgreSQL
	ctx := context.Background()
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to connect to PostgreSQL database", "message": err.Error()})
	}
	defer conn.Close(ctx)

	// Begin organization creation transaction since we will be making 2 seperate inserts
	transaction, err := conn.Begin(ctx)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to start transaction",
			"message": err.Error(),
		})
	}
	defer transaction.Rollback(ctx) // If any of the queries fail, undo all changes

	// Add organization insertion to transaction
	insertedOrgRow := transaction.QueryRow(
		ctx,
		queryInsertOrg,
		organizationData.Id,
		organizationData.CreatedBy,
		organizationData.Name,
		organizationData.Description,
	)

	if err := insertedOrgRow.Scan(&createdAt); err != nil {

		if generic.IsUniqueViolation(err) {
			return generic.Response(http.StatusConflict, generic.Json{
				"error":   "organization name already exists for this user",
				"message": err.Error(),
			})
		}

		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to add organization insertion to transaction", "message": err.Error()})
	}

	// Add member -> organization link insertion to transaction
	if _, err := transaction.Exec(ctx, queryInsertMember, organizationData.Id, organizationData.CreatedBy); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to add member and organizatio link to transaction",
			"message": err.Error(),
		})
	}

	// Commit the transaction
	if err := transaction.Commit(ctx); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to commit the organization creation transaction",
			"message": err.Error(),
		})
	}

	return generic.Response(
		http.StatusOK,
		generic.Json{"message": "new organization created successfully", "organizationData": organizationData},
	)

}
