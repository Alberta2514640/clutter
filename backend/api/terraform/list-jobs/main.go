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

// Allowed job_type filter values
var allowedJobTypes = map[string]bool{
	"ansible":   true,
	"terraform": true,
}

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
	log.Printf("Authenticated user: %s (%s)", userData.Id, userData.Email)

	// Get user ID for filtering
	userID := userData.Id

	// Validate optional status filter
	statusFilter := request.QueryStringParameters["status"]
	if statusFilter != "" && !generic.AllowedJobStatuses[statusFilter] {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid status filter. Allowed: QUEUED, STARTING, RUNNING, COMPLETED, FAILED",
		})
	}

	// Validate optional job_type filter
	jobTypeFilter := request.QueryStringParameters["job_type"]
	if jobTypeFilter != "" && !allowedJobTypes[jobTypeFilter] {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid job_type filter. Allowed: ansible, terraform",
		})
	}

	// Connect to PostgreSQL using shared helper
	conn, err := generic.PsqlConnect()
	if err != nil {
		log.Printf("ERROR: failed to connect to database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	// Build query dynamically based on filters
	query := `
		SELECT id, COALESCE(job_type, 'ansible') as job_type, status, created_at, updated_at,
		       extra_vars, task_arn, error_message,
		       target_instance_ids, playbook_s3_key,
		       terraform_directory, role_arn, assume_role_external_id
		FROM jobs
		WHERE created_by = $1`
	args := []interface{}{userID}
	argNum := 2

	if statusFilter != "" {
		query += ` AND status = $` + itoa(argNum)
		args = append(args, statusFilter)
		argNum++
	}
	if jobTypeFilter != "" {
		query += ` AND COALESCE(job_type, 'ansible') = $` + itoa(argNum)
		args = append(args, jobTypeFilter)
		argNum++
	}

	query += ` ORDER BY created_at DESC LIMIT 50`

	var rows pgx.Rows
	rows, err = conn.Query(ctx, query, args...)

	if err != nil {
		log.Printf("ERROR: failed to query jobs: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to query jobs",
			"error":   err.Error(),
		})
	}
	defer rows.Close()

	jobs := []map[string]interface{}{}

	for rows.Next() {
		var id, jobType, status string
		var createdAt, updatedAt time.Time
		var extraVars, targetInstanceIDs []byte
		var taskArn, errorMessage *string
		var playbookS3Key, terraformDirectory, roleArn, assumeRoleExternalId *string

		err := rows.Scan(
			&id, &jobType, &status, &createdAt, &updatedAt,
			&extraVars, &taskArn, &errorMessage,
			&targetInstanceIDs, &playbookS3Key,
			&terraformDirectory, &roleArn, &assumeRoleExternalId,
		)
		if err != nil {
			log.Printf("ERROR: failed to scan row: %v", err)
			continue
		}

		job := generic.BuildJobResponse(
			id, jobType, status,
			createdAt, updatedAt,
			extraVars, taskArn, errorMessage,
			targetInstanceIDs, playbookS3Key,
			terraformDirectory, roleArn, assumeRoleExternalId,
		)

		jobs = append(jobs, job)
	}

	if rows.Err() != nil {
		log.Printf("ERROR: rows iteration error: %v", rows.Err())
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to process job list",
			"error":   rows.Err().Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"data":  jobs,
		"count": len(jobs),
	})
}

// itoa converts a small int to string (avoids importing strconv for 1-digit numbers)
func itoa(n int) string {
	return string(rune('0' + n))
}
