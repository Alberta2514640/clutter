package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

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
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to retrieve user data from autorizer context",
			"error":   err.Error(),
		})
	}
	// Get user ID from user authorizer data
	userId := userData.Id

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
	err = generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, orgId)
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

	// Create query to delete organization
	queryDeleteOrg := `
		DELETE FROM organizations
		WHERE id = $1
		RETURNING id, name;
	`

	// Delete organization
	var deletedOrgId string
	var deletedOrgName string
	err = conn.QueryRow(ctx, queryDeleteOrg, orgId).Scan(&deletedOrgId, &deletedOrgName)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": fmt.Sprintf("failed to delete organization with ID %s", deletedOrgId),
			"error":   err.Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message":          fmt.Sprintf("successfully deleted organization with ID %s", deletedOrgId),
		"deleted_org_id":   deletedOrgId,
		"deleted_org_name": deletedOrgName,
	})

}
