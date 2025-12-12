package main

import (
	"testing"
	"testing/quick"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
)

func TestUpdateAuthorization_Property(t *testing.T) {
	config := &quick.Config{MaxCount: 100}

	extractsUserId := func(userID string) bool {
		if userID == "" {
			return true
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
			Authorizer: map[string]interface{}{"userId": "user-123"},
		},
	}

	userID, err := generic.GetUserIDFromRequest(req)
	if err != nil || userID != "user-123" {
		t.Errorf("Expected user-123, got %s, err: %v", userID, err)
	}
}

func TestGetUserIDFromRequest_Header(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		Headers: map[string]string{"x-user-id": "user-456"},
	}

	userID, err := generic.GetUserIDFromRequest(req)
	if err != nil || userID != "user-456" {
		t.Errorf("Expected user-456, got %s, err: %v", userID, err)
	}
}

func TestGetUserIDFromRequest_Missing(t *testing.T) {
	req := events.APIGatewayProxyRequest{}
	_, err := generic.GetUserIDFromRequest(req)
	if err == nil {
		t.Error("Expected error for missing user ID")
	}
}
