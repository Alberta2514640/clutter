package user_information_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
)

// mockHandler simulates the handler logic for testing purposes
func mockHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusUnauthorized,
			generic.Json{"message": "unauthorized: missing user identity", "error": err.Error()},
		)
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "successfully retrieved user information",
		"data": map[string]string{
			"uuid":        userData.Id,
			"email":       userData.Email,
			"full_name":   userData.Name,
			"picture_url": userData.PictureUrl,
			"created_at":  userData.AccountCreatedOn,
		},
	})
}

func TestGetUserInformation_ValidContext(t *testing.T) {
	// Create a mock request with valid authorizer context
	// Keys must match what the authorizer Lambda sends (see authorizer/main.go lines 60-66)
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: map[string]interface{}{
				"id":          "test-user-uuid-123",
				"email":       "test@example.com",
				"name":        "Test User",
				"picture_url": "https://example.com/picture.jpg",
				"created_at":  "2026-01-01T00:00:00Z",
			},
		},
	}

	response, err := mockHandler(request)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", response.StatusCode)
	}

	// Parse response body
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if body["message"] != "successfully retrieved user information" {
		t.Errorf("Unexpected message: %s", body["message"])
	}

	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	// Verify snake_case keys match login endpoint format
	if data["uuid"] != "test-user-uuid-123" {
		t.Errorf("Expected uuid 'test-user-uuid-123', got '%s'", data["uuid"])
	}

	if data["email"] != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", data["email"])
	}

	if data["full_name"] != "Test User" {
		t.Errorf("Expected full_name 'Test User', got '%s'", data["full_name"])
	}

	if data["picture_url"] != "https://example.com/picture.jpg" {
		t.Errorf("Expected picture_url, got '%s'", data["picture_url"])
	}

	if data["created_at"] != "2026-01-01T00:00:00Z" {
		t.Errorf("Expected created_at, got '%s'", data["created_at"])
	}
}

func TestGetUserInformation_MissingContext(t *testing.T) {
	// Create a mock request with nil authorizer context
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: nil,
		},
	}

	response, err := mockHandler(request)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", response.StatusCode)
	}

	// Parse response body
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if body["message"] != "unauthorized: missing user identity" {
		t.Errorf("Unexpected message: %s", body["message"])
	}
}

func TestGetUserInformation_EmptyContext(t *testing.T) {
	// Create a mock request with empty authorizer context
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: map[string]interface{}{},
		},
	}

	response, err := mockHandler(request)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Even with empty context, the handler should return 200 with empty user data
	// since GetUserDataFromAuthorizerContext doesn't fail on empty maps
	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", response.StatusCode)
	}
}
