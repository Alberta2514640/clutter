package generic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// A's runner tasks (terraform-deployer and ansible-runner) have no database
// access. When a run ends, the task writes a small status file beside the
// run's logs in the clutter bucket, and the read paths the frontend polls copy
// it into the run's row: ReconcileDeployRuns for diagram_deployment_logs
// (terraform-deployer's write_status), ReconcileJob for jobs (job_status.go,
// ansible-runner's write_job_status). A missing, unreadable or malformed file
// leaves the row as it is.
const (
	DeployStatusFile = "status.json"

	// The runners write well under 1 KB; anything far larger is not theirs.
	maxStatusFileBytes = 4 << 10
	// Bounds the S3 reads one poll can cause. Rows beyond it are reconciled
	// once the newer runs have finished.
	maxRunsPerReconcile = 10
	// A run longer than a week is not a real duration, and the column is INT.
	maxDeploySeconds = 7 * 24 * 60 * 60
)

// DeployStatus is the content of a Terraform run's status file.
type DeployStatus struct {
	Status          string `json:"status"`
	CommandID       string `json:"command_id"`
	DurationSeconds int    `json:"duration_seconds"`
}

// StatusReader is the part of the S3 client the reconcilers need.
type StatusReader interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// StatusStore is the bucket the runners write their status files to.
type StatusStore struct {
	Reader StatusReader
	Bucket string
}

// DeployStatusKey is the key of one Terraform run's status file.
func DeployStatusKey(ids OwnedIDs, commandID string) string {
	return ids.OrgID + "/" + ids.ProjectID + "/" + ids.DiagramID + "/terraform/logs/" + commandID + "/" + DeployStatusFile
}

// ReconcileDeployRuns sets the final status and duration of the diagram's
// runs still marked RUNNING, from each run's status file. ids must come from
// CheckOwnedIDsPSQL, so the rows and keys touched are the caller's own. Only a
// failure to list the rows is returned. It is safe to call concurrently: a
// row is updated only while it is still RUNNING.
func ReconcileDeployRuns(ctx context.Context, conn Conn, store StatusStore, ids OwnedIDs) error {
	rows, err := conn.Query(ctx, `
		SELECT command_id
		FROM public.diagram_deployment_logs
		WHERE diagram_id = $1 AND status = 'RUNNING'
		ORDER BY created_at DESC
		LIMIT $2
	`, ids.DiagramID, maxRunsPerReconcile)
	if err != nil {
		return fmt.Errorf("list running deploys: %w", err)
	}
	var commandIDs []string
	for rows.Next() {
		var commandID string
		if err := rows.Scan(&commandID); err != nil {
			rows.Close()
			return fmt.Errorf("scan running deploy: %w", err)
		}
		commandIDs = append(commandIDs, commandID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("list running deploys: %w", err)
	}

	for _, commandID := range commandIDs {
		status, err := readDeployStatus(ctx, store, DeployStatusKey(ids, commandID), commandID)
		if err != nil {
			log.Printf("deploy status: run %s: %v", commandID, err)
			continue
		}
		if status == nil {
			continue
		}
		if _, err := conn.Exec(ctx, `
			UPDATE public.diagram_deployment_logs
			SET status = $1, duration_seconds = $2
			WHERE diagram_id = $3 AND command_id = $4 AND status = 'RUNNING'
		`, status.Status, status.DurationSeconds, ids.DiagramID, commandID); err != nil {
			log.Printf("deploy status: run %s: record: %v", commandID, err)
		}
	}
	return nil
}

// readDeployStatus returns the run's status, nil when the file is not there
// yet (the run is still going, or ended without writing it), or an error
// when the file cannot be read or is not a finished run's status.
func readDeployStatus(ctx context.Context, store StatusStore, key, commandID string) (*DeployStatus, error) {
	var status DeployStatus
	found, err := readStatusFile(ctx, store, key, &status)
	if err != nil || !found {
		return nil, err
	}
	if status.Status != "SUCCESS" && status.Status != "FAILED" {
		return nil, errors.New("status file has an unknown status")
	}
	if status.CommandID != commandID {
		return nil, errors.New("status file names another run")
	}
	if status.DurationSeconds < 0 || status.DurationSeconds > maxDeploySeconds {
		return nil, errors.New("status file has an impossible duration")
	}
	return &status, nil
}

// readStatusFile decodes the JSON file at key into v. It reports false with
// no error when the file does not exist, and an error when it cannot be read,
// is too large or is not JSON.
func readStatusFile(ctx context.Context, store StatusStore, key string, v any) (bool, error) {
	out, err := store.Reader.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(store.Bucket), Key: aws.String(key)})
	if err != nil {
		var missing *types.NoSuchKey
		if errors.As(err, &missing) {
			return false, nil
		}
		return false, fmt.Errorf("read status file: %w", err)
	}
	defer out.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(out.Body, maxStatusFileBytes+1))
	if err != nil {
		return false, fmt.Errorf("read status file: %w", err)
	}
	if len(raw) > maxStatusFileBytes {
		return false, errors.New("status file is too large")
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return false, errors.New("status file is not valid JSON")
	}
	return true, nil
}

// newStatusReader builds the S3 client. Tests replace it.
var newStatusReader = func(ctx context.Context) (StatusReader, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}

var warnNoStatusBucket sync.Once

// StatusStoreFromEnv is the clutter bucket from S3_BUCKET_NAME, read with the
// Lambda's own S3 client. It is nil, with the cause logged, when the bucket is
// not set or the client cannot be built; the caller then answers with the
// rows as they stand.
func StatusStoreFromEnv(ctx context.Context) *StatusStore {
	bucket := os.Getenv("S3_BUCKET_NAME")
	if bucket == "" {
		warnNoStatusBucket.Do(func() {
			log.Printf("run status: S3_BUCKET_NAME is not set; runs keep the status they have")
		})
		return nil
	}
	reader, err := newStatusReader(ctx)
	if err != nil {
		log.Printf("run status: load AWS config: %v", err)
		return nil
	}
	return &StatusStore{Reader: reader, Bucket: bucket}
}

// ReconcileDeployRunsFromEnv is ReconcileDeployRuns with StatusStoreFromEnv.
// A read handler calls it before it answers; every failure is logged and the
// read goes on.
func ReconcileDeployRunsFromEnv(ctx context.Context, conn Conn, ids OwnedIDs) {
	store := StatusStoreFromEnv(ctx)
	if store == nil {
		return
	}
	if err := ReconcileDeployRuns(ctx, conn, *store, ids); err != nil {
		log.Printf("deploy status: %v", err)
	}
}
