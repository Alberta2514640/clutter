package main

import (
	"testing"
	"testing/quick"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
)

func TestCreateAuthorization_Property(t *testing.T) {
	config := &quick.Config{MaxCount: 100}

	// Property: getUserIDFromRequest correctly extracts userId from any valid header
	extractsUserId := func(userID string) bool {
		if userID == "" {
			return true // Skip empty - tested separately
		}
		req := events.APIGatewayProxyRequest{
			Headers: map[string]string{"x-user-id": userID},
		}
		extracted, err := generic.GetUserIDFromRequest(req)
		return err == nil && extracted == userID
	}

	if err := quick.Check(extractsUserId, config); err != nil {
		t.Errorf("Property failed: %v", err)
	}
}

func TestGetUserIDFromRequest_Authorizer(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: map[string]interface{}{
				"userId": "user-123",
			},
		},
	}

	userID, err := generic.GetUserIDFromRequest(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("Expected user-123, got %s", userID)
	}
}

func TestGetUserIDFromRequest_Header(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		Headers: map[string]string{"x-user-id": "user-456"},
	}

	userID, err := generic.GetUserIDFromRequest(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if userID != "user-456" {
		t.Errorf("Expected user-456, got %s", userID)
	}
}

func TestGetUserIDFromRequest_Missing(t *testing.T) {
	req := events.APIGatewayProxyRequest{}

	_, err := generic.GetUserIDFromRequest(req)
	if err == nil {
		t.Error("Expected error for missing user ID")
	}
}

func TestGetUserIDFromRequest_Sub(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: map[string]interface{}{
				"sub": "sub-789",
			},
		},
	}

	userID, err := generic.GetUserIDFromRequest(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if userID != "sub-789" {
		t.Errorf("Expected sub-789, got %s", userID)
	}
}
