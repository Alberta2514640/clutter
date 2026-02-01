// Package internal provides internal utilities for the terraform-deploy log-writer Lambda.
package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	// defaultAWSRegion is used when AWS_REGION environment variable is not set.
	defaultAWSRegion = "us-east-1"
)

// getAWSRegion returns the AWS region from environment or falls back to default.
func getAWSRegion() string {
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	return defaultAWSRegion
}

// ExtractRunIDFromKey extracts the run ID from an S3 object key.
// Expected format: logs/{run_id}.log
// Returns the run ID or an error if the format is invalid.
func ExtractRunIDFromKey(key string) (string, error) {
	// Remove "logs/" prefix
	if !strings.HasPrefix(key, "logs/") {
		return "", fmt.Errorf("invalid key format: expected prefix 'logs/', got '%s'", key)
	}

	filename := strings.TrimPrefix(key, "logs/")

	// Remove ".log" suffix
	if !strings.HasSuffix(filename, ".log") {
		return "", fmt.Errorf("invalid key format: expected suffix '.log', got '%s'", filename)
	}

	runID := strings.TrimSuffix(filename, ".log")

	if runID == "" {
		return "", fmt.Errorf("invalid key format: empty run ID")
	}

	return runID, nil
}

// DownloadLogFromS3 downloads the log content from S3.
// Returns the log content as a string or an error.
func DownloadLogFromS3(ctx context.Context, bucket, key string) (string, error) {
	// Validate inputs
	if bucket == "" {
		return "", fmt.Errorf("bucket name cannot be empty")
	}
	if key == "" {
		return "", fmt.Errorf("object key cannot be empty")
	}

	// Load AWS config with explicit region to avoid Lambda/VPC region detection failures
	region := getAWSRegion()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Get the object
	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get S3 object: %w", err)
	}
	defer result.Body.Close()

	// Read the content
	content, err := io.ReadAll(result.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return string(content), nil
}

// DetermineStatusFromLog analyzes the log content to determine deployment status.
// Returns SUCCESS if "Apply complete!" is found, otherwise FAILED.
func DetermineStatusFromLog(logContent string) LogStatus {
	// Check for success indicators
	if strings.Contains(logContent, "Apply complete!") ||
		strings.Contains(logContent, "Deployment Completed Successfully") {
		return StatusSuccess
	}

	// Default to failed if success indicators not found
	return StatusFailed
}
