package generic_test

import (
	"net/http"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func TestResponse_Success(t *testing.T) {
	resp, err := generic.Response(http.StatusOK, generic.Json{
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
		resp, err := generic.Response(code, generic.Json{"success": false})
		if err != nil {
			t.Errorf("Response(%d) returned error: %v", code, err)
		}
		if resp.StatusCode != code {
			t.Errorf("Response(%d) StatusCode = %d", code, resp.StatusCode)
		}
	}
}
