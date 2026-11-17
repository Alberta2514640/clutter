package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"regexp"
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

// deploymentIDPattern is the command id terraform-command-runner writes
// (generic.RandomID(8), stored in diagram_deployment_logs.command_id) and the
// deployer uses as the log folder name.
var deploymentIDPattern = regexp.MustCompile(`^[a-z0-9]{8}$`)

// What the user reads when the URL cannot be made. The cause stays in the log.
const urlFailure = "failed to generate the log URL"

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

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

	// The id becomes a segment of the S3 key, so a slash or ".." must never
	// reach it.
	if !deploymentIDPattern.MatchString(deploymentID) {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid deploymentId",
		})
	}

	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil || userData.Id == "" {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
		})
	}

	conn, err := connect()
	if err != nil {
		log.Printf("log URL: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// The key below is built from these ids, so each must belong to the
	// caller's organization, and the key uses the checked, canonical ids.
	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userData.Id, orgID, projID, diagramID)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	bucket := os.Getenv("S3_BUCKET_NAME")
	if bucket == "" {
		log.Printf("log URL: S3_BUCKET_NAME is not set")
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": urlFailure,
		})
	}

	key := ids.OrgID + "/" + ids.ProjectID + "/" + ids.DiagramID + "/terraform/logs/" + deploymentID + "/" + fileName

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("log URL: load AWS config: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": urlFailure,
		})
	}

	s3Client := s3.NewFromConfig(cfg)
	presigner := s3.NewPresignClient(s3Client)

	presigned, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		log.Printf("log URL: presign: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": urlFailure,
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
