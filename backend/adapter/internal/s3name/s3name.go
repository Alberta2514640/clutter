// Package s3name owns the shape of an S3 bucket name B accepts for its own
// buckets (logs, code, run files, state), so every place that reads one checks
// it the same way.
package s3name

import "regexp"

// BucketPattern is an S3 bucket name: 3 to 63 lowercase letters, digits,
// dots and hyphens, starting and ending with a letter or digit.
var BucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
