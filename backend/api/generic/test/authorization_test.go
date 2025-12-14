package generic

import (
	"testing"
	"testing/quick"
)

// Verifies ORG# prefix extraction works for any org ID
func TestExtractOrganizationID_Property(t *testing.T) {
	config := &quick.Config{MaxCount: 100}

	roundTrip := func(orgID string) bool {
		pk := "ORG#" + orgID
		extracted := ExtractOrganizationID(pk)
		return extracted == orgID
	}

	if err := quick.Check(roundTrip, config); err != nil {
		t.Errorf("Property failed: %v", err)
	}
}

func TestExtractOrganizationID_InvalidPrefix(t *testing.T) {
	cases := []string{"", "ORG", "org#test", "PROJECT#123", "INVALID"}
	for _, pk := range cases {
		if got := ExtractOrganizationID(pk); got != "" {
			t.Errorf("ExtractOrganizationID(%q) = %q, want empty", pk, got)
		}
	}
}

func TestAuthorizationError(t *testing.T) {
	// Test NewForbiddenError
	err := NewForbiddenError("test message")
	if err.StatusCode != 403 || err.Code != "FORBIDDEN" || err.Error() != "test message" {
		t.Errorf("NewForbiddenError incorrect: %+v", err)
	}

	// Test NewNotFoundError
	err = NewNotFoundError("not found")
	if err.StatusCode != 404 || err.Code != "NOT_FOUND" || err.Error() != "not found" {
		t.Errorf("NewNotFoundError incorrect: %+v", err)
	}
}

func TestCheckOrganizationMembership_EmptyInputs(t *testing.T) {
	// Empty userID should return forbidden error
	err := CheckOrganizationMembership(nil, nil, "", "", "org123")
	if authErr, ok := err.(*AuthorizationError); !ok || authErr.StatusCode != 403 {
		t.Errorf("Expected 403 for empty userID, got: %v", err)
	}

	// Empty orgID should return forbidden error
	err = CheckOrganizationMembership(nil, nil, "", "user123", "")
	if authErr, ok := err.(*AuthorizationError); !ok || authErr.StatusCode != 403 {
		t.Errorf("Expected 403 for empty orgID, got: %v", err)
	}
}

func TestGetProjectOrganization_EmptyProjectID(t *testing.T) {
	_, err := GetProjectOrganization(nil, nil, "", "")
	if authErr, ok := err.(*AuthorizationError); !ok || authErr.StatusCode != 404 {
		t.Errorf("Expected 404 for empty projectID, got: %v", err)
	}
}
