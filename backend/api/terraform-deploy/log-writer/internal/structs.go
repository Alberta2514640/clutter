// Package internal provides internal types for the terraform-deploy log-writer Lambda.
package internal

// LogStatus represents the possible deployment status values.
type LogStatus string

const (
// StatusSuccess indicates the deployment completed successfully.
StatusSuccess LogStatus = "SUCCESS"
// StatusFailed indicates the deployment failed.
StatusFailed LogStatus = "FAILED"
)

// S3EventRecord represents a single S3 event record from the S3 notification.
type S3EventRecord struct {
	S3 S3Data `json:"s3"`
}

// S3Data contains the bucket and object information from an S3 event.
type S3Data struct {
	Bucket S3Bucket `json:"bucket"`
	Object S3Object `json:"object"`
}

// S3Bucket contains the S3 bucket information.
type S3Bucket struct {
	Name string `json:"name"`
}

// S3Object contains the S3 object information.
type S3Object struct {
	Key string `json:"key"`
}
