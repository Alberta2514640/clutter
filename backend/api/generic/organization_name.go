package generic

import (
	"strings"
	"unicode/utf8"
)

// The limits on an organization's name, the same as the website's
// (frontend/clutter/src/lib/features/organization/rules.ts). The database
// holds up to 32 characters (organizations.name is VARCHAR(32)).
const (
	OrganizationNameMin = 4
	OrganizationNameMax = 32
)

// OrganizationNameProblem is the 400 message for a name outside the limits.
const OrganizationNameProblem = "Use 4 to 32 characters for the organization name."

// OrganizationNameOK reports whether name fits the limits: at least
// OrganizationNameMin characters once trimmed, and at most OrganizationNameMax
// as stored. Characters are counted as the database counts them, not bytes.
func OrganizationNameOK(name string) bool {
	return utf8.RuneCountInString(strings.TrimSpace(name)) >= OrganizationNameMin &&
		utf8.RuneCountInString(name) <= OrganizationNameMax
}
