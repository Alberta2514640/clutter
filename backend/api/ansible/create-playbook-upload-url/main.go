package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/uploadutils"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

const defaultUploadURLTTLSeconds = 900

type createPlaybookUploadURLRequest struct {
	FileName  string `json:"file_name"`
	ProjectID string `json:"project_id"`
	DiagramID string `json:"diagram_id"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		log.Printf("ERROR: unauthorized request: %v", err)
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized",
		})
	}

	var body createPlaybookUploadURLRequest
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		log.Printf("ERROR: failed to unmarshal request body: %v", err)
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid request body",
		})
	}

	normalizedFileName, err := uploadutils.ValidatePlaybookFileName(body.FileName)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": err.Error(),
		})
	}

	if body.ProjectID == "" || body.DiagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "project_id and diagram_id are required",
		})
	}

	conn, err := generic.PsqlConnect()
	if err != nil {
		log.Printf("ERROR: failed to connect to database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": "internal server error"})
	}
	defer conn.Close(ctx)

	// 1. Get org from project (404 if project doesn't exist)
	orgID, err := generic.GetProjectOrganizationPSQL(ctx, conn, body.ProjectID)
	if err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{"message": authErr.Message})
		}
		log.Printf("ERROR: failed to fetch project: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": "internal server error"})
	}

	// 2. Check org membership (403 if not a member)
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userData.Id, orgID); err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{"message": authErr.Message})
		}
		log.Printf("ERROR: failed to check membership: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": "internal server error"})
	}

	// 3. Confirm diagram belongs to this project (404 if not)
	var diagramExists bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM diagrams WHERE id = $1 AND project_id = $2)`,
		body.DiagramID, body.ProjectID,
	).Scan(&diagramExists)
	if err != nil {
		log.Printf("ERROR: failed to verify diagram: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": "internal server error"})
	}
	if !diagramExists {
		return generic.Response(http.StatusNotFound, generic.Json{"message": "diagram not found"})
	}

	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "server configuration error: missing S3_BUCKET_NAME",
		})
	}

	uploadID := uuid.NewString()
	objectKey := uploadutils.BuildPlaybookObjectKey(orgID, body.ProjectID, body.DiagramID, normalizedFileName, uploadID)

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("ERROR: failed to load AWS config: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
		})
	}

	presigner := s3.NewPresignClient(s3.NewFromConfig(awsCfg))
	presignedRequest, err := presigner.PresignPutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket:               aws.String(bucketName),
			Key:                  aws.String(objectKey),
			ServerSideEncryption: s3types.ServerSideEncryptionAes256,
		},
		func(opts *s3.PresignOptions) {
			opts.Expires = time.Duration(defaultUploadURLTTLSeconds) * time.Second
		},
	)
	if err != nil {
		log.Printf("ERROR: failed to create upload URL: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
		})
	}

	return generic.Response(http.StatusCreated, generic.Json{
		"message": "playbook upload URL created successfully",
		"data": generic.Json{
			"method":             "PUT",
			"upload_url":         presignedRequest.URL,
			"playbook_s3_key":    objectKey,
			"file_name":          normalizedFileName,
			"expires_in_seconds": defaultUploadURLTTLSeconds,
			"required_headers": generic.Json{
				"x-amz-server-side-encryption": "AES256",
			},
		},
	})
}
