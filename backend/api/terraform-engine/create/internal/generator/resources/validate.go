package resources

import (
	"errors"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
)

// Every canvas value a generator puts in a template is checked here before it
// is rendered: names with checkResourceName, enumerated fields against an
// allow-list owned by the generator that uses it, and free text escaped with
// hclsafe.Escape where the generator builds its template variables. The
// templates do no checking of their own.

// ErrInvalidVariable marks a diagram value the generator refuses. Its message is
// for the user: it names the field and what it accepts.
var ErrInvalidVariable = errors.New("invalid variable")

type invalidVariableError struct{ message string }

func (e invalidVariableError) Error() string { return e.message }
func (e invalidVariableError) Unwrap() error { return ErrInvalidVariable }

func invalidVariable(format string, args ...any) error {
	return invalidVariableError{message: fmt.Sprintf(format, args...)}
}

// checkResourceName checks resource_name, which every generator renders as an
// AWS name and turns into the Terraform identifier.
func checkResourceName(vars map[string]any) error {
	if err := hclsafe.CheckName(generic.GetString(vars, "resource_name", "")); err != nil {
		return invalidVariable("resource_name %v", err)
	}
	return nil
}

// checkOneOf refuses a string value of key that is not in allowed. A missing or
// non-string value passes: generic.GetString then gives the generator's default.
func checkOneOf(vars map[string]any, key string, allowed hclsafe.AllowList) error {
	v, ok := vars[key].(string)
	if !ok || allowed.Contains(v) {
		return nil
	}
	return invalidVariable("%s must be one of: %s", key, allowed)
}
