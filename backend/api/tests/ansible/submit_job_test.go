package ansible_test

import (
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/uploadutils"
)

func TestExtractOrgIDFromPlaybookKey_ValidKey(t *testing.T) {
	key := "org-123/proj-456/diag-789/playbooks/upload-abc-main.yml"
	got, err := uploadutils.ExtractOrgIDFromPlaybookKey(key)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != "org-123" {
		t.Fatalf("expected org-123, got %q", got)
	}
}

func TestExtractOrgIDFromPlaybookKey_InvalidKeys(t *testing.T) {
	cases := []string{
		"",
		"no-slash-at-all",
		"/proj-456/diag-789/playbooks/main.yml",
		"just-a-filename.yml",
	}
	for _, key := range cases {
		t.Run(key, func(t *testing.T) {
			if _, err := uploadutils.ExtractOrgIDFromPlaybookKey(key); err == nil {
				t.Fatalf("expected error for key %q", key)
			}
		})
	}
}

func TestValidatePlaybookFileName_Valid(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"deploy.yml", "deploy.yml"},
		{"Deploy.YAML", "Deploy.YAML"},
		{"my-playbook.yaml", "my-playbook.yaml"},
		{"  spaced.yml  ", "spaced.yml"},
		{strings.Repeat("a", 124) + ".yml", strings.Repeat("a", 124) + ".yml"}, // exactly 128 chars
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := uploadutils.ValidatePlaybookFileName(tc.input)
			if err != nil {
				t.Fatalf("expected no error for %q, got: %v", tc.input, err)
			}
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestValidatePlaybookFileName_Invalid(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"path traversal ..", "../etc/passwd.yml"},
		{"forward slash", "dir/file.yml"},
		{"backslash", `dir\file.yml`},
		{"wrong extension", "playbook.txt"},
		{"no extension", "playbook"},
		{"too long", strings.Repeat("a", 125) + ".yml"}, // 129 chars
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := uploadutils.ValidatePlaybookFileName(tc.input); err == nil {
				t.Fatalf("expected error for input %q", tc.input)
			}
		})
	}
}

func TestBuildPlaybookObjectKey(t *testing.T) {
	cases := []struct {
		name      string
		orgID     string
		projectID string
		diagramID string
		fileName  string
		uploadID  string
		wantKey   string
	}{
		{
			name:      "normal yaml file",
			orgID:     "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "deploy.yml", uploadID: "uuid-abc",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-abc-deploy.yml",
		},
		{
			name:      "yaml extension preserved",
			orgID:     "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "setup.yaml", uploadID: "uuid-xyz",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-xyz-setup.yaml",
		},
		{
			name:      "mixed case lowercased",
			orgID:     "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "MyPlaybook.YML", uploadID: "uuid-1",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-1-myplaybook.yml",
		},
		{
			name:      "special chars replaced with dash",
			orgID:     "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "my playbook!.yml", uploadID: "uuid-2",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-2-my-playbook.yml",
		},
		{
			name:      "all special chars falls back to playbook",
			orgID:     "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "!!!.yml", uploadID: "uuid-3",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-3-playbook.yml",
		},
		{
			name:      "repeated special chars collapsed",
			orgID:     "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "a   b.yml", uploadID: "uuid-4",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-4-a-b.yml",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := uploadutils.BuildPlaybookObjectKey(tc.orgID, tc.projectID, tc.diagramID, tc.fileName, tc.uploadID)
			if got != tc.wantKey {
				t.Fatalf("expected %q, got %q", tc.wantKey, got)
			}
		})
	}
}
