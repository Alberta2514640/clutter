package generic

import (
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestGetUserIDFromRequest_FromAuthorizer(t *testing.T) {
	tests := []struct {
		name      string
		request   events.APIGatewayProxyRequest
		wantID    string
		wantError bool
	}{
		{
			name: "userId from authorizer context",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"userId": "user-123",
					},
				},
			},
			wantID:    "user-123",
			wantError: false,
		},
		{
			name: "sub from authorizer context",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"sub": "sub-456",
					},
				},
			},
			wantID:    "sub-456",
			wantError: false,
		},
		{
			name: "email from authorizer context",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"email": "user@example.com",
					},
				},
			},
			wantID:    "user@example.com",
			wantError: false,
		},
		{
			name: "x-user-id from headers",
			request: events.APIGatewayProxyRequest{
				Headers: map[string]string{
					"x-user-id": "header-user-789",
				},
			},
			wantID:    "header-user-789",
			wantError: false,
		},
		{
			name: "X-User-Id from headers (case insensitive)",
			request: events.APIGatewayProxyRequest{
				Headers: map[string]string{
					"X-User-Id": "header-user-upper",
				},
			},
			wantID:    "header-user-upper",
			wantError: false,
		},
		{
			name: "userId takes priority over sub",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"userId": "priority-user",
						"sub":    "secondary-sub",
					},
				},
			},
			wantID:    "priority-user",
			wantError: false,
		},
		{
			name:      "empty request returns error",
			request:   events.APIGatewayProxyRequest{},
			wantID:    "",
			wantError: true,
		},
		{
			name: "empty userId in authorizer returns error",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"userId": "",
					},
				},
			},
			wantID:    "",
			wantError: true,
		},
		{
			name: "non-string userId in authorizer falls through",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"userId": 12345, // not a string
						"sub":    "fallback-sub",
					},
				},
			},
			wantID:    "fallback-sub",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := GetUserIDFromRequest(tt.request)
			if tt.wantError {
				if err == nil {
					t.Errorf("GetUserIDFromRequest() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetUserIDFromRequest() unexpected error: %v", err)
				}
				if gotID != tt.wantID {
					t.Errorf("GetUserIDFromRequest() = %q, want %q", gotID, tt.wantID)
				}
			}
		})
	}
}

func TestResponse_Success(t *testing.T) {
	resp, err := Response(http.StatusOK, Json{
		"success": true,
		"message": "test",
	})

	if err != nil {
		t.Errorf("Response() returned error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Response() StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("Response() Content-Type = %q, want %q", resp.Headers["Content-Type"], "application/json")
	}

	if resp.Headers["Access-Control-Allow-Origin"] != "*" {
		t.Errorf("Response() missing CORS header")
	}

	// Check body contains expected JSON
	expectedSubstring := `"success":true`
	if !containsSubstring(resp.Body, expectedSubstring) {
		t.Errorf("Response() Body missing %q, got: %s", expectedSubstring, resp.Body)
	}
}

func TestResponse_WithCookie(t *testing.T) {
	resp, err := Response(http.StatusOK, Json{"test": true}, "session=abc123; HttpOnly")

	if err != nil {
		t.Errorf("Response() returned error: %v", err)
	}

	if resp.Headers["Set-Cookie"] != "session=abc123; HttpOnly" {
		t.Errorf("Response() Set-Cookie = %q, want %q", resp.Headers["Set-Cookie"], "session=abc123; HttpOnly")
	}
}

func TestResponse_ErrorCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	}

	for _, code := range statusCodes {
		resp, err := Response(code, Json{"success": false})
		if err != nil {
			t.Errorf("Response(%d) returned error: %v", code, err)
		}
		if resp.StatusCode != code {
			t.Errorf("Response(%d) StatusCode = %d", code, resp.StatusCode)
		}
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
