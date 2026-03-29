package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/uploadutils"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jackc/pgx/v5"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		log.Printf("ERROR: unauthorized request: %v", err)
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
		})
	}

	jobID := request.PathParameters["jobId"]
	if jobID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "jobId path parameter is required",
		})
	}

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
		})
	}
	defer conn.Close(ctx)

	var status string
	var logS3Key *string
	var errorMessage *string
	var playbookS3Key *string
	var terraformDirectory *string

	err = conn.QueryRow(ctx, `
		SELECT status, log_s3_key, error_message, playbook_s3_key, terraform_directory
		FROM jobs
		WHERE id = $1
	`, jobID).Scan(&status, &logS3Key, &errorMessage, &playbookS3Key, &terraformDirectory)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generic.Response(http.StatusNotFound, generic.Json{
				"message": "job not found",
			})
		}
		log.Printf("ERROR: failed to retrieve job %s: %v", jobID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to retrieve job",
		})
	}

	// Enforce Organization-Level Access
	orgID, err := uploadutils.ExtractOrgIDFromJobPaths(playbookS3Key, terraformDirectory)
	if err != nil {
		return generic.Response(http.StatusNotFound, generic.Json{"message": "job not found"})
	}
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userData.Id, orgID); err != nil {
		var authErr *generic.AuthorizationError
		if errors.As(err, &authErr) {
			return generic.Response(http.StatusNotFound, generic.Json{"message": "job not found"})
		}
		log.Printf("ERROR: failed to check org membership for job %s: %v", jobID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": "failed to retrieve job"})
	}

	if logS3Key == nil || *logS3Key == "" {
		resp := generic.Json{
			"message": "logs not available yet",
			"status":  status,
		}
		if errorMessage != nil && *errorMessage != "" {
			resp["error_message"] = *errorMessage
		}
		return generic.Response(http.StatusNotFound, resp)
	}

	if _, _, _, err := uploadutils.ExtractPathComponentsFromLogKey(*logS3Key); err != nil {
		log.Printf("ERROR: invalid log_s3_key format for job %s: %s", jobID, *logS3Key)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to retrieve log file",
		})
	}

	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "server configuration error: missing S3_BUCKET_NAME",
		})
	}

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("ERROR: failed to load AWS config: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to load AWS config",
		})
	}

	s3Client := s3.NewFromConfig(awsCfg)
	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(*logS3Key),
	})
	if err != nil {
		var noSuchKey *s3types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return generic.Response(http.StatusNotFound, generic.Json{
				"message": "log file not found in storage",
			})
		}
		log.Printf("ERROR: failed to fetch log from S3 key %s: %v", *logS3Key, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to retrieve log file",
		})
	}
	defer result.Body.Close()

	const maxLogBytes int64 = 10 * 1024 * 1024 // 10 MB
	// Read up to maxLogBytes + 1 to detect truncation
	logBytes, err := io.ReadAll(io.LimitReader(result.Body, maxLogBytes+1))
	if err != nil {
		log.Printf("ERROR: failed to read log body: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to read log content",
		})
	}
	// Check if the log was truncated (more data exists beyond maxLogBytes)
	truncated := int64(len(logBytes)) > maxLogBytes
	if truncated {
		logBytes = logBytes[:maxLogBytes]
	}

	responseData := generic.Json{
		"job_id":      jobID,
		"status":      status,
		"log_s3_key":  *logS3Key,
		"log_content": string(logBytes),
	}
	if truncated {
		responseData["truncated"] = true
	}

	return generic.Response(http.StatusOK, generic.Json{
		"data": responseData,
	})
}

