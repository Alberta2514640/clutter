package uploadutils

import (
	"fmt"
	"regexp"
	"strings"
)

var invalidPlaybookChars = regexp.MustCompile(`[^a-z0-9._-]+`)
var repeatedDash = regexp.MustCompile(`-+`)

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

	return fmt.Sprintf("orgs/%s/projects/%s/diagrams/%s/playbooks/%s-%s%s", orgID, projectID, diagramID, uploadID, baseName, ext)
}
