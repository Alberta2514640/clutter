// Package internal provides internal utilities for the terraform-deploy create Lambda.
package internal

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// LoadECSConfig loads ECS configuration from environment variables.
// Returns an error if any required configuration is missing.
func LoadECSConfig() (*ECSConfig, error) {
	clusterARN := os.Getenv("ECS_CLUSTER_ARN")
	if clusterARN == "" {
		return nil, fmt.Errorf("ECS_CLUSTER_ARN environment variable is required")
	}

	taskDefinitionARN := os.Getenv("ECS_TASK_DEFINITION_ARN")
	if taskDefinitionARN == "" {
		return nil, fmt.Errorf("ECS_TASK_DEFINITION_ARN environment variable is required")
	}

	subnetIDsStr := os.Getenv("ECS_SUBNET_IDS")
	if subnetIDsStr == "" {
		return nil, fmt.Errorf("ECS_SUBNET_IDS environment variable is required")
	}
	rawSubnetIDs := strings.Split(subnetIDsStr, ",")
	var subnetIDs []string
	for _, id := range rawSubnetIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			subnetIDs = append(subnetIDs, trimmed)
		}
	}
	if len(subnetIDs) == 0 {
		return nil, fmt.Errorf("ECS_SUBNET_IDS contains no valid subnet IDs")
	}

	securityGroupID := os.Getenv("ECS_SECURITY_GROUP_ID")
	if securityGroupID == "" {
		return nil, fmt.Errorf("ECS_SECURITY_GROUP_ID environment variable is required")
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET environment variable is required")
	}

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-east-1"
	}

	return &ECSConfig{
		ClusterARN:        clusterARN,
		TaskDefinitionARN: taskDefinitionARN,
		SubnetIDs:         subnetIDs,
		SecurityGroupID:   securityGroupID,
		S3Bucket:          s3Bucket,
		AWSRegion:         awsRegion,
	}, nil
}

// RunECSTask starts an ECS Fargate task for the Terraform deployment.
// It returns an error if the task fails to start.
func RunECSTask(ctx context.Context, cfg *ECSConfig, projectID, runID, userID string) error {
	// Validate config is not nil
	if cfg == nil {
		return fmt.Errorf("nil ECSConfig provided")
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.AWSRegion))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create ECS client
	ecsClient := ecs.NewFromConfig(awsCfg)

	// Build container overrides with environment variables
	containerOverrides := []types.ContainerOverride{
		{
			Name: aws.String("terraform-runner"),
			Environment: []types.KeyValuePair{
				{Name: aws.String("PROJECT_ID"), Value: aws.String(projectID)},
				{Name: aws.String("RUN_ID"), Value: aws.String(runID)},
				{Name: aws.String("USER_ID"), Value: aws.String(userID)},
				{Name: aws.String("S3_BUCKET"), Value: aws.String(cfg.S3Bucket)},
				{Name: aws.String("AWS_REGION"), Value: aws.String(cfg.AWSRegion)},
			},
		},
	}

	// Run the ECS task
	input := &ecs.RunTaskInput{
		Cluster:        aws.String(cfg.ClusterARN),
		TaskDefinition: aws.String(cfg.TaskDefinitionARN),
		LaunchType:     types.LaunchTypeFargate,
		Count:          aws.Int32(1),
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        cfg.SubnetIDs,
				SecurityGroups: []string{cfg.SecurityGroupID},
				AssignPublicIp: types.AssignPublicIpEnabled, // Required for public subnet (no NAT)
			},
		},
		Overrides: &types.TaskOverride{
			ContainerOverrides: containerOverrides,
		},
	}

	result, err := ecsClient.RunTask(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to run ECS task: %w", err)
	}

	// Check for failures
	if len(result.Failures) > 0 {
		failureReasons := make([]string, len(result.Failures))
		for i, f := range result.Failures {
			failureReasons[i] = aws.ToString(f.Reason)
		}
		return fmt.Errorf("ECS task failures: %s", strings.Join(failureReasons, "; "))
	}

	// Verify task was started
	if len(result.Tasks) == 0 {
		return fmt.Errorf("no tasks were started")
	}

	return nil
}
