package ansible_test

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/jobsutils"
)

func TestItoa_ConvertsSmallIntsToStrings(t *testing.T) {
	cases := []struct {
		input int
		want  string
	}{
		{2, "2"},
		{3, "3"},
		{4, "4"},
		{9, "9"},
	}

	for _, tc := range cases {
		got := jobsutils.Itoa(tc.input)
		if got != tc.want {
			t.Errorf("Itoa(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestAllowedJobTypes_ContainsAnsibleAndTerraform(t *testing.T) {
	if !jobsutils.AllowedJobTypes["ansible"] {
		t.Error("expected ansible to be an allowed job type")
	}
	if !jobsutils.AllowedJobTypes["terraform"] {
		t.Error("expected terraform to be an allowed job type")
	}
	if jobsutils.AllowedJobTypes["unknown"] {
		t.Error("expected unknown to not be an allowed job type")
	}
}
