package runtaskutils

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type ECSRuntimeConfig struct {
	ClusterARN        string
	TaskDefinitionARN string
	SubnetIDsRaw      string
	SecurityGroupID   string
	S3BucketName      string
}

type RuntimeConfig struct {
	Ansible   ECSRuntimeConfig
	Terraform ECSRuntimeConfig
	AWSRegion string
}

var sensitivePattern = regexp.MustCompile(`\b\d{12}\b|arn:aws:[^\s,]+|i-[0-9a-f]+|sg-[0-9a-f]+|subnet-[0-9a-f]+|vpc-[0-9a-f]+`)
var sensitiveKeywords = regexp.MustCompile(`(?i)\b(password|postgresql://|postgres://|token|secret|AKIA)[^\s,]*`)

const maxErrorMessageLength = 500

func SanitizeError(err error) string {
	if err == nil {
		return "unknown error"
	}
	msg := err.Error()

	msg = sensitivePattern.ReplaceAllString(msg, "***REDACTED***")
	msg = sensitiveKeywords.ReplaceAllString(msg, "***REDACTED***")

	if len(msg) > maxErrorMessageLength {
		msg = msg[:maxErrorMessageLength] + "...(truncated)"
	}

	return msg
}

func LoadRuntimeConfig() RuntimeConfig {
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-west-2"
	}

	return RuntimeConfig{
		Ansible: ECSRuntimeConfig{
			ClusterARN:        os.Getenv("ANSIBLE_ECS_CLUSTER_ARN"),
			TaskDefinitionARN: os.Getenv("ANSIBLE_TASK_DEFINITION_ARN"),
			SubnetIDsRaw:      os.Getenv("ANSIBLE_SUBNET_IDS"),
			SecurityGroupID:   os.Getenv("ANSIBLE_SECURITY_GROUP_ID"),
			S3BucketName:      os.Getenv("S3_BUCKET_NAME"),
		},
		Terraform: ECSRuntimeConfig{
			ClusterARN:        os.Getenv("TERRAFORM_ECS_CLUSTER_ARN"),
			TaskDefinitionARN: os.Getenv("TERRAFORM_TASK_DEFINITION_ARN"),
			SubnetIDsRaw:      os.Getenv("TERRAFORM_SUBNET_IDS"),
			SecurityGroupID:   os.Getenv("TERRAFORM_SECURITY_GROUP_ID"),
		},
		AWSRegion: awsRegion,
	}
}

func ValidateAnsibleMessage(msg map[string]string) error {
	if msg["playbook_s3_key"] == "" || msg["target_instance_ids"] == "" || msg["role_arn"] == "" || msg["assume_role_external_id"] == "" {
		return fmt.Errorf("missing required Ansible fields: playbook_s3_key, target_instance_ids, role_arn, or assume_role_external_id")
	}
	if strings.Contains(msg["playbook_s3_key"], "..") ||
		strings.Contains(msg["playbook_s3_key"], "~") ||
		strings.HasPrefix(msg["playbook_s3_key"], "/") ||
		strings.HasPrefix(msg["playbook_s3_key"], "\\") {
		return fmt.Errorf("invalid playbook_s3_key: path traversal not allowed")
	}
	var instanceIDs []string
	if err := json.Unmarshal([]byte(msg["target_instance_ids"]), &instanceIDs); err != nil {
		return fmt.Errorf("target_instance_ids is not valid JSON array: %w", err)
	}
	if len(instanceIDs) == 0 {
		return fmt.Errorf("target_instance_ids must not be empty")
	}
	return nil
}

func ValidateTerraformMessage(msg map[string]string) error {
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

func FilterEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func BuildAnsibleRunTaskInput(clusterARN, taskDefARN string, subnets []string, securityGroupID, s3BucketName string, msg map[string]string) *ecs.RunTaskInput {
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
						{Name: aws.String("ORG_ID"), Value: aws.String(msg["org_id"])},
						{Name: aws.String("PROJECT_ID"), Value: aws.String(msg["project_id"])},
						{Name: aws.String("DIAGRAM_ID"), Value: aws.String(msg["diagram_id"])},
						{Name: aws.String("CLIENT_ROLE_ARN"), Value: aws.String(msg["role_arn"])},
						{Name: aws.String("ASSUME_ROLE_EXTERNAL_ID"), Value: aws.String(msg["assume_role_external_id"])},
					},
				},
			},
		},
	}
}

func BuildTerraformRunTaskInput(clusterARN, taskDefARN string, subnets []string, securityGroupID, awsRegion string, msg map[string]string) *ecs.RunTaskInput {
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
