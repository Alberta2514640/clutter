package generic

import (
	"testing"
)

func TestCheckOrganizationMembershipPSQL_EmptyInputs(t *testing.T) {
	// Test with empty user ID
	err := CheckOrganizationMembershipPSQL(nil, nil, "", "org123")
	if err == nil {
		t.Error("Expected error for empty userID, got nil")
	}
	authErr, ok := err.(*AuthorizationError)
	if !ok || authErr.StatusCode != 403 {
		t.Errorf("Expected 403 Forbidden error, got: %v", err)
	}

	// Test with empty org ID
	err = CheckOrganizationMembershipPSQL(nil, nil, "user123", "")
	if err == nil {
		t.Error("Expected error for empty orgID, got nil")
	}
	authErr, ok = err.(*AuthorizationError)
	if !ok || authErr.StatusCode != 403 {
		t.Errorf("Expected 403 Forbidden error, got: %v", err)
	}
}

func TestGetProjectOrganizationPSQL_EmptyProjectID(t *testing.T) {
	_, err := GetProjectOrganizationPSQL(nil, nil, "")
	if err == nil {
		t.Error("Expected error for empty projectID, got nil")
	}
	authErr, ok := err.(*AuthorizationError)
	if !ok || authErr.StatusCode != 404 {
		t.Errorf("Expected 404 Not Found error, got: %v", err)
	}
}

func TestGetDiagramProjectPSQL_EmptyDiagramID(t *testing.T) {
	_, err := GetDiagramProjectPSQL(nil, nil, "")
	if err == nil {
		t.Error("Expected error for empty diagramID, got nil")
	}
	authErr, ok := err.(*AuthorizationError)
	if !ok || authErr.StatusCode != 404 {
		t.Errorf("Expected 404 Not Found error, got: %v", err)
	}
}
