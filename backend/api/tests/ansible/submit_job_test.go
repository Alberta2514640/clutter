package ansible_test

import (
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/uploadutils"
)

func TestExtractPathComponentsFromPlaybookKey_ValidKey(t *testing.T) {
	key := "11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222/33333333-3333-3333-3333-333333333333/playbooks/upload-abc-main.yml"
	orgID, projectID, diagramID, err := uploadutils.ExtractPathComponentsFromPlaybookKey(key)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if orgID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected orgID 11111111-1111-1111-1111-111111111111, got %q", orgID)
	}
	if projectID != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("expected projectID 22222222-2222-2222-2222-222222222222, got %q", projectID)
	}
	if diagramID != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("expected diagramID 33333333-3333-3333-3333-333333333333, got %q", diagramID)
	}
}

func TestExtractPathComponentsFromPlaybookKey_InvalidKeys(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"no slashes", "no-slash-at-all"},
		{"only two parts", "11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222"},
		{"empty org", "/22222222-2222-2222-2222-222222222222/33333333-3333-3333-3333-333333333333/playbooks/main.yml"},
		{"empty project", "11111111-1111-1111-1111-111111111111//33333333-3333-3333-3333-333333333333/playbooks/main.yml"},
		{"empty diagram", "11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222//playbooks/main.yml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := uploadutils.ExtractPathComponentsFromPlaybookKey(tc.key); err == nil {
				t.Fatalf("expected error for key %q", tc.key)
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
			name:  "normal yaml file",
			orgID: "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "deploy.yml", uploadID: "uuid-abc",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-abc-deploy.yml",
		},
		{
			name:  "yaml extension preserved",
			orgID: "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "setup.yaml", uploadID: "uuid-xyz",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-xyz-setup.yaml",
		},
		{
			name:  "mixed case lowercased",
			orgID: "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "MyPlaybook.YML", uploadID: "uuid-1",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-1-myplaybook.yml",
		},
		{
			name:  "special chars replaced with dash",
			orgID: "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "my playbook!.yml", uploadID: "uuid-2",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-2-my-playbook.yml",
		},
		{
			name:  "all special chars falls back to playbook",
			orgID: "org-1", projectID: "proj-2", diagramID: "diag-3",
			fileName: "!!!.yml", uploadID: "uuid-3",
			wantKey: "org-1/proj-2/diag-3/playbooks/uuid-3-playbook.yml",
		},
		{
			name:  "repeated special chars collapsed",
			orgID: "org-1", projectID: "proj-2", diagramID: "diag-3",
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

func TestValidatePlaybookHostsConstraint_Valid(t *testing.T) {
	playbook := `---
- name: Configure localhost
  hosts: localhost
  tasks:
    - name: Debug
      ansible.builtin.debug:
        msg: hello

- hosts: "all"
  tasks:
    - name: Ping
      ansible.builtin.ping:
`

	if err := uploadutils.ValidatePlaybookHostsConstraint(playbook); err != nil {
		t.Fatalf("expected valid playbook hosts, got: %v", err)
	}
}

func TestValidatePlaybookHostsConstraint_RejectsUnsupportedHosts(t *testing.T) {
	playbook := `- name: Configure web fleet
  hosts: web
  tasks:
    - name: Ping
      ansible.builtin.ping:
`

	err := uploadutils.ValidatePlaybookHostsConstraint(playbook)
	if err == nil {
		t.Fatal("expected invalid hosts value to be rejected")
	}
	if !strings.Contains(err.Error(), `unsupported hosts value "web"`) {
		t.Fatalf("expected unsupported hosts error, got: %v", err)
	}
}

func TestValidatePlaybookHostsConstraint_RejectsMissingHosts(t *testing.T) {
	playbook := `- name: Missing hosts
  tasks:
    - name: Ping
      ansible.builtin.ping:
`

	err := uploadutils.ValidatePlaybookHostsConstraint(playbook)
	if err == nil {
		t.Fatal("expected missing hosts to be rejected")
	}
	if !strings.Contains(err.Error(), "must declare hosts") {
		t.Fatalf("expected missing hosts error, got: %v", err)
	}
}

func TestValidatePlaybookHostsConstraint_IgnoresNestedHostsKeys(t *testing.T) {
	playbook := `- name: Validate nested data
  hosts: all
  vars:
    hosts: web
  tasks:
    - name: Show value
      ansible.builtin.debug:
        var: hosts
`

	if err := uploadutils.ValidatePlaybookHostsConstraint(playbook); err != nil {
		t.Fatalf("expected nested vars.hosts to be ignored, got: %v", err)
	}
}
