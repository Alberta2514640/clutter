// Package hclsafe makes values safe to place in generated Terraform (HCL).
//
// A value that reaches a generated .tf file is either checked (a name against
// CheckName, an enumerated value against an AllowList) or escaped with Escape.
// Generated code runs with the customer's credentials, so an unescaped value is
// code injection into the customer's account.
package hclsafe

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// MaxNameLength caps a name. Names also become cloud resource names, and most
// providers cap those at 63 or 64 characters.
const MaxNameLength = 64

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// CheckName returns an error unless name starts with a lowercase letter, holds
// only lowercase letters, digits, hyphens and underscores, and is at most
// MaxNameLength characters long. A name that passes is safe inside an HCL
// string as is, and is a valid Terraform identifier.
//
// The error text completes a sentence that starts with the field's name, such
// as "resource_name " + err.Error().
func CheckName(name string) error {
	if name == "" {
		return errors.New("is required")
	}
	if len(name) > MaxNameLength {
		return fmt.Errorf("must be at most %d characters", MaxNameLength)
	}
	if !namePattern.MatchString(name) {
		return errors.New("must start with a lowercase letter and use only lowercase letters, digits, hyphens and underscores")
	}
	return nil
}

// Escape returns s escaped for use between the double quotes of an HCL quoted
// string: backslashes, quotes and control characters become escape sequences,
// and "${" and "%{" become "$${" and "%%{" so Terraform reads them as text, not
// as an interpolation or a directive. Bytes that are not valid UTF-8 become
// U+FFFD. The result holds no newline.
func Escape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == '"':
			b.WriteString(`\"`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f || r == ' ' || r == ' ':
			fmt.Fprintf(&b, `\u%04X`, r)
		case (r == '$' || r == '%') && i+1 < len(s) && s[i+1] == '{':
			// "$${" and "%%{" are HCL's escapes for a literal "${" and "%{".
			b.WriteRune(r)
			b.WriteRune(r)
		default:
			// range yields U+FFFD for a byte that is not valid UTF-8.
			b.WriteRune(r)
		}
	}
	return b.String()
}

// AllowList is a fixed set of accepted values, kept in the order it was given
// so that messages list them the way their owner wrote them.
type AllowList struct {
	values []string
	set    map[string]struct{}
}

// NewAllowList returns an AllowList of values.
func NewAllowList(values ...string) AllowList {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return AllowList{values: append([]string(nil), values...), set: set}
}

// Contains reports whether v is one of the accepted values. It is exact: case
// and spaces matter.
func (a AllowList) Contains(v string) bool {
	_, ok := a.set[v]
	return ok
}

// Values returns a copy of the accepted values in their given order.
func (a AllowList) Values() []string {
	return append([]string(nil), a.values...)
}

// String lists the accepted values for a message, such as "a, b, c".
func (a AllowList) String() string {
	return strings.Join(a.values, ", ")
}
