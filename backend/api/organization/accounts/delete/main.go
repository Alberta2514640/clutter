package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context retains data about the user (extracted from the JWT provided in the Authorization header)
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from authorizer context"},
		)
	}
	userId := userData.Id

	// Read path parameters
	// Required: organizationId (Path Param)
	// Required: accountId      (Path Param)
	organizationId := request.PathParameters["organizationId"]
	accountId := request.PathParameters["accountId"]
	// Check to see if organization ID is not passed in
	if organizationId == "" {
		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": "no 'organizationId' path parameter inputted."},
		)
	}
	// Check to see if organization ID passed in is valid UUID
	isValidUuid := generic.IsValidUuid(organizationId)
	if !isValidUuid {
		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": "inputted 'organizationId' path parameter is not a valid UUID."},
		)
	}
	// Check to see if account ID is not passed in
	if accountId == "" {
		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": "no 'accountId' path parameter inputted."},
		)
	}
	// Check to see if account ID passed in is valid UUID
	isValidUuid = generic.IsValidUuid(accountId)
	if !isValidUuid {
		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": "inputted 'accountId' path parameter is not a valid UUID."},
		)
	}

	// Connect to PostgreSQL
	ctx := context.Background()
	conn, err := generic.PsqlConnect()
	if err != nil {
		log.Printf("accounts delete: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// Check membership of user to organization
	err = generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, organizationId)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	// Verify the account access role entry exists before attempting deletion
	var entryExists bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM aws_account_access_roles
			WHERE id = $1 AND organization_id = $2
		)`,
		accountId, organizationId,
	).Scan(&entryExists)
	if err != nil {
		log.Printf("accounts delete: look up the account: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to verify account access role existence",
		})
	}
	if !entryExists {
		return generic.Response(http.StatusNotFound, generic.Json{
			"message": "account not found",
		})
	}

	// Delete the account role entry
	_, err = conn.Exec(ctx,
		`DELETE FROM aws_account_access_roles
		 WHERE id = $1 AND organization_id = $2`,
		accountId, organizationId,
	)
	if err != nil {
		log.Printf("accounts delete: delete: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to delete account access role",
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "successfully deleted account access role",
	})
}
