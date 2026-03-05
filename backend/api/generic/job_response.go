package generic

import (
	"encoding/json"
	"time"
)

// BuildJobResponse converts database row data into a standardized job response map.
// Fields that are nil or empty are omitted from the response.
func BuildJobResponse(
	id, jobType, status string,
	createdAt, updatedAt time.Time,
	extraVars []byte,
	taskArn, errorMessage *string,

	// Ansible-specific (nullable)
	targetInstanceIDs []byte,
	playbookS3Key *string,

	// Terraform-specific (nullable)
	terraformDirectory, roleArn, assumeRoleExternalId *string,
) map[string]interface{} {
	job := map[string]interface{}{
		"id":         id,
		"job_type":   jobType,
		"status":     status,
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	if taskArn != nil {
		job["task_arn"] = *taskArn
	}
	if errorMessage != nil {
		job["error_message"] = *errorMessage
	}

	// Unmarshal extra_vars JSON
	var vars map[string]interface{}
	if len(extraVars) > 0 {
		if err := json.Unmarshal(extraVars, &vars); err == nil {
			job["extra_vars"] = vars
		}
	}

	// Ansible-specific fields
	if playbookS3Key != nil && *playbookS3Key != "" {
		job["playbook_s3_key"] = *playbookS3Key
	}
	if len(targetInstanceIDs) > 0 {
		var targets []string
		if err := json.Unmarshal(targetInstanceIDs, &targets); err == nil {
			job["target_instance_ids"] = targets
		} else {
			job["target_instance_ids"] = targetInstanceIDs
		}
	}

	// Terraform-specific fields
	if terraformDirectory != nil && *terraformDirectory != "" {
		job["terraform_directory"] = *terraformDirectory
	}
	if roleArn != nil && *roleArn != "" {
		job["role_arn"] = *roleArn
	}
	if assumeRoleExternalId != nil && *assumeRoleExternalId != "" {
		job["assume_role_external_id"] = *assumeRoleExternalId
	}

	return job
}
