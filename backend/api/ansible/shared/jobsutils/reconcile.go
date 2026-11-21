package jobsutils

import (
	"context"
	"log"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/uploadutils"
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// JobState is the part of a job row the runner's status file can change, as
// a read handler holds it.
type JobState struct {
	ID            string
	Status        string
	ErrorMessage  *string
	LogS3Key      *string
	UpdatedAt     time.Time
	PlaybookS3Key *string
}

// Reconcile records the runner's status file for a job that is not finished
// and updates state to match, so the answer shows what was recorded. The
// caller must already have shown that the user may read the job. A nil store
// (no bucket configured) or any failure leaves state as it is; failures are
// logged, never returned, so the read goes on.
func Reconcile(ctx context.Context, conn generic.Execer, store *generic.StatusStore, state *JobState) {
	if store == nil || generic.IsJobFinished(state.Status) {
		return
	}
	folder, err := uploadutils.JobFolderFromPlaybookKey(state.PlaybookS3Key)
	if err != nil {
		// Terraform jobs and rows without a playbook have no status file.
		return
	}
	file, err := generic.ReconcileJob(ctx, conn, *store, state.ID, state.Status, folder)
	if err != nil {
		log.Printf("job status: job %s: %v", state.ID, err)
		return
	}
	if file == nil {
		return
	}
	state.Status = file.Status
	state.UpdatedAt = time.Now().UTC()
	if file.ErrorMessage != "" {
		message := file.ErrorMessage
		state.ErrorMessage = &message
	}
	if file.LogS3Key != "" {
		key := file.LogS3Key
		state.LogS3Key = &key
	}
}
