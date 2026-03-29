package ansible_test

import "github.com/Alberta2514640/clutter/backend/api/ansible/shared/runtaskutils"

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// ── runtaskutils.LoadRuntimeConfig ─────────────────────────────────────────────────────────

func TestLoadRuntimeConfig_UsesAnsiblePrefixedEnvironmentVariables(t *testing.T) {
	t.Setenv("ANSIBLE_ECS_CLUSTER_ARN", "ansible-cluster")
	t.Setenv("ANSIBLE_TASK_DEFINITION_ARN", "ansible-task")
	t.Setenv("ANSIBLE_SUBNET_IDS", "subnet-1,subnet-2")
	t.Setenv("ANSIBLE_SECURITY_GROUP_ID", "sg-1")
	t.Setenv("S3_BUCKET_NAME", "clutter-bucket")
	t.Setenv("AWS_REGION", "ca-central-1")

	cfg := runtaskutils.LoadRuntimeConfig()

	if cfg.AWSRegion != "ca-central-1" {
		t.Fatalf("expected awsRegion ca-central-1, got %q", cfg.AWSRegion)
	}
	if cfg.Ansible.ClusterARN != "ansible-cluster" {
		t.Fatalf("expected ansible cluster ARN from prefixed env var, got %q", cfg.Ansible.ClusterARN)
	}
	if cfg.Ansible.TaskDefinitionARN != "ansible-task" {
		t.Fatalf("expected ansible task definition ARN from prefixed env var, got %q", cfg.Ansible.TaskDefinitionARN)
	}
	if cfg.Ansible.SubnetIDsRaw != "subnet-1,subnet-2" {
		t.Fatalf("expected ansible subnet IDs from prefixed env var, got %q", cfg.Ansible.SubnetIDsRaw)
	}
	if cfg.Ansible.SecurityGroupID != "sg-1" {
		t.Fatalf("expected ansible security group ID from prefixed env var, got %q", cfg.Ansible.SecurityGroupID)
	}
	if cfg.Ansible.S3BucketName != "clutter-bucket" {
		t.Fatalf("expected ansible S3 bucket name, got %q", cfg.Ansible.S3BucketName)
	}
}

func TestLoadRuntimeConfig_DefaultsRegionWhenUnset(t *testing.T) {
	t.Setenv("AWS_REGION", "")

	cfg := runtaskutils.LoadRuntimeConfig()

	if cfg.AWSRegion != "us-west-2" {
		t.Fatalf("expected default region us-west-2, got %q", cfg.AWSRegion)
	}
}

// ── runtaskutils.SanitizeError ─────────────────────────────────────────────────────────────

func TestSanitizeError_ReturnsUnknownErrorForNil(t *testing.T) {
	got := runtaskutils.SanitizeError(nil)
	if got != "unknown error" {
		t.Fatalf("expected %q, got %q", "unknown error", got)
	}
}

func TestSanitizeError_RedactsSensitiveKeywords(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		pattern string // the sensitive keyword that must be absent after sanitization
	}{
		{"password keyword", "password=hunter2", "password"},
		{"postgresql scheme", "postgresql://user:pass@host/db", "postgresql://"},
		{"postgres scheme", "postgres://user:pass@host/db", "postgres://"},
		{"token keyword", "token=abc123", "token"},
		{"secret keyword", "secret=mysecret", "secret"},
		{"AWS key prefix", "AKIA1234EXAMPLE", "AKIA"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runtaskutils.SanitizeError(fmt.Errorf("%s", tc.input))
			if strings.Contains(got, tc.pattern) {
				t.Errorf("sensitive pattern %q still present in output %q", tc.pattern, got)
			}
			if !strings.Contains(got, "***REDACTED***") {
				t.Errorf("expected redaction marker in output %q", got)
			}
		})
	}
}

