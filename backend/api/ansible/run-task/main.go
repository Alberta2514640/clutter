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
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {

	clusterARN := os.Getenv("ECS_CLUSTER_ARN")
	taskDefARN := os.Getenv("TASK_DEFINITION_ARN")
	subnetIDsRaw := os.Getenv("SUBNET_IDS")
	securityGroupID := os.Getenv("SECURITY_GROUP_ID")
	connString := os.Getenv("PSQL_CONNECTION_STRING") // Also passed to Fargate container
	s3BucketName := os.Getenv("S3_BUCKET_NAME")

	// Validate all required environment variables
	var missingVars []string
	if clusterARN == "" {
		missingVars = append(missingVars, "ECS_CLUSTER_ARN")
	}
	if taskDefARN == "" {
		missingVars = append(missingVars, "TASK_DEFINITION_ARN")
	}
	if subnetIDsRaw == "" {
		missingVars = append(missingVars, "SUBNET_IDS")
	}
	if securityGroupID == "" {
		missingVars = append(missingVars, "SECURITY_GROUP_ID")
	}
	if connString == "" {
		missingVars = append(missingVars, "PSQL_CONNECTION_STRING")
	}
	if s3BucketName == "" {
		missingVars = append(missingVars, "S3_BUCKET_NAME")
	}
	if len(missingVars) > 0 {
		return events.SQSEventResponse{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missingVars, ", "))
	}

	// Parse SUBNET_IDS safely — an empty string would produce [""] from Split
	subnetIDs := strings.Split(subnetIDsRaw, ",")

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
		playbookS3Key := msg["playbook_s3_key"]
		rawTargetInstanceIDs := msg["target_instance_ids"]
		extraVars := msg["extra_vars"]

		// Validate required message fields
		var missingFields []string
		if jobID == "" {
			missingFields = append(missingFields, "job_id")
		}
		if playbookS3Key == "" {
			missingFields = append(missingFields, "playbook_s3_key")
		}
		if rawTargetInstanceIDs == "" {
			missingFields = append(missingFields, "target_instance_ids")
		}
		if len(missingFields) > 0 {
			log.Printf("ERROR: SQS message missing required fields: %s", strings.Join(missingFields, ", "))
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue
		}

		// Convert target_instance_ids from JSON array to comma-separated string
		// submit-job sends it as '["i-abc","i-def"]' but entrypoint.sh expects 'i-abc,i-def'
		var targetInstanceIDsList []string
		var targetInstanceIDs string
		if err := json.Unmarshal([]byte(rawTargetInstanceIDs), &targetInstanceIDsList); err == nil {
			targetInstanceIDs = strings.Join(targetInstanceIDsList, ",")
		} else {
			// Fallback: assume it's already comma-separated
			targetInstanceIDs = rawTargetInstanceIDs
		}

		log.Printf("Processing job %s: playbook=%s, targets=%s", jobID, playbookS3Key, targetInstanceIDs)

		// Update job status to STARTING
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = conn.Exec(ctx, `
			UPDATE jobs
			SET status = $1, updated_at = $2
			WHERE id = $3
		`, "STARTING", now, jobID)

		if err != nil {
			log.Printf("ERROR: failed to update job status for %s: %v", jobID, err)
			// Proceeding anyway as we want to try to run the task? Or fail?
			// If we can't update DB, we might want to retry.
			// For now, let's log and proceed, but maybe we should fail the batch item?
		}

		// Launch Fargate task with environment variable overrides
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
							{Name: aws.String("PSQL_CONNECTION_STRING"), Value: aws.String(connString)}, // Pass DB conn string to task if needed
						},
					},
				},
			},
		}

		result, err := ecsClient.RunTask(ctx, runInput)
		if err != nil {
			log.Printf("ERROR: failed to launch Fargate task for job %s: %v", jobID, err)
			// Update job status to FAILED with fresh timestamp
			failNow := time.Now().UTC().Format(time.RFC3339)
			_, _ = conn.Exec(ctx, `
				UPDATE jobs
				SET status = $1, updated_at = $2, error_message = $3
				WHERE id = $4
			`, "FAILED", failNow, err.Error(), jobID)

			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue
		}

		// Check for RunTask failures (e.g., placement/capacity issues)
		if len(result.Failures) > 0 {
			var failureMsgs []string
			for _, f := range result.Failures {
				failureMsg := fmt.Sprintf("arn=%s reason=%s detail=%s",
					aws.ToString(f.Arn), aws.ToString(f.Reason), aws.ToString(f.Detail))
				failureMsgs = append(failureMsgs, failureMsg)
			}
			aggregatedFailure := strings.Join(failureMsgs, "; ")
			log.Printf("ERROR: RunTask returned failures for job %s: %s", jobID, aggregatedFailure)

			// Update job status to FAILED with failure details and fresh timestamp
			failNow := time.Now().UTC().Format(time.RFC3339)

			_, _ = conn.Exec(ctx, `
				UPDATE jobs
				SET status = $1, updated_at = $2, error_message = $3
				WHERE id = $4
			`, "FAILED", failNow, aggregatedFailure, jobID)

			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue
		}

		if len(result.Tasks) > 0 {
			taskARN := *result.Tasks[0].TaskArn
			log.Printf("Launched Fargate task %s for job %s", taskARN, jobID)

			// Update job with task ARN and fresh timestamp
			successNow := time.Now().UTC().Format(time.RFC3339)
			_, _ = conn.Exec(ctx, `
				UPDATE jobs
				SET task_arn = $1, updated_at = $2
				WHERE id = $3
			`, taskARN, successNow, jobID)
		}
	}

	return events.SQSEventResponse{BatchItemFailures: batchItemFailures}, nil
}
