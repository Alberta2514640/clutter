package main

import (
	"context"
	"encoding/json"
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

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	statusFilter := request.QueryStringParameters["status"]

	ctx := context.Background()

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

	var rows pgx.Rows
	var queryErr error

	if statusFilter != "" {
		rows, queryErr = conn.Query(ctx, `
			SELECT id, status, created_at, updated_at, target_instance_ids, playbook_s3_key, extra_vars, task_arn, error_message
			FROM jobs
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT 50
		`, statusFilter)
	} else {
		rows, queryErr = conn.Query(ctx, `
			SELECT id, status, created_at, updated_at, target_instance_ids, playbook_s3_key, extra_vars, task_arn, error_message
			FROM jobs
			ORDER BY created_at DESC
			LIMIT 50
		`)
	}

	if queryErr != nil {
		log.Printf("ERROR: failed to query jobs: %v", queryErr)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to query jobs",
			"error":   queryErr.Error(),
		})
	}
	defer rows.Close()

	jobs := []map[string]interface{}{}

	for rows.Next() {
		var id, status, playbookS3Key string
		var createdAt, updatedAt time.Time
		var targetInstanceIDs, extraVars []byte
		var taskArn, errorMessage *string

		err := rows.Scan(
			&id, &status, &createdAt, &updatedAt,
			&targetInstanceIDs, &playbookS3Key, &extraVars,
			&taskArn, &errorMessage,
		)
		if err != nil {
			log.Printf("ERROR: failed to scan row: %v", err)
			continue
		}

		job := map[string]interface{}{
			"id":              id,
			"status":          status,
			"created_at":      createdAt,
			"updated_at":      updatedAt,
			"playbook_s3_key": playbookS3Key,
		}

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
		// extraVars might be empty JSON object "{}" or null depending on DB default, but we set default '{}'
		if len(extraVars) > 0 {
			if err := json.Unmarshal(extraVars, &vars); err == nil {
				job["extra_vars"] = vars
			}
		}

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
