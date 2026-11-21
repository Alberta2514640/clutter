package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-command-runner/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/jackc/pgx/v5"
)

// What the user reads when the run cannot start. The cause stays in the log.
const runFailure = "failed to start the run"

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

// taskRunner is the part of the ECS client the handler uses.
type taskRunner interface {
	RunTask(ctx context.Context, params *ecs.RunTaskInput, optFns ...func(*ecs.Options)) (*ecs.RunTaskOutput, error)
}

// newTaskRunner builds the ECS client. Tests replace it with a fake.
var newTaskRunner = func(ctx context.Context) (taskRunner, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(generic.PlatformRegion()))
	if err != nil {
		return nil, err
	}
	return ecs.NewFromConfig(cfg), nil
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context retains data about the user (extracted from the JWT provided in the Authorization header)
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil || userData.Id == "" {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
		})
	}
	userId := userData.Id

	// Parse request body
	var body internal.RequestBody

	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "invalid JSON body"})
	}
	organizationId := body.OrganizationId
	projectId := body.ProjectId
	diagramId := body.DiagramId
	accountAccessRoleId := body.AccountAccessRoleId
	region := body.Region
	command := body.Command
	// Set default region if not passed in
	if region == "" {
		region = generic.DefaultRegion()
	}

	// Every id goes into the S3 key or a query, so each must be a real UUID
	for _, id := range []string{organizationId, projectId, diagramId, accountAccessRoleId} {
		if !generic.IsValidUuid(id) {
			return generic.Response(http.StatusBadRequest, generic.Json{
				"message": "organizationId, projectId, diagramId and accountAccessRoleId must be valid UUIDs",
			})
		}
	}

	// The deployer runs only these two (its entrypoint refuses anything else),
	// so any other value is refused before a log row is written.
	if command != "apply" && command != "destroy" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "command must be apply or destroy",
		})
	}

	// Connect to PostgreSQL
	conn, err := connect()
	if err != nil {
		log.Printf("runner: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// Check user's membership to organization, and that the project and diagram
	// belong to it. The key, the log row and the task's overrides use the
	// checked, canonical ids from here on.
	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userId, organizationId, projectId, diagramId)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}
	organizationId, projectId, diagramId = ids.OrgID, ids.ProjectID, ids.DiagramID

	// Get Client Role ARN from DB
	roleArnQuery := `
		SELECT role_arn, external_id
		FROM aws_account_access_roles
		WHERE id = $1
			AND organization_id = $2
			AND status = 'complete';
	`
	var roleArn string
	var externalId string
	if err := conn.QueryRow(
		ctx,
		roleArnQuery,
		accountAccessRoleId,
		organizationId,
	).Scan(&roleArn, &externalId); err != nil {
		// The query is scoped to the organization, so another organization's account is not found
		if errors.Is(err, pgx.ErrNoRows) {
			return generic.Response(http.StatusNotFound, generic.Json{"message": "account not found"})
		}
		log.Printf("runner: read the account role: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to read the linked account",
		})
	}

	// Run terraform-deployer Fargate

	ecsClient, err := newTaskRunner(ctx)
	if err != nil {
		log.Printf("runner: load AWS config: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": runFailure})
	}

	// Define task parameters
	clusterName := os.Getenv("ECS_CLUSTER_NAME")
	taskDefinition := os.Getenv("TASK_DEFINITION_ARN")
	containerName := os.Getenv("CONTAINER_NAME")

	subnets := os.Getenv("SUBNET_IDS")
	securityGroup := os.Getenv("SECURITY_GROUP_ID")
	assignPublicIp := "ENABLED"

	// Create Terraform S3 key from given IDs
	// S3 key structure: <organizationId>/<projectId>/<diagramId>/terraform
	s3Key := fmt.Sprintf("%s/%s/%s/terraform", organizationId, projectId, diagramId)

	// Convert comma-separated subnets to slice (AWS SDK expects a slice of strings)
	subnetSlice := []string{}
	for _, s := range internal.SplitAndTrim(subnets) {
		subnetSlice = append(subnetSlice, s)
	}

	// Create deployment log record in table and get unique command ID from the function
	commandId, err := internal.InsertIntoLogTable(conn, ctx, diagramId, command)
	if err != nil {
		log.Printf("runner: write the deployment log row: %v", err)
		return generic.Response(http.StatusConflict, generic.Json{"message": "failed to record the run"})
	}

	// Build container overrides with dynamic environment variables. The task
	// gets this run's role and external id and nothing that reads the
	// database: with the database string it could read every organization's
	// external id and assume any customer's role. It reports its result in a
	// status file beside its logs (generic.ReconcileDeployRuns).
	containerOverrides := []types.ContainerOverride{
		{
			Name: aws.String(containerName),
			Environment: []types.KeyValuePair{
				{Name: aws.String("AWS_REGION"), Value: aws.String(region)},
				{Name: aws.String("TERRAFORM_DIRECTORY"), Value: aws.String(s3Key)},
				{Name: aws.String("CLIENT_ROLE_ARN"), Value: aws.String(roleArn)},
				{Name: aws.String("ASSUME_ROLE_EXTERNAL_ID"), Value: aws.String(externalId)},
				{Name: aws.String("COMMAND"), Value: aws.String(command)},
				{Name: aws.String("COMMAND_ID"), Value: aws.String(commandId)},
				// logs/live matches the task to the diagram by this value.
				{Name: aws.String("DIAGRAM_ID"), Value: aws.String(diagramId)},
			},
		},
	}

	// Initialize task input
	runTaskInput := &ecs.RunTaskInput{
		Cluster:        aws.String(clusterName),
		TaskDefinition: aws.String(taskDefinition),
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        subnetSlice,
				SecurityGroups: []string{securityGroup},
				AssignPublicIp: types.AssignPublicIp(assignPublicIp),
			},
		},
		Overrides: &types.TaskOverride{
			ContainerOverrides: containerOverrides,
		},
	}

	// Run the task and get the task run request output
	// The row was written as RUNNING before the task started. A task that does
	// not start writes no status file, so the row is marked FAILED here or it
	// would read RUNNING forever.
	RunTaskOutput, err := ecsClient.RunTask(ctx, runTaskInput)
	if err != nil {
		log.Printf("runner: run task: %v", err)
		internal.MarkRunNotStarted(conn, ctx, diagramId, commandId)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": runFailure})
	}
	// RunTask reports a task it could not place in Failures, not in err.
	if len(RunTaskOutput.Tasks) == 0 {
		for _, failure := range RunTaskOutput.Failures {
			log.Printf("runner: run task: %s %s", aws.ToString(failure.Reason), aws.ToString(failure.Detail))
		}
		internal.MarkRunNotStarted(conn, ctx, diagramId, commandId)
		return generic.Response(http.StatusInternalServerError, generic.Json{"message": runFailure})
	}

	return generic.Response(200, generic.Json{
		"message": "Terraform command fargate task started successfully",
		"data": generic.Json{
			"commandId": commandId,
			"taskArn":   *RunTaskOutput.Tasks[0].TaskArn,
		},
	})
}
