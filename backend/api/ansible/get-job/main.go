package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1. Extract authenticated user from authorizer context
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		log.Printf("ERROR: unauthorized request: %v", err)
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
			"error":   err.Error(),
		})
	}

	// Get user ID for filtering
	userID := userData.Id

	jobID := request.PathParameters["jobId"]
	if jobID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "jobId path parameter is required",
		})
	}

	// Validate jobId is a valid UUID
	if !generic.IsValidUuid(jobID) {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "jobId must be a valid UUID",
		})
	}

	var conn *pgx.Conn
	conn, err = generic.PsqlConnect()
	if err != nil {
		log.Printf("ERROR: failed to connect to database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	var id, jobType, status string
	var createdAt, updatedAt time.Time
	var extraVars, targetInstanceIDs []byte
	var taskArn, errorMessage *string
	var playbookS3Key, terraformDirectory, roleArn, assumeRoleExternalId *string

	// Query job filtered by user_id (ensures user can only access their own jobs)
	err = conn.QueryRow(ctx, `
		SELECT id, COALESCE(job_type, 'ansible') as job_type, status, created_at, updated_at,
		       extra_vars, task_arn, error_message,
		       target_instance_ids, playbook_s3_key,
		       terraform_directory, role_arn, assume_role_external_id
		FROM jobs
		WHERE id = $1 AND created_by = $2
	`, jobID, userID).Scan(
		&id, &jobType, &status, &createdAt, &updatedAt,
		&extraVars, &taskArn, &errorMessage,
		&targetInstanceIDs, &playbookS3Key,
		&terraformDirectory, &roleArn, &assumeRoleExternalId,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Return 404 whether job doesn't exist or belongs to another user
			// This prevents user enumeration
			return generic.Response(http.StatusNotFound, generic.Json{
				"message": "job not found",
			})
		}
		log.Printf("ERROR: failed to retrieve job %s: %v", jobID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to retrieve job",
			"error":   err.Error(),
		})
	}

	job := generic.BuildJobResponse(
		id, jobType, status,
		createdAt, updatedAt,
		extraVars, taskArn, errorMessage,
		targetInstanceIDs, playbookS3Key,
		terraformDirectory, roleArn, assumeRoleExternalId,
	)

	return generic.Response(http.StatusOK, generic.Json{
		"data": job,
	})
}
