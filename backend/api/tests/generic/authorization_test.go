package generic_test

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func TestAuthorizationError(t *testing.T) {
	// Test NewForbiddenError
	err := generic.NewForbiddenError("test message")
	if err.StatusCode != 403 || err.Code != "FORBIDDEN" || err.Error() != "test message" {
		t.Errorf("NewForbiddenError incorrect: %+v", err)
	}

	// Test NewNotFoundError
	err = generic.NewNotFoundError("not found")
	if err.StatusCode != 404 || err.Code != "NOT_FOUND" || err.Error() != "not found" {
		t.Errorf("NewNotFoundError incorrect: %+v", err)
	}
}
