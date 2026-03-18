package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/organization/accounts/submit-role-arn/internal"
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
			generic.Json{"message": "failed to retrieve user data from authorizer context", "error": err.Error()},
		)
	}
	userId := userData.Id

	// Process body from API Gateway request
	var body internal.RequestBody

	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request", "error": err.Error()})
	}
	roleArn := body.RoleArn

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
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to PostgreSQL database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	// Check membership of user to organization
	// Source: ChatGPT
	err = generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, organizationId)
	if err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			// Handle 404 Not Found
			if authErr.StatusCode == 404 {
				return generic.Response(http.StatusNotFound, generic.Json{
					"error": authErr.Message,
				})
			}

			// Handle 403 Forbidden
			if authErr.StatusCode == 403 {
				return generic.Response(http.StatusForbidden, generic.Json{
					"error": authErr.Message,
				})
			}
		}

		// Fallback for unexpected errors
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
			"error":   err.Error(),
		})
	}

	// Get unique ID from the role ARN the user passed in.
	// This MUST match what is on the database!
	uniqueId := internal.GetUniqueIdFromArn(roleArn)

	// Insertion query
	query := `
		UPDATE aws_account_access_roles
		SET role_arn = $1, status = 'complete'
		WHERE id = $2
		AND organization_id = $3
		AND unique_id = $4
		AND status = 'incomplete';
	`

	// Execute query
	cmdTag, err := conn.Exec(ctx, query,
		roleArn,
		accountId,
		organizationId,
		uniqueId,
	)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to update role arn in database",
			"error":   err.Error(),
		})
	}
	if cmdTag.RowsAffected() == 0 {
		// Check if the row exists but is already complete
		var status string
		err := conn.QueryRow(ctx, `
			SELECT status
			FROM aws_account_access_roles
			WHERE id = $1
			AND organization_id = $2
			AND unique_id = $3;
		`,
			accountId,
			organizationId,
			uniqueId,
		).Scan(&status)

		if err == nil && status == "complete" {
			return generic.Response(http.StatusBadRequest, generic.Json{
				"message": "role ARN already submitted for this account",
			})
		}

		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "account ID, organization ID, or unique ID mismatch",
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "role ARN submitted successfully",
	})
}
