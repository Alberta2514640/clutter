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

type SubmitTerraformJobRequest struct {
	TerraformDirectory      string            `json:"terraform_directory"`
	RoleArn                 string            `json:"role_arn"`
	AssumeRoleExternalId    string            `json:"assume_role_external_id"`
	ExtraVars               map[string]string `json:"extra_vars,omitempty"`
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
	var body SubmitTerraformJobRequest
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		log.Printf("ERROR: failed to unmarshal request body: %v", err)
		return generic.Response(http.StatusBadRequest, generic.Json{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if body.TerraformDirectory == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "terraform_directory is required",
		})
	}
	if body.RoleArn == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "role_arn is required",
		})
	}
	if body.AssumeRoleExternalId == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "assume_role_external_id is required",
		})
	}

	// Validate terraform_directory to prevent path traversal
	if strings.Contains(body.TerraformDirectory, "..") ||
		strings.Contains(body.TerraformDirectory, "~") ||
		strings.HasPrefix(body.TerraformDirectory, "/") ||
		strings.HasPrefix(body.TerraformDirectory, "\\") {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "terraform_directory contains invalid path",
		})
	}

	// Validate role_arn format
	if !strings.HasPrefix(body.RoleArn, "arn:aws:iam::") {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "role_arn must be a valid AWS IAM role ARN",
		})
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

	// Marshal extra vars to JSON
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

	// Insert Terraform job into PostgreSQL
	_, err = tx.Exec(ctx, `
		INSERT INTO jobs (id, job_type, status, created_at, updated_at, created_by, terraform_directory, role_arn, assume_role_external_id, extra_vars)
		VALUES ($1, $2, $3, $4, $4, $5, $6, $7, $8, $9)
	`, jobID, "terraform", "QUEUED", now, userID, body.TerraformDirectory, body.RoleArn, body.AssumeRoleExternalId, extraVarsJSON)

	if err != nil {
		log.Printf("ERROR: failed to create job record for job %s: %v", jobID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to create job record",
			"error":   err.Error(),
		})
	}

	// Prepare SQS message
	sqsMessage := map[string]string{
		"job_id":                      jobID,
		"job_type":                    "terraform",
		"terraform_directory":          body.TerraformDirectory,
		"role_arn":                    body.RoleArn,
		"assume_role_external_id":      body.AssumeRoleExternalId,
		"extra_vars":                  string(extraVarsJSON),
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
		"message": "terraform job submitted successfully",
		"data": generic.Json{
			"job_id":     jobID,
			"status":     "QUEUED",
			"job_type":   "terraform",
		},
	})
}