func TestSanitizeError_TruncatesLongMessages(t *testing.T) {
	long := strings.Repeat("a", 600)
	got := runtaskutils.SanitizeError(fmt.Errorf("%s", long))
	if !strings.Contains(got, "...(truncated)") {
		t.Fatalf("expected truncation marker in output %q", got)
	}
	if len(got) > 520 {
		t.Fatalf("expected truncated message length <= 520, got %d", len(got))
	}
}

func TestSanitizeError_PreservesNonSensitiveMessages(t *testing.T) {
	input := "connection refused: dial tcp 127.0.0.1:5432"
	got := runtaskutils.SanitizeError(fmt.Errorf("%s", input))
	if got != input {
		t.Fatalf("expected non-sensitive message unchanged, got %q", got)
	}
}

// ── runtaskutils.ValidateAnsibleMessage ────────────────────────────────────────────────────

func TestValidateAnsibleMessage_AcceptsValidMessage(t *testing.T) {
	msg := map[string]string{
		"job_id":              "job-123",
		"playbook_s3_key":     "org-1/proj-1/diag-1/playbooks/upload-abc-site.yml",
		"target_instance_ids": `["i-0abc123def456"]`,
		"role_arn":            "arn",
		"assume_role_external_id": "ext",
	}
	if err := runtaskutils.ValidateAnsibleMessage(msg); err != nil {
		t.Fatalf("expected no error for valid message, got: %v", err)
	}
}

func TestValidateAnsibleMessage_RejectsMissingRequiredFields(t *testing.T) {
	base := map[string]string{
		"playbook_s3_key":         "org-1/proj-1/diag-1/playbooks/upload-abc-site.yml",
		"target_instance_ids":     `["i-0abc123def456"]`,
		"role_arn":                "arn:aws:iam::123:role/AnsibleClientRole",
		"assume_role_external_id": "ext-id",
	}

	for _, field := range []string{"playbook_s3_key", "target_instance_ids", "role_arn", "assume_role_external_id"} {
		t.Run("missing "+field, func(t *testing.T) {
			msg := map[string]string{
				"job_id":                  "job-123",
				"playbook_s3_key":         base["playbook_s3_key"],
				"target_instance_ids":     base["target_instance_ids"],
				"role_arn":                base["role_arn"],
				"assume_role_external_id": base["assume_role_external_id"],
			}
			delete(msg, field)
			if err := runtaskutils.ValidateAnsibleMessage(msg); err == nil {
				t.Errorf("expected error when %q is absent, got nil", field)
			}
		})
	}
}

func TestValidateAnsibleMessage_RejectsPathTraversalInPlaybookKey(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{"double dot", "playbooks/../etc/shadow"},
		{"tilde", "~/playbooks/site.yml"},
		{"leading slash", "/tmp/playbook.yml"},
		{"leading backslash", `\tmp\playbook.yml`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := map[string]string{
				"job_id":              "job-123",
				"playbook_s3_key":     tc.key,
				"target_instance_ids": `["i-0abc"]`,
				"role_arn":            "arn",
				"assume_role_external_id": "ext",
			}
			if err := runtaskutils.ValidateAnsibleMessage(msg); err == nil {
				t.Errorf("expected path traversal error for key %q, got nil", tc.key)
			}
		})
	}
}

// ── runtaskutils.ValidateTerraformMessage ──────────────────────────────────────────────────

func TestValidateTerraformMessage_AcceptsValidMessage(t *testing.T) {
	msg := map[string]string{
		"job_id":                  "job-tf-1",
		"terraform_directory":     "infra/prod",
		"role_arn":                "arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-abc",
		"assume_role_external_id": "ext-id-abc",
	}
	if err := runtaskutils.ValidateTerraformMessage(msg); err != nil {
		t.Fatalf("expected no error for valid Terraform message, got: %v", err)
	}
}

