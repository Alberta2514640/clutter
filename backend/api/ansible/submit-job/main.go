package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
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
			"message": "target_instance_ids is required and must not be empty",
		})
	}
	if body.PlaybookS3Key == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "playbook_s3_key is required",
		})
	}

	// Validate playbook_s3_key to prevent path traversal
	if strings.Contains(body.PlaybookS3Key, "..") ||
		strings.Contains(body.PlaybookS3Key, "~") ||
		strings.HasPrefix(body.PlaybookS3Key, "/") ||
		strings.HasPrefix(body.PlaybookS3Key, "\\") {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "playbook_s3_key contains invalid path",
		})
	}

	// Validate target_instance_ids are valid EC2 instance ID format (i-xxxxxxxx)
	for _, id := range body.TargetInstanceIDs {
		if len(id) < 10 || len(id) > 20 {
			return generic.Response(http.StatusBadRequest, generic.Json{
				"message": "invalid instance ID format: " + id,
			})
		}
	}

	// Validate extra_vars size (limit to 64KB)
	if body.ExtraVars != nil {
		extraVarsJSON, err := json.Marshal(body.ExtraVars)
		if err == nil && len(extraVarsJSON) > 65536 {
			return generic.Response(http.StatusBadRequest, generic.Json{
				"message": "extra_vars exceeds maximum size of 64KB",
			})
		}
	}

	// Generate job ID and timestamp
	jobID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)

	// Get user ID from authorizer context
	userID := userData.Id

	// Get and validate environment variables
	queueURL := os.Getenv("JOB_QUEUE_URL")
	if queueURL == "" {
		log.Printf("ERROR: JOB_QUEUE_URL environment variable is not set")
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "server configuration error",
		})
	}

	// Initialize AWS SDK clients
	cfg, cfgErr := config.LoadDefaultConfig(ctx)
	if cfgErr != nil {
		log.Printf("ERROR: failed to load AWS config: %v", cfgErr)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to load AWS config",
			"error":   cfgErr.Error(),
		})
	}
	sqsClient := sqs.NewFromConfig(cfg)

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

	// Marshal instance IDs and extra vars to JSON
	instanceIDsJSON, err := json.Marshal(body.TargetInstanceIDs)
	if err != nil {
		log.Printf("ERROR: failed to marshal target_instance_ids: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to process target instance IDs",
			"error":   err.Error(),
		})
	}
	extraVarsJSON, err := json.Marshal(body.ExtraVars)
	if err != nil {
		log.Printf("ERROR: failed to marshal extra_vars: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to process extra vars",
			"error":   err.Error(),
		})
	}

	// Start transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Printf("ERROR: failed to begin transaction: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "database error",
			"error":   err.Error(),
		})
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Insert job into PostgreSQL with created_by
	var configID *string
	if body.ConfigID != "" {
		configID = &body.ConfigID
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO jobs (id, status, created_at, updated_at, created_by, target_instance_ids, playbook_s3_key, extra_vars, config_id)
		VALUES ($1, $2, $3, $3, $4, $5, $6, $7, $8)
	`, jobID, "QUEUED", now, userID, instanceIDsJSON, body.PlaybookS3Key, extraVarsJSON, configID)

	if err != nil {
		log.Printf("ERROR: failed to create job record for job %s: %v", jobID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to create job record",
			"error":   err.Error(),
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
			"message": "failed to prepare job",
			"error":   err.Error(),
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
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to queue job",
			"error":   err.Error(),
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Printf("ERROR: failed to commit transaction: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "database error during commit",
			"error":   err.Error(),
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
