package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/jackc/pgx/v5"
)

type ecsRuntimeConfig struct {
	clusterARN        string
	taskDefinitionARN string
	subnetIDsRaw      string
	securityGroupID   string
	s3BucketName      string
}

type runtimeConfig struct {
	ansible   ecsRuntimeConfig
	terraform ecsRuntimeConfig
	awsRegion string
}

// sanitizeError removes potentially sensitive information from error messages
func sanitizeError(err error) string {
	if err == nil {
		return "unknown error"
	}

	errMsg := err.Error()

	// List of patterns to redact
	sensitivePatterns := []string{
		"aws_access_key",
		"aws_secret_access_key",
		"password",
		"secret",
		"token",
		"AKIA", // AWS access key prefix
		"postgres://",
		"postgresql://",
	}

	redacted := errMsg
	for _, pattern := range sensitivePatterns {
		redacted = strings.ReplaceAll(redacted, pattern, "***REDACTED***")
	}

	// Truncate to prevent very long error messages
	if len(redacted) > 500 {
		redacted = redacted[:500] + "...(truncated)"
	}

	return redacted
}

func main() {
	lambda.Start(handler)
}

func loadRuntimeConfig() runtimeConfig {
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-west-2"
	}

	return runtimeConfig{
		ansible: ecsRuntimeConfig{
			clusterARN:        os.Getenv("ANSIBLE_ECS_CLUSTER_ARN"),
			taskDefinitionARN: os.Getenv("ANSIBLE_TASK_DEFINITION_ARN"),
			subnetIDsRaw:      os.Getenv("ANSIBLE_SUBNET_IDS"),
			securityGroupID:   os.Getenv("ANSIBLE_SECURITY_GROUP_ID"),
			s3BucketName:      os.Getenv("S3_BUCKET_NAME"),
		},
		terraform: ecsRuntimeConfig{
			clusterARN:        os.Getenv("TERRAFORM_ECS_CLUSTER_ARN"),
			taskDefinitionARN: os.Getenv("TERRAFORM_TASK_DEFINITION_ARN"),
			subnetIDsRaw:      os.Getenv("TERRAFORM_SUBNET_IDS"),
			securityGroupID:   os.Getenv("TERRAFORM_SECURITY_GROUP_ID"),
		},
		awsRegion: awsRegion,
	}
}

// validateAnsibleMessage checks that all required Ansible SQS message fields are
// present and that playbook_s3_key does not contain path traversal sequences.
func validateAnsibleMessage(msg map[string]string) error {
	if msg["playbook_s3_key"] == "" || msg["target_instance_ids"] == "" {
		return fmt.Errorf("missing required Ansible fields: playbook_s3_key or target_instance_ids")
	}
	if strings.Contains(msg["playbook_s3_key"], "..") ||
		strings.Contains(msg["playbook_s3_key"], "~") ||
		strings.HasPrefix(msg["playbook_s3_key"], "/") ||
		strings.HasPrefix(msg["playbook_s3_key"], "\\") {
		return fmt.Errorf("invalid playbook_s3_key: path traversal not allowed")
	}
	return nil
}

// validateTerraformMessage checks that all required Terraform SQS message fields
// are present and that terraform_directory does not contain path traversal sequences.
func validateTerraformMessage(msg map[string]string) error {
	if msg["terraform_directory"] == "" || msg["role_arn"] == "" || msg["assume_role_external_id"] == "" {
		return fmt.Errorf("missing required Terraform fields")
	}
	if strings.Contains(msg["terraform_directory"], "..") ||
		strings.Contains(msg["terraform_directory"], "~") ||
		strings.HasPrefix(msg["terraform_directory"], "/") ||
		strings.HasPrefix(msg["terraform_directory"], "\\") {
		return fmt.Errorf("invalid terraform_directory: path traversal not allowed")
	}
	return nil
}

// buildAnsibleRunTaskInput constructs the ECS RunTaskInput for an Ansible job.
// target_instance_ids may be a JSON array (["i-abc", "i-def"]) or a raw
// comma-separated string; both are normalised to a comma-separated scalar.
func buildAnsibleRunTaskInput(clusterARN, taskDefARN, subnetIDsRaw, securityGroupID, s3BucketName string, msg map[string]string) *ecs.RunTaskInput {
	raw := msg["target_instance_ids"]
	targetInstanceIDs := raw
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		targetInstanceIDs = strings.Join(list, ",")
	}

	return &ecs.RunTaskInput{
		Cluster:        aws.String(clusterARN),
		TaskDefinition: aws.String(taskDefARN),
		LaunchType:     ecstypes.LaunchTypeFargate,
		Count:          aws.Int32(1),
		NetworkConfiguration: &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        strings.Split(subnetIDsRaw, ","),
				SecurityGroups: []string{securityGroupID},
				AssignPublicIp: ecstypes.AssignPublicIpEnabled,
			},
		},
		Overrides: &ecstypes.TaskOverride{
			ContainerOverrides: []ecstypes.ContainerOverride{
				{
					Name: aws.String("ansible-executor"),
					Environment: []ecstypes.KeyValuePair{
						{Name: aws.String("JOB_ID"), Value: aws.String(msg["job_id"])},
						{Name: aws.String("PLAYBOOK_S3_KEY"), Value: aws.String(msg["playbook_s3_key"])},
						{Name: aws.String("TARGET_INSTANCE_IDS"), Value: aws.String(targetInstanceIDs)},
						{Name: aws.String("EXTRA_VARS"), Value: aws.String(msg["extra_vars"])},
						{Name: aws.String("S3_BUCKET_NAME"), Value: aws.String(s3BucketName)},
					},
				},
			},
		},
	}
}

