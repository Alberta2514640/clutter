package ansible_test

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/uploadutils"
)

func TestValidatePlaybookFileName_AllowsYamlExtensions(t *testing.T) {
	got, err := uploadutils.ValidatePlaybookFileName("test-main.yml")
	if err != nil {
		t.Fatalf("expected yml file name to be accepted, got error: %v", err)
	}
	if got != "test-main.yml" {
		t.Fatalf("expected normalized file name test-main.yml, got %q", got)
	}
}

func TestValidatePlaybookFileName_RejectsInvalidPathsAndExtensions(t *testing.T) {
	cases := []string{
		"",
		"../playbook.yml",
		"/tmp/playbook.yml",
		"playbooks\\test.yml",
		"test.txt",
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			if _, err := uploadutils.ValidatePlaybookFileName(input); err == nil {
				t.Fatalf("expected error for %q", input)
			}
		})
	}
}

func TestBuildPlaybookObjectKey_UsesUserScopedPrefix(t *testing.T) {
	key := uploadutils.BuildPlaybookObjectKey("org-123", "proj-456", "diag-789", "My Playbook.yml", "upload-abc")
	want := "orgs/org-123/projects/proj-456/diagrams/diag-789/playbooks/upload-abc-my-playbook.yml"
	if key != want {
		t.Fatalf("expected object key %q, got %q", want, key)
	}
}
