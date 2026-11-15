package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type DeploymentLogs struct {
	DeploymentID string   `json:"deploymentId"`
	Files        []string `json:"files"`
}

// What the user reads when the log list cannot be read. The cause stays in
// the log.
const listFailure = "failed to list the deployment logs"

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	orgID := request.QueryStringParameters["orgId"]
	projID := request.QueryStringParameters["projId"]
	diagramID := request.QueryStringParameters["diagramId"]

	if orgID == "" || projID == "" || diagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "orgId, projId, and diagramId are required",
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
		log.Printf("deployment logs: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// The S3 prefix below is built from these ids, so each must belong to the
	// caller's organization, and the prefix uses the checked, canonical ids.
	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userData.Id, orgID, projID, diagramID)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	bucket := os.Getenv("S3_BUCKET_NAME")
	if bucket == "" {
		log.Printf("deployment logs: S3_BUCKET_NAME is not set")
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": listFailure,
		})
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("deployment logs: load AWS config: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": listFailure,
		})
	}

	s3Client := s3.NewFromConfig(cfg)
	basePrefix := ids.OrgID + "/" + ids.ProjectID + "/" + ids.DiagramID + "/terraform/logs/"

	resp, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(basePrefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		log.Printf("deployment logs: list deployment folders: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": listFailure,
		})
	}

	deployments := make([]DeploymentLogs, 0)

	for _, cp := range resp.CommonPrefixes {
		fullPrefix := aws.ToString(cp.Prefix)
		deploymentID := strings.TrimSuffix(strings.TrimPrefix(fullPrefix, basePrefix), "/")
		if deploymentID == "" {
			continue
		}

		filesResp, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String(fullPrefix),
		})
		if err != nil {
			log.Printf("deployment logs: list deployment files: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": listFailure,
			})
		}

		files := make([]string, 0)
		for _, obj := range filesResp.Contents {
			key := aws.ToString(obj.Key)
			name := strings.TrimPrefix(key, fullPrefix)
			// The run's status file sits beside its logs but is not a log the
			// user can open (logs/url serves init.log and command.log only).
			if name != "" && name != generic.DeployStatusFile && !strings.Contains(name, "/") {
				files = append(files, name)
			}
		}

		sort.Strings(files)

		deployments = append(deployments, DeploymentLogs{
			DeploymentID: deploymentID,
			Files:        files,
		})
	}

	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].DeploymentID > deployments[j].DeploymentID
	})

	return generic.Response(http.StatusOK, generic.Json{
		"success": true,
		"data": generic.Json{
			"deployments": deployments,
		},
	})
}
