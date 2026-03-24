package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-command-runner/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context retains data about the user (extracted from the JWT provided in the Authorization header)
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from authorizer context", "error": err.Error()},
		)
	}
	userId := userData.Id

	// Parse request body
	var body internal.RequestBody

	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request", "error": err.Error()})
	}
	organizationId := body.OrganizationId
	projectId := body.ProjectId
	diagramId := body.DiagramId
	accountAccessRoleId := body.AccountAccessRoleId
	region := body.Region
	command := body.Command
	// Set default region if not passed in
	if region == "" {
		region = generic.DefaultRegion
	}

	// Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	// Check user's membership to organization
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, organizationId); err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"message": authErr.Message,
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to check authorization",
			"error":   err.Error(),
		})
	}

	// Get Client Role ARN from DB
	roleArnQuery := `
		SELECT role_arn
		FROM aws_account_access_roles
		WHERE id = $1
			AND organization_id = $2
			AND status = 'complete';
	`
	var roleArn string
	if err := conn.QueryRow(
		ctx,
		roleArnQuery,
		accountAccessRoleId,
		organizationId,
	).Scan(&roleArn); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to retrieve role ARN",
			"error":   err.Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "Terraform command executed successfully",
		"data": generic.Json{
			"organizationId": organizationId,
			"projectId":      projectId,
			"diagramId":      diagramId,
			"roleArn":        roleArn,
			"region":         region,
			"command":        command,
			"userId":         userId,
		},
	})
}
