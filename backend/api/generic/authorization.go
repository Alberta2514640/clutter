package generic

// AuthorizationError represents an authorization failure with HTTP status details
type AuthorizationError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *AuthorizationError) Error() string {
	return e.Message
}

// NewForbiddenError creates a 403 Forbidden authorization error
func NewForbiddenError(message string) *AuthorizationError {
	return &AuthorizationError{
		StatusCode: 403,
		Code:       "FORBIDDEN",
		Message:    message,
	}
}

// NewNotFoundError creates a 404 Not Found error
func NewNotFoundError(message string) *AuthorizationError {
	return &AuthorizationError{
		StatusCode: 404,
		Code:       "NOT_FOUND",
		Message:    message,
	}
}
