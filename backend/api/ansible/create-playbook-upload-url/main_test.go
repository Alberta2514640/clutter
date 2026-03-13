package main

import "testing"

func TestValidatePlaybookFileName_AllowsYamlExtensions(t *testing.T) {
	got, err := validatePlaybookFileName("test-main.yml")
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
			if _, err := validatePlaybookFileName(input); err == nil {
				t.Fatalf("expected error for %q", input)
			}
		})
	}
}

func TestBuildPlaybookObjectKey_UsesUserScopedPrefix(t *testing.T) {
	key := buildPlaybookObjectKey("user-123", "My Playbook.yml", "upload-456")
	want := "playbooks/user-123/upload-456-my-playbook.yml"
	if key != want {
		t.Fatalf("expected object key %q, got %q", want, key)
	}
}
