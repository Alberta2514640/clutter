package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	cloudwatchlogs "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type LiveLogEvent struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	orgID := request.QueryStringParameters["orgId"]
	projID := request.QueryStringParameters["projId"]
	diagramID := request.QueryStringParameters["diagramId"]
	taskArn := request.QueryStringParameters["taskArn"]
	nextToken := request.QueryStringParameters["nextToken"]

	if orgID == "" || projID == "" || diagramID == "" || taskArn == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "orgId, projId, diagramId, and taskArn are required",
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

	clusterName := os.Getenv("ECS_CLUSTER_NAME")
	logGroup := os.Getenv("TERRAFORM_DEPLOYER_LOG_GROUP")
	streamPrefix := os.Getenv("TERRAFORM_DEPLOYER_LOG_STREAM_PREFIX")
	containerName := os.Getenv("TERRAFORM_DEPLOYER_CONTAINER_NAME")

	if clusterName == "" || logGroup == "" || streamPrefix == "" || containerName == "" {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "required environment variables are missing",
		})
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to load AWS config",
			"error":   err.Error(),
		})
	}

	ecsClient := ecs.NewFromConfig(cfg)

	describeOutput, err := ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(clusterName),
		Tasks:   []string{taskArn},
	})
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to describe ECS task",
			"error":   err.Error(),
		})
	}

	if len(describeOutput.Tasks) == 0 {
		return generic.Response(http.StatusNotFound, generic.Json{
			"message": "task not found",
		})
	}

	task := describeOutput.Tasks[0]
	taskStatus := aws.ToString(task.LastStatus)
	isComplete := taskStatus == "STOPPED"

	taskID, err := extractTaskID(taskArn)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid taskArn",
			"error":   err.Error(),
		})
	}

	logStream := streamPrefix + "/" + containerName + "/" + taskID

	cwClient := cloudwatchlogs.NewFromConfig(cfg)

	getLogInput := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(logStream),
		StartFromHead: aws.Bool(true),
	}

	if nextToken != "" {
		getLogInput.NextToken = aws.String(nextToken)
	}

	logOutput, err := cwClient.GetLogEvents(ctx, getLogInput)
	if err != nil {
		var resourceNotFound *cwtypes.ResourceNotFoundException
		if errors.As(err, &resourceNotFound) {
			return generic.Response(http.StatusOK, generic.Json{
				"success": true,
				"data": generic.Json{
					"events":     []LiveLogEvent{},
					"nextToken":  "",
					"taskArn":    taskArn,
					"taskStatus": taskStatus,
					"isComplete": isComplete,
					"logGroup":   logGroup,
					"logStream":  logStream,
				},
			})
		}

		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch CloudWatch log events",
			"error":   err.Error(),
		})
	}

	eventsList := make([]LiveLogEvent, 0, len(logOutput.Events))
	for _, event := range logOutput.Events {
		eventsList = append(eventsList, LiveLogEvent{
			Timestamp: aws.ToInt64(event.Timestamp),
			Message:   aws.ToString(event.Message),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"success": true,
		"data": generic.Json{
			"events":     eventsList,
			"nextToken":  aws.ToString(logOutput.NextForwardToken),
			"taskArn":    taskArn,
			"taskStatus": taskStatus,
			"isComplete": isComplete,
			"logGroup":   logGroup,
			"logStream":  logStream,
		},
	})
}

func extractTaskID(taskArn string) (string, error) {
	parts := strings.Split(taskArn, "/")
	if len(parts) < 3 {
		return "", errors.New("unexpected ECS task ARN format")
	}

	taskID := parts[len(parts)-1]
	if taskID == "" {
		return "", errors.New("task ID is empty")
	}

	return taskID, nil
}

