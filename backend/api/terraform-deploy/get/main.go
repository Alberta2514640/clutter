// Package main provides the Lambda handler for retrieving deployment logs.
//
// This Lambda is invoked via API Gateway GET /deployment-log/{runId} and performs:
//  1. Validates the JWT and extracts user identity
//  2. Extracts runId from path parameters
//  3. Queries the deployment_runs table for the run details
//  4. Validates user has access to the deployment's project
//  5. Returns the deployment metadata and log content
package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-deploy/get/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1) Extract user data from authorizer context
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		log.Printf("Failed to extract user data: %v", err)
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized",
		})
	}
	userID := userData.Id

	// 2) Extract runId from path parameters
	runID := request.PathParameters["runId"]
	if runID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "runId path parameter is required",
		})
	}

	if !generic.IsValidUuid(runID) {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid runId format",
		})
	}

	// 3) Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
		})
	}
	defer conn.Close(ctx)

	// 4) Query deployment run
	query := `
		SELECT id, project_id, user_id, status, log, created_at, completed_at
		FROM deployment_runs
		WHERE id = $1
	`
	var deploymentLog internal.DeploymentLogResponse
	err = conn.QueryRow(ctx, query, runID).Scan(
		&deploymentLog.ID,
		&deploymentLog.ProjectID,
		&deploymentLog.UserID,
		&deploymentLog.Status,
		&deploymentLog.Log,
		&deploymentLog.CreatedAt,
		&deploymentLog.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generic.Response(http.StatusNotFound, generic.Json{
				"message": "deployment run not found",
			})
		}
		log.Printf("Failed to query deployment run: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
		})
	}

	// 5) Verify user has access to the project
	if err := generic.CheckProjectMembershipPSQL(ctx, conn, userID, deploymentLog.ProjectID); err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"message": authErr.Message,
			})
		}
		log.Printf("Failed to check authorization: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
		})
	}

	// 6) Return deployment log response
	return generic.Response(http.StatusOK, generic.Json{
		"id":          deploymentLog.ID,
		"projectId":   deploymentLog.ProjectID,
		"userId":      deploymentLog.UserID,
		"status":      deploymentLog.Status,
		"log":         deploymentLog.Log,
		"createdAt":   deploymentLog.CreatedAt,
		"completedAt": deploymentLog.CompletedAt,
	})
}
