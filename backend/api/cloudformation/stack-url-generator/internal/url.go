package internal

import (
	"net/url"
	"strings"
)

// Source ChatGPT
func BuildCloudFormationURL(region string, params map[string]string) string {

	// Base URL with region
	base := "https://" + region + ".console.aws.amazon.com/cloudformation/home?region=" + region

	// Build fragment query (everything after #)
	var fragmentParts []string
	for k, v := range params {
		// URL-encode each value
		fragmentParts = append(fragmentParts, k+"="+url.QueryEscape(v))
	}
	fragmentQuery := strings.Join(fragmentParts, "&")

	// Return final URL
	return base + "#/stacks/create/review?" + fragmentQuery

}
