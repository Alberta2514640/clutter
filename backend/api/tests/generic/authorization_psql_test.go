package generic_test

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func TestCheckOrganizationMembershipPSQL_EmptyInputs(t *testing.T) {
	// Test with empty user ID
	err := generic.CheckOrganizationMembershipPSQL(nil, nil, "", "org123")
	if err == nil {
		t.Error("Expected error for empty userID, got nil")
	}
	authErr, ok := err.(*generic.AuthorizationError)
	if !ok || authErr.StatusCode != 403 {
		t.Errorf("Expected 403 Forbidden error, got: %v", err)
	}

	// Test with empty org ID
	err = generic.CheckOrganizationMembershipPSQL(nil, nil, "user123", "")
	if err == nil {
		t.Error("Expected error for empty orgID, got nil")
	}
	authErr, ok = err.(*generic.AuthorizationError)
	if !ok || authErr.StatusCode != 403 {
		t.Errorf("Expected 403 Forbidden error, got: %v", err)
	}
}

func TestGetProjectOrganizationPSQL_EmptyProjectID(t *testing.T) {
	_, err := generic.GetProjectOrganizationPSQL(nil, nil, "")
	if err == nil {
		t.Error("Expected error for empty projectID, got nil")
	}
	authErr, ok := err.(*generic.AuthorizationError)
	if !ok || authErr.StatusCode != 404 {
		t.Errorf("Expected 404 Not Found error, got: %v", err)
	}
}

func TestGetDiagramProjectPSQL_EmptyDiagramID(t *testing.T) {
	_, err := generic.GetDiagramProjectPSQL(nil, nil, "")
	if err == nil {
		t.Error("Expected error for empty diagramID, got nil")
	}
	authErr, ok := err.(*generic.AuthorizationError)
	if !ok || authErr.StatusCode != 404 {
		t.Errorf("Expected 404 Not Found error, got: %v", err)
	}
}