// buildTerraformRunTaskInput constructs the ECS RunTaskInput for a Terraform job.
func buildTerraformRunTaskInput(clusterARN, taskDefARN, subnetIDsRaw, securityGroupID, awsRegion string, msg map[string]string) *ecs.RunTaskInput {
	return &ecs.RunTaskInput{
		Cluster:        aws.String(clusterARN),
		TaskDefinition: aws.String(taskDefARN),
		LaunchType:     ecstypes.LaunchTypeFargate,
		Count:          aws.Int32(1),
		NetworkConfiguration: &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        strings.Split(subnetIDsRaw, ","),
				SecurityGroups: []string{securityGroupID},
				AssignPublicIp: ecstypes.AssignPublicIpEnabled,
			},
		},
		Overrides: &ecstypes.TaskOverride{
			ContainerOverrides: []ecstypes.ContainerOverride{
				{
					Name: aws.String("terraform-deployer"),
					Environment: []ecstypes.KeyValuePair{
						{Name: aws.String("JOB_ID"), Value: aws.String(msg["job_id"])},
						{Name: aws.String("AWS_REGION"), Value: aws.String(awsRegion)},
						{Name: aws.String("TERRAFORM_DIRECTORY"), Value: aws.String(msg["terraform_directory"])},
						{Name: aws.String("CLIENT_ROLE_ARN"), Value: aws.String(msg["role_arn"])},
						{Name: aws.String("ASSUME_ROLE_EXTERNAL_ID"), Value: aws.String(msg["assume_role_external_id"])},
						{Name: aws.String("EXTRA_VARS"), Value: aws.String(msg["extra_vars"])},
					},
				},
			},
		},
	}
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	cfg := loadRuntimeConfig()

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.SQSEventResponse{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ecsClient := ecs.NewFromConfig(awsCfg)

	// Connect to PostgreSQL using shared helper
	conn, err := generic.PsqlConnect()
	if err != nil {
		return events.SQSEventResponse{}, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	var batchItemFailures []events.SQSBatchItemFailure

	for _, record := range sqsEvent.Records {
		// Parse SQS message
		var msg map[string]string
		if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
			log.Printf("ERROR: failed to parse SQS message: %v", err)
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue
		}

		jobID := msg["job_id"]
		jobType := msg["job_type"] // "ansible" or "terraform"

		// Default to ansible for backwards compatibility
		if jobType == "" {
			jobType = "ansible"
		}

		log.Printf("Processing job %s (type: %s)", jobID, jobType)

		// Route to appropriate handler
		if jobType == "terraform" {
			err = handleTerraformJob(
				ctx,
				conn,
				ecsClient,
				msg,
				cfg.terraform.clusterARN,
				cfg.terraform.taskDefinitionARN,
				cfg.terraform.subnetIDsRaw,
				cfg.terraform.securityGroupID,
				cfg.awsRegion,
			)
		} else {
			err = handleAnsibleJob(
				ctx,
				conn,
				ecsClient,
				msg,
				cfg.ansible.clusterARN,
				cfg.ansible.taskDefinitionARN,
				cfg.ansible.subnetIDsRaw,
				cfg.ansible.securityGroupID,
				cfg.ansible.s3BucketName,
			)
		}

		if err != nil {
			log.Printf("ERROR: failed to process job %s: %v", jobID, err)
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	return events.SQSEventResponse{BatchItemFailures: batchItemFailures}, nil
}

func handleAnsibleJob(
	ctx context.Context,
	conn *pgx.Conn,
	ecsClient *ecs.Client,
	msg map[string]string,
	clusterARN, taskDefARN, subnetIDsRaw, securityGroupID, s3BucketName string,
) error {
	if clusterARN == "" || taskDefARN == "" || subnetIDsRaw == "" || securityGroupID == "" {
		return fmt.Errorf("missing required Ansible environment variables")
	}

	jobID := msg["job_id"]

	if err := validateAnsibleMessage(msg); err != nil {
		updateJobStatus(ctx, conn, jobID, "FAILED", err.Error())
		return err
	}

	log.Printf("Processing Ansible job %s: playbook=%s, targets=%s",
		jobID, msg["playbook_s3_key"], msg["target_instance_ids"])

	if err := updateJobStatus(ctx, conn, jobID, "STARTING", ""); err != nil {
		log.Printf("ERROR: failed to update job status for %s: %v", jobID, err)
	}

	runInput := buildAnsibleRunTaskInput(clusterARN, taskDefARN, subnetIDsRaw, securityGroupID, s3BucketName, msg)

	result, err := ecsClient.RunTask(ctx, runInput)
	if err != nil {
		sanitizedErr := sanitizeError(err)
		updateJobStatus(ctx, conn, jobID, "FAILED", sanitizedErr)
		return fmt.Errorf("failed to launch Fargate task: %w", err)
	}

	if len(result.Failures) > 0 {
		var failureMsgs []string
		for _, f := range result.Failures {
			failureMsgs = append(failureMsgs, fmt.Sprintf("arn=%s reason=%s detail=%s",
				aws.ToString(f.Arn), aws.ToString(f.Reason), aws.ToString(f.Detail)))
		}
		aggregatedFailure := strings.Join(failureMsgs, "; ")
		sanitizedErr := sanitizeError(errors.New(aggregatedFailure))
		updateJobStatus(ctx, conn, jobID, "FAILED", sanitizedErr)
		return fmt.Errorf("RunTask returned failures: %s", aggregatedFailure)
	}

	if len(result.Tasks) > 0 {
		taskARN := *result.Tasks[0].TaskArn
		log.Printf("Launched Ansible Fargate task %s for job %s", taskARN, jobID)
		updateJobWithTaskArn(ctx, conn, jobID, taskARN)
	}

	return nil
}

func handleTerraformJob(
	ctx context.Context,
	conn *pgx.Conn,
	ecsClient *ecs.Client,
	msg map[string]string,
	clusterARN, taskDefARN, subnetIDsRaw, securityGroupID, awsRegion string,
) error {
	if clusterARN == "" || taskDefARN == "" || subnetIDsRaw == "" || securityGroupID == "" {
		return fmt.Errorf("missing required Terraform environment variables")
	}

	jobID := msg["job_id"]

	if err := validateTerraformMessage(msg); err != nil {
		updateJobStatus(ctx, conn, jobID, "FAILED", err.Error())
		return err
	}

	log.Printf("Processing Terraform job %s: directory=%s, role=%s",
		jobID, msg["terraform_directory"], msg["role_arn"])

	if err := updateJobStatus(ctx, conn, jobID, "STARTING", ""); err != nil {
		log.Printf("ERROR: failed to update job status for %s: %v", jobID, err)
	}

	runInput := buildTerraformRunTaskInput(clusterARN, taskDefARN, subnetIDsRaw, securityGroupID, awsRegion, msg)

	result, err := ecsClient.RunTask(ctx, runInput)
	if err != nil {
		sanitizedErr := sanitizeError(err)
		updateJobStatus(ctx, conn, jobID, "FAILED", sanitizedErr)
		return fmt.Errorf("failed to launch Fargate task: %w", err)
	}

	if len(result.Failures) > 0 {
		var failureMsgs []string
		for _, f := range result.Failures {
			failureMsgs = append(failureMsgs, fmt.Sprintf("arn=%s reason=%s detail=%s",
				aws.ToString(f.Arn), aws.ToString(f.Reason), aws.ToString(f.Detail)))
		}
		aggregatedFailure := strings.Join(failureMsgs, "; ")
		sanitizedErr := sanitizeError(errors.New(aggregatedFailure))
		updateJobStatus(ctx, conn, jobID, "FAILED", sanitizedErr)
		return fmt.Errorf("RunTask returned failures: %s", aggregatedFailure)
	}

	if len(result.Tasks) > 0 {
		taskARN := *result.Tasks[0].TaskArn
		log.Printf("Launched Terraform Fargate task %s for job %s", taskARN, jobID)
		updateJobWithTaskArn(ctx, conn, jobID, taskARN)
	}

	return nil
}

func updateJobStatus(ctx context.Context, conn *pgx.Conn, jobID, status, errorMessage string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	if errorMessage != "" {
		_, err := conn.Exec(ctx, `
			UPDATE jobs
			SET status = $1, updated_at = $2, error_message = $3
			WHERE id = $4
		`, status, now, errorMessage, jobID)
		return err
	}

	_, err := conn.Exec(ctx, `
		UPDATE jobs
		SET status = $1, updated_at = $2
		WHERE id = $3
	`, status, now, jobID)
	return err
}

func updateJobWithTaskArn(ctx context.Context, conn *pgx.Conn, jobID, taskArn string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := conn.Exec(ctx, `
		UPDATE jobs
		SET task_arn = $1, updated_at = $2
		WHERE id = $3
	`, taskArn, now, jobID)
	return err
}
