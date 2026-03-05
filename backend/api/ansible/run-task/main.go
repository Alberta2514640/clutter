package main

import (
	"context"
	"encoding/json"
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

func handler(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {

	// Ansible environment variables
	ansibleClusterARN := os.Getenv("ANSIBLE_ECS_CLUSTER_ARN")
	ansibleTaskDefARN := os.Getenv("ANSIBLE_TASK_DEFINITION_ARN")
	ansibleSubnetIDsRaw := os.Getenv("ANSIBLE_SUBNET_IDS")
	ansibleSecurityGroupID := os.Getenv("ANSIBLE_SECURITY_GROUP_ID")
	ansibleS3BucketName := os.Getenv("S3_BUCKET_NAME")

	// Terraform environment variables
	terraformClusterARN := os.Getenv("TERRAFORM_ECS_CLUSTER_ARN")
	terraformTaskDefARN := os.Getenv("TERRAFORM_TASK_DEFINITION_ARN")
	terraformSubnetIDsRaw := os.Getenv("TERRAFORM_SUBNET_IDS")
	terraformSecurityGroupID := os.Getenv("TERRAFORM_SECURITY_GROUP_ID")

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-west-2"
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.SQSEventResponse{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ecsClient := ecs.NewFromConfig(cfg)

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
			err = handleTerraformJob(ctx, conn, ecsClient, msg, terraformClusterARN, terraformTaskDefARN, terraformSubnetIDsRaw, terraformSecurityGroupID, awsRegion)
		} else {
			err = handleAnsibleJob(ctx, conn, ecsClient, msg, ansibleClusterARN, ansibleTaskDefARN, ansibleSubnetIDsRaw, ansibleSecurityGroupID, ansibleS3BucketName)
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
	// Validate Ansible-specific env vars
	if clusterARN == "" || taskDefARN == "" || subnetIDsRaw == "" || securityGroupID == "" {
		return fmt.Errorf("missing required Ansible environment variables")
	}

	jobID := msg["job_id"]
	playbookS3Key := msg["playbook_s3_key"]
	rawTargetInstanceIDs := msg["target_instance_ids"]
	extraVars := msg["extra_vars"]

	// Validate required fields
	if playbookS3Key == "" || rawTargetInstanceIDs == "" {
		return fmt.Errorf("missing required Ansible fields: playbook_s3_key or target_instance_ids")
	}

	// Validate playbook_s3_key to prevent path traversal
	if strings.Contains(playbookS3Key, "..") ||
		strings.Contains(playbookS3Key, "~") ||
		strings.HasPrefix(playbookS3Key, "/") ||
		strings.HasPrefix(playbookS3Key, "\\") {
		err := updateJobStatus(ctx, conn, jobID, "FAILED", "invalid playbook_s3_key: path traversal not allowed")
		return err
	}

	// Convert target_instance_ids from JSON array to comma-separated string
	var targetInstanceIDsList []string
	var targetInstanceIDs string
	if err := json.Unmarshal([]byte(rawTargetInstanceIDs), &targetInstanceIDsList); err == nil {
		targetInstanceIDs = strings.Join(targetInstanceIDsList, ",")
	} else {
		targetInstanceIDs = rawTargetInstanceIDs
	}

	log.Printf("Processing Ansible job %s: playbook=%s, targets=%s", jobID, playbookS3Key, targetInstanceIDs)

	// Update job status to STARTING
	if err := updateJobStatus(ctx, conn, jobID, "STARTING", ""); err != nil {
		log.Printf("ERROR: failed to update job status for %s: %v", jobID, err)
	}

	// Parse subnet IDs
	subnetIDs := strings.Split(subnetIDsRaw, ",")

	// Launch Fargate task
	runInput := &ecs.RunTaskInput{
		Cluster:        &clusterARN,
		TaskDefinition: &taskDefARN,
		LaunchType:     ecstypes.LaunchTypeFargate,
		Count:          aws.Int32(1),
		NetworkConfiguration: &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        subnetIDs,
				SecurityGroups: []string{securityGroupID},
				AssignPublicIp: ecstypes.AssignPublicIpEnabled,
			},
		},
		Overrides: &ecstypes.TaskOverride{
			ContainerOverrides: []ecstypes.ContainerOverride{
				{
					Name: aws.String("ansible-executor"),
					Environment: []ecstypes.KeyValuePair{
						{Name: aws.String("JOB_ID"), Value: aws.String(jobID)},
						{Name: aws.String("PLAYBOOK_S3_KEY"), Value: aws.String(playbookS3Key)},
						{Name: aws.String("TARGET_INSTANCE_IDS"), Value: aws.String(targetInstanceIDs)},
						{Name: aws.String("EXTRA_VARS"), Value: aws.String(extraVars)},
						{Name: aws.String("S3_BUCKET_NAME"), Value: aws.String(s3BucketName)},
					},
				},
			},
		},
	}

	result, err := ecsClient.RunTask(ctx, runInput)
	if err != nil {
		sanitizedErr := sanitizeError(err)
		updateJobStatus(ctx, conn, jobID, "FAILED", sanitizedErr)
		return fmt.Errorf("failed to launch Fargate task: %w", err)
	}

	if len(result.Failures) > 0 {
		var failureMsgs []string
		for _, f := range result.Failures {
			failureMsg := fmt.Sprintf("arn=%s reason=%s detail=%s",
				aws.ToString(f.Arn), aws.ToString(f.Reason), aws.ToString(f.Detail))
			failureMsgs = append(failureMsgs, failureMsg)
		}
		aggregatedFailure := strings.Join(failureMsgs, "; ")
		sanitizedErr := sanitizeError(fmt.Errorf(aggregatedFailure))
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
	// Validate Terraform-specific env vars
	if clusterARN == "" || taskDefARN == "" || subnetIDsRaw == "" || securityGroupID == "" {
		return fmt.Errorf("missing required Terraform environment variables")
	}

	jobID := msg["job_id"]
	terraformDirectory := msg["terraform_directory"]
	roleArn := msg["role_arn"]
	assumeRoleExternalId := msg["assume_role_external_id"]
	extraVars := msg["extra_vars"]

	// Validate required fields
	if terraformDirectory == "" || roleArn == "" || assumeRoleExternalId == "" {
		return fmt.Errorf("missing required Terraform fields")
	}

	// Validate terraform_directory to prevent path traversal
	if strings.Contains(terraformDirectory, "..") ||
		strings.Contains(terraformDirectory, "~") ||
		strings.HasPrefix(terraformDirectory, "/") ||
		strings.HasPrefix(terraformDirectory, "\\") {
		err := updateJobStatus(ctx, conn, jobID, "FAILED", "invalid terraform_directory: path traversal not allowed")
		return err
	}

	log.Printf("Processing Terraform job %s: directory=%s, role=%s", jobID, terraformDirectory, roleArn)

	// Update job status to STARTING
	if err := updateJobStatus(ctx, conn, jobID, "STARTING", ""); err != nil {
		log.Printf("ERROR: failed to update job status for %s: %v", jobID, err)
	}

	// Parse subnet IDs
	subnetIDs := strings.Split(subnetIDsRaw, ",")

	// Launch Terraform Fargate task
	runInput := &ecs.RunTaskInput{
		Cluster:        &clusterARN,
		TaskDefinition: &taskDefARN,
		LaunchType:     ecstypes.LaunchTypeFargate,
		Count:          aws.Int32(1),
		NetworkConfiguration: &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        subnetIDs,
				SecurityGroups: []string{securityGroupID},
				AssignPublicIp: ecstypes.AssignPublicIpEnabled,
			},
		},
		Overrides: &ecstypes.TaskOverride{
			ContainerOverrides: []ecstypes.ContainerOverride{
				{
					Name: aws.String("terraform-deployer"),
					Environment: []ecstypes.KeyValuePair{
						{Name: aws.String("JOB_ID"), Value: aws.String(jobID)},
						{Name: aws.String("AWS_REGION"), Value: aws.String(awsRegion)},
						{Name: aws.String("TERRAFORM_DIRECTORY"), Value: aws.String(terraformDirectory)},
						{Name: aws.String("CLIENT_ROLE_ARN"), Value: aws.String(roleArn)},
						{Name: aws.String("ASSUME_ROLE_EXTERNAL_ID"), Value: aws.String(assumeRoleExternalId)},
						{Name: aws.String("EXTRA_VARS"), Value: aws.String(extraVars)},
					},
				},
			},
		},
	}

	result, err := ecsClient.RunTask(ctx, runInput)
	if err != nil {
		sanitizedErr := sanitizeError(err)
		updateJobStatus(ctx, conn, jobID, "FAILED", sanitizedErr)
		return fmt.Errorf("failed to launch Fargate task: %w", err)
	}

	if len(result.Failures) > 0 {
		var failureMsgs []string
		for _, f := range result.Failures {
			failureMsg := fmt.Sprintf("arn=%s reason=%s detail=%s",
				aws.ToString(f.Arn), aws.ToString(f.Reason), aws.ToString(f.Detail))
			failureMsgs = append(failureMsgs, failureMsg)
		}
		aggregatedFailure := strings.Join(failureMsgs, "; ")
		sanitizedErr := sanitizeError(fmt.Errorf(aggregatedFailure))
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
