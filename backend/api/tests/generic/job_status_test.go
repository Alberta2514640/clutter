package generic_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	jobID      = "66666666-6666-4666-8666-666666666666"
	otherJobID = "99999999-9999-4999-8999-999999999999"
)

var jobFolder = generic.JobFolder{OrgID: ownOrg, ProjectID: ownProject, DiagramID: ownDiagram}

func jobStatusKey() string {
	return ownOrg + "/" + ownProject + "/" + ownDiagram + "/playbooks/logs/" + jobID + ".status.json"
}

func jobLogKey() string {
	return ownOrg + "/" + ownProject + "/" + ownDiagram + "/playbooks/logs/" + jobID + ".log"
}

func jobFiles(body string) *statusFiles {
	return &statusFiles{objects: map[string]string{jobStatusKey(): body}}
}

const finishSQL = `UPDATE jobs\s+SET status = \$1,\s+error_message = COALESCE\(NULLIF\(\$2, ''\), error_message\),\s+log_s3_key = COALESCE\(NULLIF\(\$3, ''\), log_s3_key\)`

func TestJobKeysMatchTheRunner(t *testing.T) {
	if got := generic.JobLogKey(jobFolder, jobID); got != jobLogKey() {
		t.Errorf("log key %s", got)
	}
	if got := generic.JobStatusKey(jobFolder, jobID); got != jobStatusKey() {
		t.Errorf("status key %s", got)
	}
}

func TestReconcileJobRecordsAFinishedJob(t *testing.T) {
	cases := map[string]struct {
		body, status, message, logKey string
	}{
		"completed":           {`{"status":"COMPLETED","job_id":"` + jobID + `","error_message":"","log_s3_key":"` + jobLogKey() + `"}`, "COMPLETED", "", jobLogKey()},
		"failed":              {`{"status":"FAILED","job_id":"` + jobID + `","error_message":"Playbook failed on one or more instances","log_s3_key":"` + jobLogKey() + `"}`, "FAILED", "Playbook failed on one or more instances", jobLogKey()},
		"failed before a log": {`{"status":"FAILED","job_id":"` + jobID + `","error_message":"Failed to assume client AWS role","log_s3_key":""}`, "FAILED", "Failed to assume client AWS role", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			mock, err := pgxmock.NewConn()
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectExec(finishSQL).WithArgs(c.status, c.message, c.logKey, jobID).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			files := jobFiles(c.body)

			got, err := generic.ReconcileJob(context.Background(), mock, store(files), jobID, "RUNNING", jobFolder)
			if err != nil {
				t.Fatal(err)
			}
			if got == nil || got.Status != c.status || got.ErrorMessage != c.message || got.LogS3Key != c.logKey {
				t.Fatalf("recorded %+v", got)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			if len(files.reads) != 1 || files.reads[0] != deployBucket+"/"+jobStatusKey() {
				t.Fatalf("read %v", files.reads)
			}
		})
	}
}

func TestReconcileJobMarksAStartedJobRunning(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(`UPDATE jobs\s+SET status = 'RUNNING', updated_at = NOW\(\)\s+WHERE id = \$1 AND status IN \('QUEUED', 'STARTING'\)`).
		WithArgs(jobID).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	files := jobFiles(`{"status":"RUNNING","job_id":"` + jobID + `"}`)

	got, err := generic.ReconcileJob(context.Background(), mock, store(files), jobID, "STARTING", jobFolder)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Status != "RUNNING" {
		t.Fatalf("recorded %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// A row that moved on meanwhile (another poll recorded it) is not reported as
// changed by this call.
func TestReconcileJobReportsNothingWhenTheRowMovedOn(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(finishSQL).WithArgs("COMPLETED", "", jobLogKey(), jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	files := jobFiles(`{"status":"COMPLETED","job_id":"` + jobID + `","log_s3_key":"` + jobLogKey() + `"}`)

	got, err := generic.ReconcileJob(context.Background(), mock, store(files), jobID, "RUNNING", jobFolder)
	if err != nil || got != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// Each of these leaves the row as it is, with no write.
func TestReconcileJobLeavesTheRowWhenTheFileIsMissingOrBad(t *testing.T) {
	otherLog := strangeOrg + "/" + ownProject + "/" + ownDiagram + "/playbooks/logs/" + jobID + ".log"
	cases := map[string]struct {
		files   *statusFiles
		status  string
		wantErr bool
	}{
		"missing":                   {&statusFiles{objects: map[string]string{}}, "RUNNING", false},
		"job already finished":      {jobFiles(`{"status":"FAILED","job_id":"` + jobID + `"}`), "COMPLETED", false},
		"running file, running row": {jobFiles(`{"status":"RUNNING","job_id":"` + jobID + `"}`), "RUNNING", false},
		"not JSON":                  {jobFiles(`COMPLETED`), "RUNNING", true},
		"unknown status":            {jobFiles(`{"status":"SUCCESS","job_id":"` + jobID + `"}`), "RUNNING", true},
		"queued status":             {jobFiles(`{"status":"QUEUED","job_id":"` + jobID + `"}`), "STARTING", true},
		"another job":               {jobFiles(`{"status":"COMPLETED","job_id":"` + otherJobID + `"}`), "RUNNING", true},
		"no job id":                 {jobFiles(`{"status":"COMPLETED"}`), "RUNNING", true},
		"another tenant's log":      {jobFiles(`{"status":"COMPLETED","job_id":"` + jobID + `","log_s3_key":"` + otherLog + `"}`), "RUNNING", true},
		"a path in the log key":     {jobFiles(`{"status":"FAILED","job_id":"` + jobID + `","log_s3_key":"../../x.log"}`), "RUNNING", true},
		"log while running":         {jobFiles(`{"status":"RUNNING","job_id":"` + jobID + `","log_s3_key":"` + jobLogKey() + `"}`), "STARTING", true},
		"error without failure":     {jobFiles(`{"status":"COMPLETED","job_id":"` + jobID + `","error_message":"x"}`), "RUNNING", true},
		"long error":                {jobFiles(`{"status":"FAILED","job_id":"` + jobID + `","error_message":"` + strings.Repeat("x", 2001) + `"}`), "RUNNING", true},
		"too large":                 {jobFiles(`{"status":"FAILED","job_id":"` + jobID + `","pad":"` + strings.Repeat("x", 8192) + `"}`), "RUNNING", true},
		"S3 error":                  {&statusFiles{err: errors.New("access denied")}, "RUNNING", true},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			mock, err := pgxmock.NewConn()
			if err != nil {
				t.Fatal(err)
			}
			conn := &writeCounter{PgxConnIface: mock}

			got, err := generic.ReconcileJob(context.Background(), conn, store(c.files), jobID, c.status, jobFolder)
			if (err != nil) != c.wantErr {
				t.Fatalf("error %v, want error %v", err, c.wantErr)
			}
			if got != nil {
				t.Fatalf("recorded %+v", got)
			}
			if conn.writes != 0 {
				t.Fatalf("%d writes, want none", conn.writes)
			}
		})
	}
}
