package main

import "testing"

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
		got := itoa(tc.input)
		if got != tc.want {
			t.Errorf("itoa(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestAllowedJobTypes_ContainsAnsibleAndTerraform(t *testing.T) {
	if !allowedJobTypes["ansible"] {
		t.Error("expected ansible to be an allowed job type")
	}
	if !allowedJobTypes["terraform"] {
		t.Error("expected terraform to be an allowed job type")
	}
	if allowedJobTypes["unknown"] {
		t.Error("expected unknown to not be an allowed job type")
	}
}
