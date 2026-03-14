package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
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
var sensitivePattern = regexp.MustCompile(`\b\d{12}\b|arn:aws:[^\s,]+|i-[0-9a-f]+|sg-[0-9a-f]+|subnet-[0-9a-f]+|vpc-[0-9a-f]+`)
var sensitiveKeywords = regexp.MustCompile(`(?i)\b(password|postgresql://|postgres://|token|secret|AKIA)[^\s,]*`)

const maxErrorMessageLength = 500

func sanitizeError(err error) string {
	if err == nil {
		return "unknown error"
	}
	msg := err.Error()

	// Redact AWS-specific patterns first (before truncation)
	msg = sensitivePattern.ReplaceAllString(msg, "***REDACTED***")

	// Redact sensitive keywords (before truncation)
	msg = sensitiveKeywords.ReplaceAllString(msg, "***REDACTED***")

	// Truncate long messages after redaction
	if len(msg) > maxErrorMessageLength {
		msg = msg[:maxErrorMessageLength] + "...(truncated)"
	}

	return msg
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

// filterEmpty removes empty and whitespace-only strings from a slice, trimming whitespace.
func filterEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// buildAnsibleRunTaskInput constructs the ECS RunTaskInput for an Ansible job.
// target_instance_ids may be a JSON array (["i-abc", "i-def"]) or a raw
// comma-separated string; both are normalised to a comma-separated scalar.
func buildAnsibleRunTaskInput(clusterARN, taskDefARN string, subnets []string, securityGroupID, s3BucketName string, msg map[string]string) *ecs.RunTaskInput {
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
				Subnets:        subnets,
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
func buildTerraformRunTaskInput(clusterARN, taskDefARN string, subnets []string, securityGroupID, awsRegion string, msg map[string]string) *ecs.RunTaskInput {
	return &ecs.RunTaskInput{
		Cluster:        aws.String(clusterARN),
		TaskDefinition: aws.String(taskDefARN),
		LaunchType:     ecstypes.LaunchTypeFargate,
		Count:          aws.Int32(1),
		NetworkConfiguration: &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        subnets,
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

	conn, err := generic.PsqlConnect()
	if err != nil {
		return events.SQSEventResponse{}, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	var batchItemFailures []events.SQSBatchItemFailure

	for _, record := range sqsEvent.Records {
		var msg map[string]string
		if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
			log.Printf("ERROR: failed to parse SQS message: %v", err)
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue
		}

		jobID := msg["job_id"]
		jobType := msg["job_type"]

		if jobID == "" {
			log.Printf("ERROR: SQS message missing job_id")
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue
		}

		if jobType == "" {
			jobType = "ansible"
		}

		log.Printf("Processing job %s (type: %s)", jobID, jobType)

		if err := processJob(ctx, conn, ecsClient, msg, jobType, cfg); err != nil {
			log.Printf("ERROR: failed to process job %s: %v", jobID, err)
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	return events.SQSEventResponse{BatchItemFailures: batchItemFailures}, nil
}

// jobConfig holds the configuration needed to process a job
type jobConfig struct {
	clusterARN        string
	taskDefinitionARN string
	subnetIDsRaw      string
	securityGroupID   string
	s3BucketName      string
	awsRegion         string
}

// processJob routes a job to the appropriate handler based on type
func processJob(ctx context.Context, conn *pgx.Conn, ecsClient *ecs.Client, msg map[string]string, jobType string, cfg runtimeConfig) error {
	var jobCfg jobConfig
	var validateFn func(map[string]string) error
	var logFn func()
	var buildInputFn func() *ecs.RunTaskInput
	var launchLogFn func(string)

	jobID := msg["job_id"]

	if jobType == "terraform" {
		jobCfg = jobConfig{
			clusterARN:        cfg.terraform.clusterARN,
			taskDefinitionARN: cfg.terraform.taskDefinitionARN,
			subnetIDsRaw:      cfg.terraform.subnetIDsRaw,
			securityGroupID:   cfg.terraform.securityGroupID,
			awsRegion:         cfg.awsRegion,
		}
		validateFn = validateTerraformMessage
		logFn = func() {
			log.Printf("Processing Terraform job %s: directory=%s, role=%s",
				jobID, msg["terraform_directory"], msg["role_arn"])
		}
		buildInputFn = func() *ecs.RunTaskInput {
			subnets := filterEmpty(strings.Split(jobCfg.subnetIDsRaw, ","))
			return buildTerraformRunTaskInput(jobCfg.clusterARN, jobCfg.taskDefinitionARN, subnets, jobCfg.securityGroupID, jobCfg.awsRegion, msg)
		}
		launchLogFn = func(taskARN string) {
			log.Printf("Launched Terraform Fargate task %s for job %s", taskARN, jobID)
		}
	} else {
		jobCfg = jobConfig{
			clusterARN:        cfg.ansible.clusterARN,
			taskDefinitionARN: cfg.ansible.taskDefinitionARN,
			subnetIDsRaw:      cfg.ansible.subnetIDsRaw,
			securityGroupID:   cfg.ansible.securityGroupID,
			s3BucketName:      cfg.ansible.s3BucketName,
		}
		validateFn = validateAnsibleMessage
		logFn = func() {
			log.Printf("Processing Ansible job %s: playbook=%s, targets=%s",
				jobID, msg["playbook_s3_key"], msg["target_instance_ids"])
		}
		buildInputFn = func() *ecs.RunTaskInput {
			subnets := filterEmpty(strings.Split(jobCfg.subnetIDsRaw, ","))
			return buildAnsibleRunTaskInput(jobCfg.clusterARN, jobCfg.taskDefinitionARN, subnets, jobCfg.securityGroupID, jobCfg.s3BucketName, msg)
		}
		launchLogFn = func(taskARN string) {
			log.Printf("Launched Ansible Fargate task %s for job %s", taskARN, jobID)
		}
	}

	if err := validateEnvVars(jobCfg); err != nil {
		return err
	}

	if err := validateFn(msg); err != nil {
		updateJobStatus(ctx, conn, jobID, "FAILED", err.Error())
		return err
	}

	logFn()

	if err := updateJobStatus(ctx, conn, jobID, "STARTING", ""); err != nil {
		log.Printf("ERROR: failed to update job status for %s: %v", jobID, err)
	}

	subnets := filterEmpty(strings.Split(jobCfg.subnetIDsRaw, ","))
	if len(subnets) == 0 {
		updateJobStatus(ctx, conn, jobID, "FAILED", "no valid subnet IDs configured")
		return fmt.Errorf("no valid subnet IDs configured")
	}

	return launchECSTask(ctx, conn, ecsClient, jobID, buildInputFn(), launchLogFn)
}

// validateEnvVars checks that all required environment variables are configured
func validateEnvVars(cfg jobConfig) error {
	if cfg.clusterARN == "" || cfg.taskDefinitionARN == "" || cfg.subnetIDsRaw == "" || cfg.securityGroupID == "" {
		return fmt.Errorf("missing required environment variables")
	}
	return nil
}

// launchECSTask launches an ECS task and handles the response
func launchECSTask(ctx context.Context, conn *pgx.Conn, ecsClient *ecs.Client, jobID string, runInput *ecs.RunTaskInput, launchLogFn func(string)) error {
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
		return fmt.Errorf("RunTask returned failures: %s", sanitizedErr)
	}

	if len(result.Tasks) > 0 && result.Tasks[0].TaskArn != nil {
		taskARN := *result.Tasks[0].TaskArn
		launchLogFn(taskARN)
		if err := updateJobWithTaskArn(ctx, conn, jobID, taskARN); err != nil {
			log.Printf("ERROR: failed to record task ARN for job %s: %v", jobID, err)
		}
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
