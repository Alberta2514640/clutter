package generic

import (
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestGetUserIDFromRequest(t *testing.T) {
	tests := []struct {
		name      string
		request   events.APIGatewayProxyRequest
		wantID    string
		wantError bool
	}{
		{
			name: "uuid from authorizer context",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"uuid": "user-123",
					},
				},
			},
			wantID:    "user-123",
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
			name:      "empty request returns error",
			request:   events.APIGatewayProxyRequest{},
			wantID:    "",
			wantError: true,
		},
		{
			name: "empty uuid returns error",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"uuid": "",
					},
				},
			},
			wantID:    "",
			wantError: true,
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
}

func TestResponse_ErrorCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
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
