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
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SubmitJobRequest struct {
	ConfigID          string            `json:"config_id"`
	TargetInstanceIDs []string          `json:"target_instance_ids"`
	PlaybookS3Key     string            `json:"playbook_s3_key"`
	ExtraVars         map[string]string `json:"extra_vars,omitempty"`
}

func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Parse request body
	var body SubmitJobRequest
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		log.Printf("ERROR: failed to unmarshal request body: %v", err)
		return generic.Response(http.StatusBadRequest, generic.Json{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if len(body.TargetInstanceIDs) == 0 {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"error": "target_instance_ids is required and must not be empty",
		})
	}
	if body.PlaybookS3Key == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"error": "playbook_s3_key is required",
		})
	}

	// Generate job ID and timestamp
	jobID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)

	// Get and validate environment variables
	connString := os.Getenv("PSQL_CONNECTION_STRING")
	queueURL := os.Getenv("JOB_QUEUE_URL")
	if connString == "" {
		log.Printf("ERROR: PSQL_CONNECTION_STRING environment variable is not set")
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "server configuration error",
		})
	}
	if queueURL == "" {
		log.Printf("ERROR: JOB_QUEUE_URL environment variable is not set")
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "server configuration error",
		})
	}

	// Initialize AWS SDK clients
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("ERROR: failed to load AWS config: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "failed to load AWS config",
		})
	}
	sqsClient := sqs.NewFromConfig(cfg)

	// Initialize PostgreSQL connection
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		log.Printf("ERROR: failed to connect to database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// Marshal instance IDs and extra vars to JSON
	instanceIDsJSON, err := json.Marshal(body.TargetInstanceIDs)
	if err != nil {
		log.Printf("ERROR: failed to marshal target_instance_ids: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "failed to process target instance IDs",
		})
	}
	extraVarsJSON, err := json.Marshal(body.ExtraVars)
	if err != nil {
		log.Printf("ERROR: failed to marshal extra_vars: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "failed to process extra vars",
		})
	}

	// Start transaction to ensure atomicity if possible (SQS is external, but we can structure logic)
	// For simplicity in Lambda, we'll insert into DB first, then SQS. If SQS fails, we mark job as FAILED or delete it.

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Printf("ERROR: failed to begin transaction: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "database error",
		})
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Insert job into PostgreSQL
	// config_id can be UUID or NULL. If string is empty, we pass nil.
	var configID *string
	if body.ConfigID != "" {
		configID = &body.ConfigID
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO jobs (id, status, created_at, updated_at, target_instance_ids, playbook_s3_key, extra_vars, config_id)
		VALUES ($1, $2, $3, $3, $4, $5, $6, $7)
	`, jobID, "QUEUED", now, instanceIDsJSON, body.PlaybookS3Key, extraVarsJSON, configID)

	if err != nil {
		log.Printf("ERROR: failed to create job record for job %s: %v", jobID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "failed to create job record",
		})
	}

	// Prepare SQS message
	sqsMessage := map[string]string{
		"job_id":              jobID,
		"playbook_s3_key":     body.PlaybookS3Key,
		"target_instance_ids": string(instanceIDsJSON),
		"extra_vars":          string(extraVarsJSON),
	}
	if body.ConfigID != "" {
		sqsMessage["config_id"] = body.ConfigID
	}
	msgBody, err := json.Marshal(sqsMessage)
	if err != nil {
		log.Printf("ERROR: failed to marshal SQS message: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "failed to prepare job",
		})
	}
	msgStr := string(msgBody)

	// Send to SQS
	_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &queueURL,
		MessageBody: &msgStr,
	})
	if err != nil {
		log.Printf("ERROR: failed to send SQS message for job %s: %v", jobID, err)
		// We can return error here; transaction rollbacks on defer
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "failed to queue job",
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Printf("ERROR: failed to commit transaction: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error": "database error during commit",
		})
	}

	return generic.Response(http.StatusAccepted, generic.Json{
		"message": "job submitted successfully",
		"data": generic.Json{
			"job_id": jobID,
			"status": "QUEUED",
		},
	})
}
