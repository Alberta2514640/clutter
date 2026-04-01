package main

import (
	"context"
	"encoding/json"
	"fmt"
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

var runtimeHandlers = map[string]string{
	"nodejs24":      "index.handler",
	"nodejs22":      "index.handler",
	"python3.14":      "lambda_function.lambda_handler",
	"python3.13":      "lambda_function.lambda_handler",
	"python3.12":      "lambda_function.lambda_handler",
	"provided.al2023": "bootstrap",
}

type requestBody struct {
	OrgID              string `json:"org_id"`
	ProjectID          string `json:"project_id"`
	DiagramID          string `json:"diagram_id"`
	LambdaResourceName string `json:"lambda_resource_name"`
	Runtime            string `json:"runtime"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "failed to get user from authorizer context",
			"error":   err.Error(),
		})
	}

	var body requestBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid JSON body",
			"error":   err.Error(),
		})
	}

	if body.OrgID == "" || body.ProjectID == "" || body.DiagramID == "" || body.LambdaResourceName == "" || body.Runtime == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "org_id, project_id, diagram_id, lambda_resource_name, and runtime are required",
		})
	}

	defaultHandler, ok := runtimeHandlers[body.Runtime]
	if !ok {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": fmt.Sprintf("unsupported runtime: %s", body.Runtime),
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

	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userData.Id, body.OrgID); err != nil {
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

	s3Key := fmt.Sprintf("%s/%s/%s/terraform/code/%s/bootstrap.zip",
		body.OrgID, body.ProjectID, body.DiagramID, body.LambdaResourceName,
	)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to load AWS config",
			"error":   err.Error(),
		})
	}

	presigner := s3.NewPresignClient(s3.NewFromConfig(cfg))

	presigned, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to generate presigned URL",
			"error":   err.Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"upload_url":      presigned.URL,
		"s3_bucket":       bucket,
		"s3_key":          s3Key,
		"default_handler": defaultHandler,
	})
}
