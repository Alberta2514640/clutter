package main

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func TestGetUserDataFromAuthorizerContext_Valid(t *testing.T) {
	context := map[string]interface{}{
		"id":               "user-123",
		"email":            "test@example.com",
		"name":             "Test User",
		"pictureUrl":       "https://example.com/pic.jpg",
		"accountCreatedOn": "2024-01-01T00:00:00Z",
	}

	userData, err := generic.GetUserDataFromAuthorizerContext(context)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if userData.Id != "user-123" {
		t.Errorf("Expected user-123, got %s", userData.Id)
	}
	if userData.Email != "test@example.com" {
		t.Errorf("Expected test@example.com, got %s", userData.Email)
	}
}

func TestGetUserDataFromAuthorizerContext_MissingContext(t *testing.T) {
	_, err := generic.GetUserDataFromAuthorizerContext(nil)
	if err == nil {
		t.Error("Expected error for nil context")
	}
}

func TestGetUserDataFromAuthorizerContext_EmptyContext(t *testing.T) {
	context := map[string]interface{}{}
	userData, err := generic.GetUserDataFromAuthorizerContext(context)
	// Empty context should not error, but should have empty values
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if userData.Id != "" {
		t.Errorf("Expected empty ID for empty context, got %s", userData.Id)
	}
}
