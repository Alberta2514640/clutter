package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type requestBody struct {
	OrgID              string `json:"org_id"`
	ProjectID          string `json:"project_id"`
	DiagramID          string `json:"diagram_id"`
	LambdaResourceName string `json:"lambda_resource_name"`
	Runtime            string `json:"runtime"`
}

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil || userData.Id == "" {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
		})
	}

	var body requestBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid JSON body",
		})
	}

	if body.OrgID == "" || body.ProjectID == "" || body.DiagramID == "" || body.LambdaResourceName == "" || body.Runtime == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "org_id, project_id, diagram_id, lambda_resource_name, and runtime are required",
		})
	}

	// The name becomes a segment of the S3 key and the Terraform resource name,
	// so it follows the generator's rule for resource names.
	if err := hclsafe.CheckName(body.LambdaResourceName); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "lambda_resource_name " + err.Error(),
		})
	}

	defaultHandler, ok := generic.LambdaRuntimes[body.Runtime]
	if !ok {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "unsupported runtime",
		})
	}

	conn, err := connect()
	if err != nil {
		log.Printf("code upload: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// The key below is built from these ids, so each must belong to the
	// caller's organization, and the key uses the checked, canonical ids.
	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userData.Id, body.OrgID, body.ProjectID, body.DiagramID)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	bucket := os.Getenv("S3_BUCKET_NAME")
	if bucket == "" {
		log.Printf("code upload: S3_BUCKET_NAME is not set")
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to generate the upload URL",
		})
	}

	s3Key := fmt.Sprintf("%s/%s/%s/terraform/code/%s/bootstrap.zip",
		ids.OrgID, ids.ProjectID, ids.DiagramID, body.LambdaResourceName,
	)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("code upload: load AWS config: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to generate the upload URL",
		})
	}

	presigner := s3.NewPresignClient(s3.NewFromConfig(cfg))

	presigned, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String(s3Key),
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		log.Printf("code upload: presign: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to generate the upload URL",
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"upload_url":      presigned.URL,
		"s3_bucket":       bucket,
		"s3_key":          s3Key,
		"default_handler": defaultHandler,
	})
}