func TestValidateTerraformMessage_RejectsMissingRequiredFields(t *testing.T) {
	base := map[string]string{
		"terraform_directory":     "infra/prod",
		"role_arn":                "arn:aws:iam::123456789012:role/SomeRole",
		"assume_role_external_id": "ext-id",
	}

	for _, field := range []string{"terraform_directory", "role_arn", "assume_role_external_id"} {
		t.Run("missing "+field, func(t *testing.T) {
			msg := map[string]string{
				"terraform_directory":     base["terraform_directory"],
				"role_arn":                base["role_arn"],
				"assume_role_external_id": base["assume_role_external_id"],
			}
			delete(msg, field)
			if err := runtaskutils.ValidateTerraformMessage(msg); err == nil {
				t.Errorf("expected error when %q is absent, got nil", field)
			}
		})
	}
}

func TestValidateTerraformMessage_RejectsPathTraversalInDirectory(t *testing.T) {
	cases := []struct {
		name string
		dir  string
	}{
		{"double dot", "../../../etc"},
		{"tilde", "~/infra"},
		{"leading slash", "/tmp/infra"},
		{"leading backslash", `\infra\prod`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := map[string]string{
				"terraform_directory":     tc.dir,
				"role_arn":                "arn:aws:iam::123456789012:role/SomeRole",
				"assume_role_external_id": "ext-id",
			}
			if err := runtaskutils.ValidateTerraformMessage(msg); err == nil {
				t.Errorf("expected path traversal error for directory %q, got nil", tc.dir)
			}
		})
	}
}

// ── runtaskutils.BuildAnsibleRunTaskInput ──────────────────────────────────────────────────

func TestBuildAnsibleRunTaskInput_SetsClusterAndTaskDefinition(t *testing.T) {
	msg := map[string]string{
		"job_id":              "job-abc",
		"playbook_s3_key":     "org-1/proj-1/diag-1/playbooks/upload-abc-site.yml",
		"target_instance_ids": "i-0abc123",
		"extra_vars":          "",
		"role_arn":            "arn",
		"assume_role_external_id": "ext",
	}

	input := runtaskutils.BuildAnsibleRunTaskInput(
		"arn:aws:ecs:us-west-2:123:cluster/clutter",
		"arn:aws:ecs:us-west-2:123:task-definition/ansible-runner:1",
		[]string{"subnet-aaa", "subnet-bbb"},
		"sg-111",
		"clutter-bucket",
		msg,
	)

	if aws.ToString(input.Cluster) != "arn:aws:ecs:us-west-2:123:cluster/clutter" {
		t.Errorf("unexpected cluster ARN: %s", aws.ToString(input.Cluster))
	}
	if aws.ToString(input.TaskDefinition) != "arn:aws:ecs:us-west-2:123:task-definition/ansible-runner:1" {
		t.Errorf("unexpected task definition ARN: %s", aws.ToString(input.TaskDefinition))
	}
}

func TestBuildAnsibleRunTaskInput_UsesLaunchTypeFargate(t *testing.T) {
	msg := map[string]string{
		"job_id":              "job-abc",
		"playbook_s3_key":     "org-1/proj-1/diag-1/playbooks/site.yml",
		"target_instance_ids": "i-0abc123",
		"role_arn":            "arn",
		"assume_role_external_id": "ext",
	}

	input := runtaskutils.BuildAnsibleRunTaskInput("cluster", "taskdef", []string{"subnet-a"}, "sg-1", "bucket", msg)

	if input.LaunchType != ecstypes.LaunchTypeFargate {
		t.Errorf("expected LaunchTypeFargate, got %v", input.LaunchType)
	}
	if aws.ToInt32(input.Count) != 1 {
		t.Errorf("expected task count 1, got %d", aws.ToInt32(input.Count))
	}
}

