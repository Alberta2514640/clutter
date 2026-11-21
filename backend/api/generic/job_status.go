package generic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
)

// The ansible-runner task writes a job's status file at
// <org>/<project>/<diagram>/playbooks/logs/<job id>.status.json, beside the
// job's log (JobLogKey): RUNNING once it starts, then COMPLETED or FAILED with
// the log key once the log is uploaded (entrypoint.sh, write_job_status).
// ReconcileJob copies it into the job's row.

// Job statuses. QUEUED and STARTING are written by submit-job and run-task.
const (
	JobQueued    = "QUEUED"
	JobStarting  = "STARTING"
	JobRunning   = "RUNNING"
	JobCompleted = "COMPLETED"
	JobFailed    = "FAILED"
)

// The runner caps its messages at 500 characters; this leaves room for UTF-8.
const maxJobErrorBytes = 2000

// JobStatusFile is the content of a job's status file.
type JobStatusFile struct {
	Status       string `json:"status"`
	JobID        string `json:"job_id"`
	ErrorMessage string `json:"error_message"`
	LogS3Key     string `json:"log_s3_key"`
}

// Execer is the part of a Postgres connection ReconcileJob needs.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// JobFolder is a job's place in the clutter bucket: the organization, project
// and diagram of its playbook key, which submit-job checked belong together
// (CheckOwnedIDsPSQL) before it wrote the job.
type JobFolder struct {
	OrgID, ProjectID, DiagramID string
}

// JobLogKey is the key the runner uploads a job's log to.
func JobLogKey(folder JobFolder, jobID string) string {
	return folder.OrgID + "/" + folder.ProjectID + "/" + folder.DiagramID + "/playbooks/logs/" + jobID + ".log"
}

// JobStatusKey is the key of a job's status file.
func JobStatusKey(folder JobFolder, jobID string) string {
	return folder.OrgID + "/" + folder.ProjectID + "/" + folder.DiagramID + "/playbooks/logs/" + jobID + ".status.json"
}

// IsJobFinished reports whether a job's status can no longer change.
func IsJobFinished(status string) bool {
	return status == JobCompleted || status == JobFailed
}

// ReconcileJob moves a job that is not finished to the status in its status
// file. The caller must already have shown that the user may read the job.
// It returns the file it recorded, or nil when nothing changed: the job is
// finished, the file is missing or not valid, or the row moved on meanwhile.
// Every failure is returned for the caller to log; none should fail the read.
func ReconcileJob(ctx context.Context, conn Execer, store StatusStore, jobID, currentStatus string, folder JobFolder) (*JobStatusFile, error) {
	if IsJobFinished(currentStatus) {
		return nil, nil
	}
	var file JobStatusFile
	found, err := readStatusFile(ctx, store, JobStatusKey(folder, jobID), &file)
	if err != nil || !found {
		return nil, err
	}
	if err := validateJobStatusFile(file, jobID, folder); err != nil {
		return nil, err
	}

	var tag pgconn.CommandTag
	switch file.Status {
	case JobRunning:
		if currentStatus == JobRunning {
			return nil, nil
		}
		tag, err = conn.Exec(ctx, `
			UPDATE jobs
			SET status = 'RUNNING', updated_at = NOW()
			WHERE id = $1 AND status IN ('QUEUED', 'STARTING')
		`, jobID)
	default:
		// A run-task error message stays when the runner gives none.
		tag, err = conn.Exec(ctx, `
			UPDATE jobs
			SET status = $1,
			    error_message = COALESCE(NULLIF($2, ''), error_message),
			    log_s3_key = COALESCE(NULLIF($3, ''), log_s3_key),
			    updated_at = NOW()
			WHERE id = $4 AND status IN ('QUEUED', 'STARTING', 'RUNNING')
		`, file.Status, file.ErrorMessage, file.LogS3Key, jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("record job status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return &file, nil
}

// validateJobStatusFile refuses a file that is not this job's: another job's
// id, a status the runner never writes, or a log key outside the job's own
// log, which get-job-logs would otherwise read for the caller.
func validateJobStatusFile(file JobStatusFile, jobID string, folder JobFolder) error {
	if !strings.EqualFold(file.JobID, jobID) {
		return errors.New("status file names another job")
	}
	switch file.Status {
	case JobRunning, JobCompleted:
		if file.ErrorMessage != "" {
			return errors.New("status file has an error message without a failure")
		}
	case JobFailed:
	default:
		return errors.New("status file has an unknown status")
	}
	if file.Status == JobRunning && file.LogS3Key != "" {
		return errors.New("status file has a log before the job finished")
	}
	if file.LogS3Key != "" && file.LogS3Key != JobLogKey(folder, jobID) {
		return errors.New("status file names another log")
	}
	if len(file.ErrorMessage) > maxJobErrorBytes || !utf8.ValidString(file.ErrorMessage) {
		return errors.New("status file has an invalid error message")
	}
	return nil
}
