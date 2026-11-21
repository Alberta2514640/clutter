package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/logs/recent-activity/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context retains data about the user (extracted from the JWT provided in the Authorization header)
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil || userData.Id == "" {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
		})
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
	conn, err := connect()
	if err != nil {
		log.Printf("recent activity: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// Check user's membership to organization, and that the diagram belongs to it
	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userId, orgId, "", diagramId)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	// The deployer has no database access, so a finished run's status reaches
	// its row here, from the status file it wrote beside its logs.
	generic.ReconcileDeployRunsFromEnv(ctx, conn, ids)

	// Get recent activity
	query := `
		SELECT ddl.command_id, ddl.command, ddl.status, ddl.created_at, ddl.duration_seconds, d.name
		FROM public.diagram_deployment_logs ddl
		JOIN public.diagrams d ON d.id = ddl.diagram_id
		WHERE ddl.diagram_id = $1
		ORDER BY ddl.created_at DESC
	`
	recentActivity := []internal.DeploymentLog{}
	rows, err := conn.Query(ctx, query, ids.DiagramID)
	if err != nil {
		log.Printf("recent activity: query: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to get recent activity",
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
			log.Printf("recent activity: scan: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to get recent activity",
			})
		}
		recentActivity = append(recentActivity, deploymentLog)
	}

	if err := rows.Err(); err != nil {
		log.Printf("recent activity: rows: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to get recent activity",
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "successfully retrieved recent activity",
		"data":    recentActivity,
	})
}
