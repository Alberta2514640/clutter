package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/runtaskutils"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/jackc/pgx/v5"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	cfg := runtaskutils.LoadRuntimeConfig()

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
func processJob(ctx context.Context, conn *pgx.Conn, ecsClient *ecs.Client, msg map[string]string, jobType string, cfg runtaskutils.RuntimeConfig) error {
	var jobCfg jobConfig
	var validateFn func(map[string]string) error
	var logFn func()
	var buildInputFn func() *ecs.RunTaskInput
	var launchLogFn func(string)

	jobID := msg["job_id"]

	if jobType == "terraform" {
		jobCfg = jobConfig{
			clusterARN:        cfg.Terraform.ClusterARN,
			taskDefinitionARN: cfg.Terraform.TaskDefinitionARN,
			subnetIDsRaw:      cfg.Terraform.SubnetIDsRaw,
			securityGroupID:   cfg.Terraform.SecurityGroupID,
			awsRegion:         cfg.AWSRegion,
		}
		validateFn = runtaskutils.ValidateTerraformMessage
		logFn = func() {
			log.Printf("Processing Terraform job %s: directory=%s, role=%s",
				jobID, msg["terraform_directory"], msg["role_arn"])
		}
		buildInputFn = func() *ecs.RunTaskInput {
			subnets := runtaskutils.FilterEmpty(strings.Split(jobCfg.subnetIDsRaw, ","))
			return runtaskutils.BuildTerraformRunTaskInput(jobCfg.clusterARN, jobCfg.taskDefinitionARN, subnets, jobCfg.securityGroupID, jobCfg.awsRegion, msg)
		}
		launchLogFn = func(taskARN string) {
			log.Printf("Launched Terraform Fargate task %s for job %s", taskARN, jobID)
		}
	} else {
		jobCfg = jobConfig{
			clusterARN:        cfg.Ansible.ClusterARN,
			taskDefinitionARN: cfg.Ansible.TaskDefinitionARN,
			subnetIDsRaw:      cfg.Ansible.SubnetIDsRaw,
			securityGroupID:   cfg.Ansible.SecurityGroupID,
			s3BucketName:      cfg.Ansible.S3BucketName,
		}
		validateFn = runtaskutils.ValidateAnsibleMessage
		logFn = func() {
			log.Printf("Processing Ansible job %s: playbook=%s, targets=%s",
				jobID, msg["playbook_s3_key"], msg["target_instance_ids"])
		}
		buildInputFn = func() *ecs.RunTaskInput {
			subnets := runtaskutils.FilterEmpty(strings.Split(jobCfg.subnetIDsRaw, ","))
			return runtaskutils.BuildAnsibleRunTaskInput(jobCfg.clusterARN, jobCfg.taskDefinitionARN, subnets, jobCfg.securityGroupID, jobCfg.s3BucketName, msg)
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

	subnets := runtaskutils.FilterEmpty(strings.Split(jobCfg.subnetIDsRaw, ","))
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
		sanitizedErr := runtaskutils.SanitizeError(err)
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
		sanitizedErr := runtaskutils.SanitizeError(errors.New(aggregatedFailure))
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
