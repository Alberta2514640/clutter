package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

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

var invalidPlaybookChars = regexp.MustCompile(`[^a-z0-9._-]+`)
var repeatedDash = regexp.MustCompile(`-+`)

type createPlaybookUploadURLRequest struct {
	FileName string `json:"file_name"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
			"error":   err.Error(),
		})
	}

	var body createPlaybookUploadURLRequest
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid request body",
			"error":   err.Error(),
		})
	}

	normalizedFileName, err := validatePlaybookFileName(body.FileName)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": err.Error(),
		})
	}

	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "server configuration error: missing S3_BUCKET_NAME",
		})
	}

	uploadID := uuid.NewString()
	objectKey := buildPlaybookObjectKey(userData.Id, normalizedFileName, uploadID)

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to load AWS config",
			"error":   err.Error(),
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
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to create upload URL",
			"error":   err.Error(),
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

func validatePlaybookFileName(fileName string) (string, error) {
	trimmed := strings.TrimSpace(fileName)
	if trimmed == "" {
		return "", fmt.Errorf("file_name is required")
	}
	if strings.Contains(trimmed, "..") || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", fmt.Errorf("file_name contains an invalid path")
	}

	lower := strings.ToLower(trimmed)
	if !strings.HasSuffix(lower, ".yml") && !strings.HasSuffix(lower, ".yaml") {
		return "", fmt.Errorf("file_name must end with .yml or .yaml")
	}

	if len(trimmed) > 128 {
		return "", fmt.Errorf("file_name must be 128 characters or fewer")
	}

	return trimmed, nil
}

func buildPlaybookObjectKey(userID, fileName, uploadID string) string {
	lower := strings.ToLower(fileName)
	ext := ".yml"
	if strings.HasSuffix(lower, ".yaml") {
		ext = ".yaml"
	}

	baseName := strings.TrimSuffix(lower, ext)
	baseName = invalidPlaybookChars.ReplaceAllString(baseName, "-")
	baseName = repeatedDash.ReplaceAllString(baseName, "-")
	baseName = strings.Trim(baseName, "-.")
	if baseName == "" {
		baseName = "playbook"
	}

	return fmt.Sprintf("playbooks/%s/%s-%s%s", userID, uploadID, baseName, ext)
}
