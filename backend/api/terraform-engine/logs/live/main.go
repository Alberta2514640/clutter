package main

import (
	"context"
	"errors"
	"log"
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
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type LiveLogEvent struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

// What the user reads when the logs cannot be fetched. The cause stays in the
// log.
const liveLogsFailure = "failed to fetch the live logs"

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

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
	if err != nil || userData.Id == "" {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
		})
	}

	conn, err := connect()
	if err != nil {
		log.Printf("live logs: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userData.Id, orgID, projID, diagramID)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	clusterName := os.Getenv("ECS_CLUSTER_NAME")
	logGroup := os.Getenv("TERRAFORM_DEPLOYER_LOG_GROUP")
	streamPrefix := os.Getenv("TERRAFORM_DEPLOYER_LOG_STREAM_PREFIX")
	containerName := os.Getenv("TERRAFORM_DEPLOYER_CONTAINER_NAME")

	if clusterName == "" || logGroup == "" || streamPrefix == "" || containerName == "" {
		log.Printf("live logs: ECS_CLUSTER_NAME, TERRAFORM_DEPLOYER_LOG_GROUP, TERRAFORM_DEPLOYER_LOG_STREAM_PREFIX or TERRAFORM_DEPLOYER_CONTAINER_NAME is not set")
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": liveLogsFailure,
		})
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("live logs: load AWS config: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": liveLogsFailure,
		})
	}

	ecsClient := ecs.NewFromConfig(cfg)

	describeOutput, err := ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(clusterName),
		Tasks:   []string{taskArn},
	})
	if err != nil {
		log.Printf("live logs: describe task: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": liveLogsFailure,
		})
	}

	if len(describeOutput.Tasks) == 0 {
		return generic.Response(http.StatusNotFound, generic.Json{
			"message": "task not found",
		})
	}

	task := describeOutput.Tasks[0]
	// The ARN comes from the request, so the task must prove it was started for
	// this diagram before its status or logs are returned. Another diagram's
	// task gets the same answer as a task that does not exist.
	if !taskBelongsTo(task, ids.OrgID, ids.ProjectID, ids.DiagramID) {
		return generic.Response(http.StatusNotFound, generic.Json{
			"message": "task not found",
		})
	}
	taskStatus := aws.ToString(task.LastStatus)
	isComplete := taskStatus == "STOPPED"
	if isComplete {
		// The frontend stops polling this route once the task has stopped, so
		// the run's final status reaches its row now rather than on the next
		// recent-activity poll.
		generic.ReconcileDeployRunsFromEnv(ctx, conn, ids)
	}

	// The stream name comes from the checked task, never from the request text.
	taskID, err := extractTaskID(aws.ToString(task.TaskArn))
	if err != nil {
		log.Printf("live logs: task ARN from ECS: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": liveLogsFailure,
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
					"taskStatus": taskStatus,
					"isComplete": isComplete,
				},
			})
		}

		log.Printf("live logs: get log events: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": liveLogsFailure,
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
			"taskStatus": taskStatus,
			"isComplete": isComplete,
		},
	})
}

// taskBelongsTo reports whether one container of the task was started with
// this diagram's DIAGRAM_ID and TERRAFORM_DIRECTORY, the overrides
// terraform-command-runner sets when it starts a deploy. The deploy-log table
// does not store the task ARN, so the task's own overrides are the proof.
func taskBelongsTo(task ecstypes.Task, orgID, projID, diagramID string) bool {
	if task.Overrides == nil {
		return false
	}
	wantDirectory := orgID + "/" + projID + "/" + diagramID + "/terraform"
	for _, container := range task.Overrides.ContainerOverrides {
		var gotDiagram, gotDirectory bool
		for _, pair := range container.Environment {
			switch aws.ToString(pair.Name) {
			case "DIAGRAM_ID":
				gotDiagram = strings.EqualFold(aws.ToString(pair.Value), diagramID)
			case "TERRAFORM_DIRECTORY":
				gotDirectory = strings.EqualFold(aws.ToString(pair.Value), wantDirectory)
			}
		}
		if gotDiagram && gotDirectory {
			return true
		}
	}
	return false
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
