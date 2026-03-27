package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var allowedFiles = map[string]bool{
	"init.log":    true,
	"command.log": true,
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	orgID := request.QueryStringParameters["orgId"]
	projID := request.QueryStringParameters["projId"]
	diagramID := request.QueryStringParameters["diagramId"]
	deploymentID := request.QueryStringParameters["deploymentId"]
	fileName := request.QueryStringParameters["file"]

	if orgID == "" || projID == "" || diagramID == "" || deploymentID == "" || fileName == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "orgId, projId, diagramId, deploymentId, and file are required",
		})
	}

	if !allowedFiles[fileName] {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid file requested",
		})
	}

	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "failed to get user from authorizer context",
			"error":   err.Error(),
		})
	}

	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userData.Id, orgID); err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"message": authErr.Message,
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to verify organization membership",
			"error":   err.Error(),
		})
	}

	bucket := os.Getenv("S3_BUCKET_NAME")
	if bucket == "" {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "S3_BUCKET_NAME is not configured",
		})
	}

	key := orgID + "/" + projID + "/" + diagramID + "/terraform/logs/" + deploymentID + "/" + fileName

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to load AWS config",
			"error":   err.Error(),
		})
	}

	s3Client := s3.NewFromConfig(cfg)
	presigner := s3.NewPresignClient(s3Client)

	presigned, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to generate signed URL",
			"error":   err.Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"success": true,
		"data": generic.Json{
			"url":       presigned.URL,
			"expiresIn": 900,
			"file":      fileName,
		},
	})
}