func TestBuildAnsibleRunTaskInput_ParsesSubnetIDs(t *testing.T) {
	msg := map[string]string{
		"job_id":              "job-abc",
		"playbook_s3_key":     "org-1/proj-1/diag-1/playbooks/site.yml",
		"target_instance_ids": "i-0abc123",
		"role_arn":            "arn",
		"assume_role_external_id": "ext",
	}

	input := runtaskutils.BuildAnsibleRunTaskInput("cluster", "taskdef", []string{"subnet-aaa", "subnet-bbb", "subnet-ccc"}, "sg-1", "bucket", msg)

	subnets := input.NetworkConfiguration.AwsvpcConfiguration.Subnets
	if len(subnets) != 3 {
		t.Fatalf("expected 3 subnets, got %d: %v", len(subnets), subnets)
	}
	if subnets[0] != "subnet-aaa" || subnets[1] != "subnet-bbb" || subnets[2] != "subnet-ccc" {
		t.Errorf("unexpected subnet order: %v", subnets)
	}
}

func TestBuildAnsibleRunTaskInput_AssignsPublicIP(t *testing.T) {
	msg := map[string]string{
		"job_id":              "job-abc",
		"playbook_s3_key":     "org-1/proj-1/diag-1/playbooks/site.yml",
		"target_instance_ids": "i-0abc123",
		"role_arn":            "arn",
		"assume_role_external_id": "ext",
	}

	input := runtaskutils.BuildAnsibleRunTaskInput("cluster", "taskdef", []string{"subnet-a"}, "sg-1", "bucket", msg)

	got := input.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp
	if got != ecstypes.AssignPublicIpEnabled {
		t.Errorf("expected AssignPublicIpEnabled, got %v", got)
	}
}

func TestBuildAnsibleRunTaskInput_SetsContainerName(t *testing.T) {
	msg := map[string]string{
		"job_id":              "job-abc",
		"playbook_s3_key":     "org-1/proj-1/diag-1/playbooks/site.yml",
		"target_instance_ids": "i-0abc123",
		"role_arn":            "arn",
		"assume_role_external_id": "ext",
	}

	input := runtaskutils.BuildAnsibleRunTaskInput("cluster", "taskdef", []string{"subnet-a"}, "sg-1", "bucket", msg)

	overrides := input.Overrides.ContainerOverrides
	if len(overrides) != 1 {
		t.Fatalf("expected 1 container override, got %d", len(overrides))
	}
	if aws.ToString(overrides[0].Name) != "ansible-executor" {
		t.Errorf("expected container name ansible-executor, got %q", aws.ToString(overrides[0].Name))
	}
}

func TestBuildAnsibleRunTaskInput_PassesJobEnvVarsToContainer(t *testing.T) {
	msg := map[string]string{
		"job_id":              "job-xyz",
		"playbook_s3_key":     "org-1/proj-1/diag-1/playbooks/upload-abc-site.yml",
		"target_instance_ids": "i-0abc123",
		"extra_vars":          `{"env":"prod"}`,
		"org_id":              "org-1",
		"project_id":          "proj-1",
		"diagram_id":          "diag-1",
		"role_arn":            "arn",
		"assume_role_external_id": "ext",
	}

	input := runtaskutils.BuildAnsibleRunTaskInput(
		"cluster", "taskdef", []string{"subnet-a"}, "sg-1", "my-bucket", msg,
	)

	envMap := make(map[string]string)
	for _, kv := range input.Overrides.ContainerOverrides[0].Environment {
		envMap[aws.ToString(kv.Name)] = aws.ToString(kv.Value)
	}

	if envMap["JOB_ID"] != "job-xyz" {
		t.Errorf("JOB_ID: got %q", envMap["JOB_ID"])
	}
	if envMap["PLAYBOOK_S3_KEY"] != "org-1/proj-1/diag-1/playbooks/upload-abc-site.yml" {
		t.Errorf("PLAYBOOK_S3_KEY: got %q", envMap["PLAYBOOK_S3_KEY"])
	}
	if envMap["EXTRA_VARS"] != `{"env":"prod"}` {
		t.Errorf("EXTRA_VARS: got %q", envMap["EXTRA_VARS"])
	}
	if envMap["S3_BUCKET_NAME"] != "my-bucket" {
		t.Errorf("S3_BUCKET_NAME: got %q", envMap["S3_BUCKET_NAME"])
	}
	if envMap["ORG_ID"] != "org-1" {
		t.Errorf("ORG_ID: got %q", envMap["ORG_ID"])
	}
	if envMap["PROJECT_ID"] != "proj-1" {
		t.Errorf("PROJECT_ID: got %q", envMap["PROJECT_ID"])
	}
	if envMap["DIAGRAM_ID"] != "diag-1" {
		t.Errorf("DIAGRAM_ID: got %q", envMap["DIAGRAM_ID"])
	}
}

