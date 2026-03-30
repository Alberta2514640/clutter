package uploadutils

import (
	"fmt"
	"regexp"
	"strings"
)

var invalidPlaybookChars = regexp.MustCompile(`[^a-z0-9._-]+`)
var repeatedDash = regexp.MustCompile(`-+`)
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func isValidUUID(s string) bool {
	return uuidPattern.MatchString(strings.ToLower(s))
}

func ValidatePlaybookFileName(fileName string) (string, error) {
	trimmed := strings.TrimSpace(fileName)
	if trimmed == "" {
		return "", fmt.Errorf("file_name is required")
	}
	if strings.Contains(trimmed, "..") || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", fmt.Errorf("file_name contains an invalid path")
	}

	lower := strings.ToLower(trimmed)
	if !strings.HasSuffix(lower, ".yml") && !strings.HasSuffix(lower, ".yaml") {
		return "", fmt.Errorf("file_name must end with .yml or .yaml")
	}

	if len(trimmed) > 128 {
		return "", fmt.Errorf("file_name must be 128 characters or fewer")
	}

	return trimmed, nil
}

// ExtractOrgIDFromJobPaths returns the org ID from a job's S3 key fields.
// It tries playbookS3Key first, then terraformDirectory.
func ExtractOrgIDFromJobPaths(playbookS3Key, terraformDirectory *string) (string, error) {
	var s3Path string
	if playbookS3Key != nil && *playbookS3Key != "" {
		s3Path = *playbookS3Key
	} else if terraformDirectory != nil && *terraformDirectory != "" {
		s3Path = *terraformDirectory
	}
	if s3Path == "" {
		return "", fmt.Errorf("no S3 path available")
	}
	parts := strings.SplitN(s3Path, "/", 2)
	if parts[0] == "" || !isValidUUID(parts[0]) {
		return "", fmt.Errorf("invalid org ID in S3 path")
	}
	return parts[0], nil
}

// ExtractPathComponentsFromPlaybookKey parses org, project, and diagram IDs
// from a key of the form {orgID}/{projectID}/{diagramID}/playbooks/{filename}
func ExtractPathComponentsFromPlaybookKey(key string) (orgID, projectID, diagramID string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) < 5 || !isValidUUID(parts[0]) || !isValidUUID(parts[1]) || !isValidUUID(parts[2]) {
		return "", "", "", fmt.Errorf("invalid playbook key format: expected {orgID}/{projectID}/{diagramID}/playbooks/{filename}")
	}
	return parts[0], parts[1], parts[2], nil
}

func BuildPlaybookObjectKey(orgID, projectID, diagramID, fileName, uploadID string) string {
	lower := strings.ToLower(fileName)
	ext := ".yml"
	if strings.HasSuffix(lower, ".yaml") {
		ext = ".yaml"
	}

	baseName := strings.TrimSuffix(lower, ext)
	baseName = invalidPlaybookChars.ReplaceAllString(baseName, "-")
	baseName = repeatedDash.ReplaceAllString(baseName, "-")
	baseName = strings.Trim(baseName, "-.")
	if baseName == "" {
		baseName = "playbook"
	}

	return fmt.Sprintf("%s/%s/%s/playbooks/%s-%s%s", orgID, projectID, diagramID, uploadID, baseName, ext)
}

// BuildLogObjectKey builds the S3 object key for job logs
// Format: {orgID}/{projectID}/{diagramID}/playbooks/logs/{job_id}.log
func BuildLogObjectKey(orgID, projectID, diagramID, jobID string) string {
	return fmt.Sprintf("%s/%s/%s/playbooks/logs/%s.log", orgID, projectID, diagramID, jobID)
}

// ExtractOrgIDFromLogKey parses the org ID from a log key of the form
// {orgID}/{projectID}/{diagramID}/playbooks/logs/{jobID}.log
func ExtractOrgIDFromLogKey(key string) (string, error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) < 2 || parts[0] == "" {
		return "", fmt.Errorf("invalid log key format")
	}
	return parts[0], nil
}

// ExtractPathComponentsFromLogKey parses org, project, and diagram IDs
// from a key of the form {orgID}/{projectID}/{diagramID}/playbooks/logs/{jobID}.log
func ExtractPathComponentsFromLogKey(key string) (orgID, projectID, diagramID string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) < 6 || !isValidUUID(parts[0]) || !isValidUUID(parts[1]) || !isValidUUID(parts[2]) {
		return "", "", "", fmt.Errorf("invalid log key format: expected {orgID}/{projectID}/{diagramID}/playbooks/logs/{jobID}.log")
	}
	return parts[0], parts[1], parts[2], nil
}
