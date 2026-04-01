package ansible_test

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/uploadutils"
)

// ── uploadutils.BuildLogObjectKey ──────────────────────────────────────────────

func TestBuildLogObjectKey_ProducesExpectedFormat(t *testing.T) {
	cases := []struct {
		name      string
		orgID     string
		projectID string
		diagramID string
		jobID     string
		want      string
	}{
		{
			name:  "standard IDs",
			orgID: "org-123", projectID: "proj-456", diagramID: "diag-789", jobID: "job-abc",
			want: "org-123/proj-456/diag-789/playbooks/logs/job-abc.log",
		},
		{
			name:      "uuid-style IDs",
			orgID:     "550e8400-e29b-41d4-a716-446655440000",
			projectID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			diagramID: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			jobID:     "7c9e6679-7425-40de-944b-e07fc1f90ae7",
			want:      "550e8400-e29b-41d4-a716-446655440000/6ba7b810-9dad-11d1-80b4-00c04fd430c8/f47ac10b-58cc-4372-a567-0e02b2c3d479/playbooks/logs/7c9e6679-7425-40de-944b-e07fc1f90ae7.log",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := uploadutils.BuildLogObjectKey(tc.orgID, tc.projectID, tc.diagramID, tc.jobID)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

// ── uploadutils.ExtractOrgIDFromLogKey ─────────────────────────────────────────

func TestExtractOrgIDFromLogKey_ValidKey(t *testing.T) {
	key := "org-123/proj-456/diag-789/playbooks/logs/job-abc.log"
	got, err := uploadutils.ExtractOrgIDFromLogKey(key)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != "org-123" {
		t.Fatalf("expected org-123, got %q", got)
	}
}

func TestExtractOrgIDFromLogKey_InvalidKeys(t *testing.T) {
	cases := []string{
		"",
		"no-slash-at-all",
		"/proj-456/diag-789/playbooks/logs/job-abc.log",
		"just-a-filename.log",
	}
	for _, key := range cases {
		t.Run(key, func(t *testing.T) {
			if _, err := uploadutils.ExtractOrgIDFromLogKey(key); err == nil {
				t.Fatalf("expected error for key %q", key)
			}
		})
	}
}

// ── uploadutils.ExtractPathComponentsFromLogKey ────────────────────────────────

func TestExtractPathComponentsFromLogKey_ValidKey(t *testing.T) {
	key := "11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222/33333333-3333-3333-3333-333333333333/playbooks/logs/job-abc.log"
	orgID, projectID, diagramID, err := uploadutils.ExtractPathComponentsFromLogKey(key)
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

func TestExtractPathComponentsFromLogKey_InvalidKeys(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"no slashes", "no-slash-at-all"},
		{"only two parts", "11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222"},
		{"only three parts", "11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222/33333333-3333-3333-3333-333333333333"},
		{"empty org", "/22222222-2222-2222-2222-222222222222/33333333-3333-3333-3333-333333333333/playbooks/logs/job-abc.log"},
		{"empty project", "11111111-1111-1111-1111-111111111111//33333333-3333-3333-3333-333333333333/playbooks/logs/job-abc.log"},
		{"empty diagram", "11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222//playbooks/logs/job-abc.log"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := uploadutils.ExtractPathComponentsFromLogKey(tc.key); err == nil {
				t.Fatalf("expected error for key %q", tc.key)
			}
		})
	}
}