func TestBuildAnsibleRunTaskInput_NormalisesJSONArrayTargetIDs(t *testing.T) {
	msg := map[string]string{
		"job_id":              "job-abc",
		"playbook_s3_key":     "org-1/proj-1/diag-1/playbooks/site.yml",
		"target_instance_ids": `["i-0aaa","i-0bbb","i-0ccc"]`,
		"role_arn":            "arn",
		"assume_role_external_id": "ext",
	}

	input := runtaskutils.BuildAnsibleRunTaskInput("cluster", "taskdef", []string{"subnet-a"}, "sg-1", "bucket", msg)

	envMap := make(map[string]string)
	for _, kv := range input.Overrides.ContainerOverrides[0].Environment {
		envMap[aws.ToString(kv.Name)] = aws.ToString(kv.Value)
	}

	want := "i-0aaa,i-0bbb,i-0ccc"
	if envMap["TARGET_INSTANCE_IDS"] != want {
		t.Errorf("expected TARGET_INSTANCE_IDS %q, got %q", want, envMap["TARGET_INSTANCE_IDS"])
	}
}

func TestBuildAnsibleRunTaskInput_PassesThroughRawTargetIDs(t *testing.T) {
	msg := map[string]string{
		"job_id":              "job-abc",
		"playbook_s3_key":     "org-1/proj-1/diag-1/playbooks/site.yml",
		"target_instance_ids": "i-0aaa,i-0bbb",
		"role_arn":            "arn",
		"assume_role_external_id": "ext",
	}

	input := runtaskutils.BuildAnsibleRunTaskInput("cluster", "taskdef", []string{"subnet-a"}, "sg-1", "bucket", msg)

	envMap := make(map[string]string)
	for _, kv := range input.Overrides.ContainerOverrides[0].Environment {
		envMap[aws.ToString(kv.Name)] = aws.ToString(kv.Value)
	}

	if envMap["TARGET_INSTANCE_IDS"] != "i-0aaa,i-0bbb" {
		t.Errorf("expected raw target IDs unchanged, got %q", envMap["TARGET_INSTANCE_IDS"])
	}
}

// ── runtaskutils.BuildTerraformRunTaskInput ────────────────────────────────────────────────

func TestBuildTerraformRunTaskInput_SetsClusterAndTaskDefinition(t *testing.T) {
	msg := map[string]string{
		"job_id":                  "job-tf-1",
		"terraform_directory":     "infra/prod",
		"role_arn":                "arn:aws:iam::123:role/SomeRole",
		"assume_role_external_id": "ext-id",
	}

	input := runtaskutils.BuildTerraformRunTaskInput(
		"arn:aws:ecs:us-west-2:123:cluster/clutter",
		"arn:aws:ecs:us-west-2:123:task-definition/terraform-deployer:2",
		[]string{"subnet-aaa"},
		"sg-222",
		"us-west-2",
		msg,
	)

	if aws.ToString(input.Cluster) != "arn:aws:ecs:us-west-2:123:cluster/clutter" {
		t.Errorf("unexpected cluster ARN: %s", aws.ToString(input.Cluster))
	}
	if aws.ToString(input.TaskDefinition) != "arn:aws:ecs:us-west-2:123:task-definition/terraform-deployer:2" {
		t.Errorf("unexpected task definition ARN: %s", aws.ToString(input.TaskDefinition))
	}
}

