package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	jobID := request.PathParameters["jobId"]
	if jobID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"error": "jobId path parameter is required",
		})
	}

	connString := os.Getenv("PSQL_CONNECTION_STRING")
	if connString == "" {
		log.Printf("ERROR: PSQL_CONNECTION_STRING environment variable is not set")
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "server configuration error",
		})
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		log.Printf("ERROR: failed to connect to database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	var id, status, playbookS3Key string
	var createdAt, updatedAt time.Time
	var targetInstanceIDs, extraVars []byte
	var taskArn, errorMessage *string

	err = conn.QueryRow(ctx, `
		SELECT id, status, created_at, updated_at, target_instance_ids, playbook_s3_key, extra_vars, task_arn, error_message
		FROM jobs
		WHERE id = $1
	`, jobID).Scan(
		&id, &status, &createdAt, &updatedAt,
		&targetInstanceIDs, &playbookS3Key, &extraVars,
		&taskArn, &errorMessage,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return generic.Response(http.StatusNotFound, generic.Json{
				"error": "job not found",
			})
		}
		log.Printf("ERROR: failed to retrieve job %s: %v", jobID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "failed to retrieve job",
		})
	}

	job := map[string]interface{}{
		"id":              id,
		"status":          status,
		"created_at":      createdAt,
		"updated_at":      updatedAt,
		"playbook_s3_key": playbookS3Key,
	}

	// Optional fields
	if taskArn != nil {
		job["task_arn"] = *taskArn
	}
	if errorMessage != nil {
		job["error_message"] = *errorMessage
	}

	// Unmarshal JSON types
	var targets []string
	if err := json.Unmarshal(targetInstanceIDs, &targets); err == nil {
		job["target_instance_ids"] = targets
	} else {
		job["target_instance_ids"] = targetInstanceIDs // Fallback
	}

	var vars map[string]interface{}
	// extraVars might be empty JSON object "{}" or null depending on DB default
	if len(extraVars) > 0 {
		if err := json.Unmarshal(extraVars, &vars); err == nil {
			job["extra_vars"] = vars
		}
	}

	return generic.Response(http.StatusOK, generic.Json{
		"data": job,
	})
}
