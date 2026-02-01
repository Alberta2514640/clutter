// Package main provides the Lambda handler for persisting Terraform deployment logs.
//
// This Lambda is triggered by S3 s3:ObjectCreated:* events on the logs/ prefix.
// When a Terraform runner uploads its execution log to S3, this Lambda:
//  1. Parses the S3 event to extract bucket and key
//  2. Extracts the run_id from the object key (format: logs/{run_id}.log)
//  3. Downloads the log content from S3
//  4. Determines the deployment status from the log content
//  5. Updates the deployment_runs table with the log and status
package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-deploy/log-writer/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, s3Event events.S3Event) error {
	// Connect to PostgreSQL once before processing records
	conn, err := generic.PsqlConnect()
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer conn.Close(ctx)

	// Track success/failure counts and errors
	var successCount, failureCount int
	var errors []string

	// Process each S3 event record
	for _, record := range s3Event.Records {
		bucket := record.S3.Bucket.Name
		rawKey := record.S3.Object.Key

		// URL decode the S3 object key (may be URL-encoded)
		key, err := url.QueryUnescape(rawKey)
		if err != nil {
			fmt.Printf("Error URL-decoding key '%s': %v\n", rawKey, err)
			errors = append(errors, fmt.Sprintf("URL-decode key '%s': %v", rawKey, err))
			failureCount++
			continue
		}

		// 1) Extract run ID from the object key
		runID, err := internal.ExtractRunIDFromKey(key)
		if err != nil {
			fmt.Printf("Error extracting run ID from key '%s': %v\n", key, err)
			errors = append(errors, fmt.Sprintf("extract run ID from '%s': %v", key, err))
			failureCount++
			continue
		}

		// 2) Download log content from S3
		logContent, err := internal.DownloadLogFromS3(ctx, bucket, key)
		if err != nil {
			fmt.Printf("Error downloading log for run '%s': %v\n", runID, err)
			errors = append(errors, fmt.Sprintf("download log for run '%s': %v", runID, err))
			failureCount++
			continue
		}

		// 3) Determine status from log content
		status := internal.DetermineStatusFromLog(logContent)

		// 4) Update deployment_runs table
		updateQuery := `
			UPDATE deployment_runs 
			SET status = $1, log = $2, completed_at = NOW() 
			WHERE id = $3
		`
		result, err := conn.Exec(ctx, updateQuery, string(status), logContent, runID)
		if err != nil {
			fmt.Printf("Error updating deployment run '%s': %v\n", runID, err)
			errors = append(errors, fmt.Sprintf("update deployment run '%s': %v", runID, err))
			failureCount++
			continue
		}

		// Check if any rows were affected
		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			fmt.Printf("Warning: No rows updated for run '%s' - run ID may not exist in database. Query: %s\n", runID, updateQuery)
			errors = append(errors, fmt.Sprintf("no rows updated for run '%s' - run ID may not exist", runID))
			failureCount++
			continue
		}

		fmt.Printf("Successfully processed log for run '%s' with status '%s' (rows affected: %d)\n", runID, status, rowsAffected)
		successCount++
	}

	// Return error if no records succeeded or if there were failures with no successes
	totalRecords := len(s3Event.Records)
	if totalRecords > 0 && successCount == 0 {
		return fmt.Errorf("all %d record(s) failed to process: %v", failureCount, errors)
	}

	// Log summary if there were partial failures
	if failureCount > 0 {
		fmt.Printf("Processing completed with %d success(es) and %d failure(s)\n", successCount, failureCount)
	}

	return nil
}
