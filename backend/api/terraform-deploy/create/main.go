// Package main provides the Lambda handler for triggering Terraform deployments.
//
// This Lambda is invoked via API Gateway POST /deploy and performs the following:
//  1. Validates the JWT and extracts user identity
//  2. Validates project authorization (user must be a member of the project's organization)
//  3. Generates a unique run ID
//  4. Inserts a deployment_runs record with status RUNNING
//  5. Starts an ECS Fargate task to execute Terraform
//  6. Returns the run ID to the client
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-deploy/create/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
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
			"message": "unauthorized: missing user identity",
		})
	}
	userID := userData.Id

	// 2) Parse request body
	var req internal.DeployRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid request body",
		})
	}

	// Validate project ID
	if req.ProjectID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "projectId is required",
		})
	}
	if !generic.IsValidUuid(req.ProjectID) {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid projectId format",
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

	// 4) Verify user has access to this project (via organization membership)
	if err := generic.CheckProjectMembershipPSQL(ctx, conn, userID, req.ProjectID); err != nil {
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

	// 5) Load ECS configuration
	ecsConfig, err := internal.LoadECSConfig()
	if err != nil {
		log.Printf("Failed to load ECS configuration: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
		})
	}

	// 6) Generate run ID and insert deployment record
	runID := uuid.NewString()
	insertQuery := `
		INSERT INTO deployment_runs (id, project_id, user_id, status)
		VALUES ($1, $2, $3, 'RUNNING')
	`
	_, err = conn.Exec(ctx, insertQuery, runID, req.ProjectID, userID)
	if err != nil {
		log.Printf("Failed to create deployment record: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
		})
	}

	// 7) Start ECS Fargate task
	if err := internal.RunECSTask(ctx, ecsConfig, req.ProjectID, runID, userID); err != nil {
		// Update deployment record to FAILED
		updateQuery := `
			UPDATE deployment_runs 
			SET status = 'FAILED', completed_at = NOW() 
			WHERE id = $1
		`
		if _, updErr := conn.Exec(ctx, updateQuery, runID); updErr != nil {
			log.Printf("Failed to update deployment record %s to FAILED: %v", runID, updErr)
		}
		log.Printf("Failed to start ECS task for run %s: %v", runID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to start deployment task",
		})
	}

	// 8) Return success response
	return generic.Response(http.StatusAccepted, generic.Json{
		"runId":     runID,
		"projectId": req.ProjectID,
		"status":    "RUNNING",
		"message":   "Deployment started successfully",
	})
}
