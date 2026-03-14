package main

import (
	"context"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/organization/accounts/get/internal"
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

	// Read path parameters
	// Required: organizationId (Path Param)
	organizationId := request.PathParameters["organizationId"]
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

	// Fetch all account access roles for the organization
	rows, err := conn.Query(ctx,
		`SELECT id, account_name, role_arn, status
		FROM aws_account_access_roles
		WHERE organization_id = $1`,
		organizationId,
	)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to query account access roles",
			"error":   err.Error(),
		})
	}
	defer rows.Close()

	var roles []internal.AccountAccessRole
	for rows.Next() {
		var role internal.AccountAccessRole
		err := rows.Scan(&role.Id, &role.AccountName, &role.RoleArn, &role.Status)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to scan account access role row",
				"error":   err.Error(),
			})
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "error iterating account access role rows",
			"error":   err.Error(),
		})
	}

	// Separate complete vs incomplete — incomplete ones may have a nil role_arn so skip that field

	var completeRoles []internal.AccountAccessRole
	var incompleteRoles []internal.AccountAccessRoleIncomplete

	for _, role := range roles {
		if role.Status == "incomplete" {
			incompleteRoles = append(incompleteRoles, internal.AccountAccessRoleIncomplete{
				Id:          role.Id,
				AccountName: role.AccountName,
				Status:      role.Status,
			})
		} else {
			completeRoles = append(completeRoles, role)
		}
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "successfully returned list of account access roles",
		"data": generic.Json{
			"complete":   completeRoles,
			"incomplete": incompleteRoles,
		},
	})
}
