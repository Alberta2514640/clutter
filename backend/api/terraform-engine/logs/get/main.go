package main

import (
	"context"
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

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to load AWS config",
			"error":   err.Error(),
		})
	}

	s3Client := s3.NewFromConfig(cfg)
	basePrefix := orgID + "/" + projID + "/" + diagramID + "/terraform/logs/"

	resp, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(basePrefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to list deployment folders",
			"error":   err.Error(),
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
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to list deployment files",
				"error":   err.Error(),
			})
		}

		files := make([]string, 0)
		for _, obj := range filesResp.Contents {
			key := aws.ToString(obj.Key)
			name := strings.TrimPrefix(key, fullPrefix)
			if name != "" && !strings.Contains(name, "/") {
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
