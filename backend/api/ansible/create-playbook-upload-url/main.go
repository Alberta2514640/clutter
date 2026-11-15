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

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil || userData.Id == "" {
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

	conn, err := connect()
	if err != nil {
		log.Printf("ERROR: failed to connect to database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": "internal server error"})
	}
	defer conn.Close(ctx)

	// The diagram must belong to the project and the project to an organization
	// the caller is in. The organization comes from the project row, and the
	// key and the playbook row use the checked, canonical ids.
	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userData.Id, "", body.ProjectID, body.DiagramID)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}
	orgID := ids.OrgID

	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
		})
	}

	uploadID := uuid.NewString()
	objectKey := uploadutils.BuildPlaybookObjectKey(orgID, ids.ProjectID, ids.DiagramID, normalizedFileName, uploadID)

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

	// Insert playbook record into DB and get the generated ID
	var playbookID string
	err = conn.QueryRow(ctx, `
		INSERT INTO playbooks (diagram_id, org_id, project_id, file_name, s3_key, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, ids.DiagramID, orgID, ids.ProjectID, normalizedFileName, objectKey, userData.Id).Scan(&playbookID)
	if err != nil {
		log.Printf("ERROR: failed to insert playbook record: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
		})
	}

	return generic.Response(http.StatusCreated, generic.Json{
		"message": "playbook upload URL created successfully",
		"data": generic.Json{
			"playbook_id":        playbookID,
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
