package main

import (
	"context"
	"encoding/json"
	"fmt"
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
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context retains data about the user (extracted from the JWT provided in the Authorization header)
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from authorizer context", "error": err.Error()},
		)
	}
	userId := userData.Id

	// Parse request body
	var body internal.RequestBody

	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request", "error": err.Error()})
	}
	organizationId := body.OrganizationId
	projectId := body.ProjectId
	diagramId := body.DiagramId
	accountAccessRoleId := body.AccountAccessRoleId
	region := body.Region
	command := body.Command
	// Set default region if not passed in
	if region == "" {
		region = generic.DefaultRegion
	}

	// Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	// Check user's membership to organization
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, organizationId); err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"message": authErr.Message,
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to check authorization",
			"error":   err.Error(),
		})
	}

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
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to retrieve role ARN",
			"error":   err.Error(),
		})
	}

	// Run terraform-deployer Fargate

	// Load config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(generic.DefaultRegion))
	if err != nil {
		return generic.Response(500, generic.Json{"message": "failed to load AWS config", "error": err.Error()})
	}

	// Create ECS client
	ecsClient := ecs.NewFromConfig(cfg)

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
		return generic.Response(http.StatusConflict, generic.Json{"message": "failed to insert into log table", "error": err.Error()})
	}

	// Build container overrides with dynamic environment variables
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
				{Name: aws.String("PSQL_CONNECTION_STRING"), Value: aws.String(os.Getenv("PSQL_CONNECTION_STRING"))},
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
	RunTaskOutput, err := ecsClient.RunTask(ctx, runTaskInput)
	if err != nil {
		return generic.Response(500, generic.Json{"message": "failed to run ECS task", "error": err.Error()})
	}

	return generic.Response(200, generic.Json{
		"message": "Terraform command fargate task started successfully",
		"data": generic.Json{
			"commandId": commandId,
			"taskArn":   *RunTaskOutput.Tasks[0].TaskArn,
		},
	})
}
