package main

import (
	"context"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/logs/recent-activity/internal"
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
	// Get query parmeters
	orgId := request.QueryStringParameters["orgId"]
	diagramId := request.QueryStringParameters["diagramId"]
	// Verify orgId and diagramId are passed in
	if orgId == "" || diagramId == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "orgId and diagramId are required",
		})
	}
	// Ensure orgId and diagramId are valid UUIDs
	if !generic.IsValidUuid(orgId) || !generic.IsValidUuid(diagramId) {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "orgId and diagramId must be valid UUIDs",
		})
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
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, orgId); err != nil {
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

	// Get recent activity
	query := `
		SELECT ddl.command_id, ddl.command, ddl.status, ddl.created_at, ddl.duration_seconds, d.name
		FROM public.diagram_deployment_logs ddl
		JOIN public.diagrams d ON d.id = ddl.diagram_id
		WHERE ddl.diagram_id = $1
		ORDER BY ddl.created_at DESC
	`
	recentActivity := []internal.DeploymentLog{}
	rows, err := conn.Query(ctx, query, diagramId)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to get recent activity",
			"error":   err.Error(),
		})
	}
	defer rows.Close()
	for rows.Next() {
		var deploymentLog internal.DeploymentLog
		if err := rows.Scan(
			&deploymentLog.CommandId,
			&deploymentLog.Command,
			&deploymentLog.Status,
			&deploymentLog.CreatedAt,
			&deploymentLog.Duration,
			&deploymentLog.DiagramName,
		); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to scan row",
				"error":   err.Error(),
			})
		}
		recentActivity = append(recentActivity, deploymentLog)
	}

	if err := rows.Err(); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "error iterating deployment log rows",
			"error":   err.Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "successfully retrieved recent activity",
		"data":    recentActivity,
	})
}