func TestBuildTerraformRunTaskInput_SetsContainerName(t *testing.T) {
	msg := map[string]string{
		"job_id":                  "job-tf-1",
		"terraform_directory":     "infra/prod",
		"role_arn":                "arn:aws:iam::123:role/SomeRole",
		"assume_role_external_id": "ext-id",
	}

	input := runtaskutils.BuildTerraformRunTaskInput("cluster", "taskdef", []string{"subnet-a"}, "sg-1", "us-west-2", msg)

	overrides := input.Overrides.ContainerOverrides
	if len(overrides) != 1 {
		t.Fatalf("expected 1 container override, got %d", len(overrides))
	}
	if aws.ToString(overrides[0].Name) != "terraform-deployer" {
		t.Errorf("expected container name terraform-deployer, got %q", aws.ToString(overrides[0].Name))
	}
}

func TestBuildTerraformRunTaskInput_PassesJobEnvVarsToContainer(t *testing.T) {
	msg := map[string]string{
		"job_id":                  "job-tf-99",
		"terraform_directory":     "infra/staging",
		"role_arn":                "arn:aws:iam::555:role/DeployRole",
		"assume_role_external_id": "ext-staging",
		"extra_vars":              `{"region":"us-east-1"}`,
	}

	input := runtaskutils.BuildTerraformRunTaskInput("cluster", "taskdef", []string{"subnet-a"}, "sg-1", "ca-central-1", msg)

	envMap := make(map[string]string)
	for _, kv := range input.Overrides.ContainerOverrides[0].Environment {
		envMap[aws.ToString(kv.Name)] = aws.ToString(kv.Value)
	}

	if envMap["JOB_ID"] != "job-tf-99" {
		t.Errorf("JOB_ID: got %q", envMap["JOB_ID"])
	}
	if envMap["TERRAFORM_DIRECTORY"] != "infra/staging" {
		t.Errorf("TERRAFORM_DIRECTORY: got %q", envMap["TERRAFORM_DIRECTORY"])
	}
	if envMap["CLIENT_ROLE_ARN"] != "arn:aws:iam::555:role/DeployRole" {
		t.Errorf("CLIENT_ROLE_ARN: got %q", envMap["CLIENT_ROLE_ARN"])
	}
	if envMap["ASSUME_ROLE_EXTERNAL_ID"] != "ext-staging" {
		t.Errorf("ASSUME_ROLE_EXTERNAL_ID: got %q", envMap["ASSUME_ROLE_EXTERNAL_ID"])
	}
	if envMap["AWS_REGION"] != "ca-central-1" {
		t.Errorf("AWS_REGION: got %q", envMap["AWS_REGION"])
	}
	if envMap["EXTRA_VARS"] != `{"region":"us-east-1"}` {
		t.Errorf("EXTRA_VARS: got %q", envMap["EXTRA_VARS"])
	}
}

func TestBuildTerraformRunTaskInput_ParsesSubnetIDs(t *testing.T) {
	msg := map[string]string{
		"job_id":                  "job-tf-1",
		"terraform_directory":     "infra/prod",
		"role_arn":                "arn:aws:iam::123:role/SomeRole",
		"assume_role_external_id": "ext-id",
	}

	input := runtaskutils.BuildTerraformRunTaskInput("cluster", "taskdef", []string{"subnet-x", "subnet-y"}, "sg-1", "us-west-2", msg)

	subnets := input.NetworkConfiguration.AwsvpcConfiguration.Subnets
	if len(subnets) != 2 {
		t.Fatalf("expected 2 subnets, got %d: %v", len(subnets), subnets)
	}
	if subnets[0] != "subnet-x" || subnets[1] != "subnet-y" {
		t.Errorf("unexpected subnets: %v", subnets)
	}
}
