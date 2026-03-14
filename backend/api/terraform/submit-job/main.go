package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
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
	TerraformDirectory   string            `json:"terraform_directory"`
	RoleArn              string            `json:"role_arn"`
	AssumeRoleExternalId string            `json:"assume_role_external_id"`
	ExtraVars            map[string]string `json:"extra_vars,omitempty"`
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
		})
	}
	log.Printf("Authenticated user: %s", userData.Id)

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

	// Validate role_arn format (arn:aws:iam::<12-digit-account>:role/<name>)
	roleArnPattern := regexp.MustCompile(`^arn:aws:iam::\d{12}:role/.+$`)
	if !roleArnPattern.MatchString(body.RoleArn) {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "role_arn must be a valid IAM role ARN",
		})
	}

	// Validate extra_vars size (limit to 64KB)
	if body.ExtraVars != nil {
		extraVarsJSON, err := json.Marshal(body.ExtraVars)
		if err != nil {
			log.Printf("ERROR: failed to marshal extra_vars for size check: %v", err)
			return generic.Response(http.StatusBadRequest, generic.Json{
				"message": "invalid extra_vars",
			})
		}
		if len(extraVarsJSON) > 65536 {
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
		})
	}
	sqsClient := sqs.NewFromConfig(cfg)

	// Connect to PostgreSQL using shared helper
	conn, err := generic.PsqlConnect()
	if err != nil {
		log.Printf("ERROR: failed to connect to database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// Marshal extra vars to JSON
	extraVarsBytes, err := json.Marshal(body.ExtraVars)
	if err != nil {
		log.Printf("ERROR: failed to marshal extra_vars: %v", err)
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid extra_vars format",
		})
	}

	// Start transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Printf("ERROR: failed to begin transaction: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "database error",
		})
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Insert Terraform job into PostgreSQL
	_, err = tx.Exec(ctx, `
		INSERT INTO jobs (id, job_type, status, created_at, updated_at, created_by, terraform_directory, role_arn, assume_role_external_id, extra_vars)
		VALUES ($1, $2, $3, $4, $4, $5, $6, $7, $8, $9)
	`, jobID, "terraform", "QUEUED", now, userID, body.TerraformDirectory, body.RoleArn, body.AssumeRoleExternalId, extraVarsBytes)

	if err != nil {
		log.Printf("ERROR: failed to create job record for job %s: %v", jobID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to create job record",
		})
	}

	// Commit transaction before sending to SQS to avoid orphaned messages
	if err := tx.Commit(ctx); err != nil {
		log.Printf("ERROR: failed to commit transaction: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "database error during commit",
		})
	}

	// Prepare SQS message
	sqsMessage := map[string]string{
		"job_id":                 jobID,
		"job_type":               "terraform",
		"terraform_directory":    body.TerraformDirectory,
		"role_arn":               body.RoleArn,
		"assume_role_external_id": body.AssumeRoleExternalId,
		"extra_vars":             string(extraVarsBytes),
	}
	msgBody, err := json.Marshal(sqsMessage)
	if err != nil {
		log.Printf("ERROR: failed to marshal SQS message: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to prepare job",
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
		// Compensating action: update job status to FAILED since SQS send failed
		compensateConn, compensateErr := generic.PsqlConnect()
		if compensateErr != nil {
			log.Printf("ERROR: failed to open compensating DB connection for job %s: %v", jobID, compensateErr)
		} else {
			defer compensateConn.Close(ctx)
			if _, execErr := compensateConn.Exec(ctx, "UPDATE jobs SET status = 'FAILED', error_message = 'Failed to queue job', updated_at = NOW() WHERE id = $1", jobID); execErr != nil {
				log.Printf("ERROR: failed compensating status update for job %s: %v", jobID, execErr)
			}
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to queue job",
		})
	}

	return generic.Response(http.StatusAccepted, generic.Json{
		"message": "terraform job submitted successfully",
		"data": generic.Json{
			"job_id":   jobID,
			"status":   "QUEUED",
			"job_type": "terraform",
		},
	})
}
