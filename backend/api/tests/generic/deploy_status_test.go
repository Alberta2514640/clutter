package generic_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

const deployBucket = "clutter-bucket"

var deployIDs = generic.OwnedIDs{OrgID: ownOrg, ProjectID: ownProject, DiagramID: ownDiagram}

// statusFiles answers GetObject from memory, like S3 for one bucket.
type statusFiles struct {
	objects map[string]string
	err     error
	reads   []string
}

func (f *statusFiles) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	key := aws.ToString(in.Key)
	f.reads = append(f.reads, aws.ToString(in.Bucket)+"/"+key)
	if f.err != nil {
		return nil, f.err
	}
	body, ok := f.objects[key]
	if !ok {
		return nil, &types.NoSuchKey{}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(body))}, nil
}

func store(files *statusFiles) generic.StatusStore {
	return generic.StatusStore{Reader: files, Bucket: deployBucket}
}

func statusKey(commandID string) string {
	return ownOrg + "/" + ownProject + "/" + ownDiagram + "/terraform/logs/" + commandID + "/status.json"
}

func expectRunning(mock pgxmock.PgxConnIface, commandIDs ...string) {
	rows := pgxmock.NewRows([]string{"command_id"})
	for _, id := range commandIDs {
		rows.AddRow(id)
	}
	mock.ExpectQuery(`SELECT command_id\s+FROM public.diagram_deployment_logs\s+WHERE diagram_id = \$1 AND status = 'RUNNING'`).
		WithArgs(ownDiagram, pgxmock.AnyArg()).
		WillReturnRows(rows)
}

func expectRecorded(mock pgxmock.PgxConnIface, status string, seconds int, commandID string) {
	mock.ExpectExec(`UPDATE public.diagram_deployment_logs\s+SET status = \$1, duration_seconds = \$2\s+WHERE diagram_id = \$3 AND command_id = \$4 AND status = 'RUNNING'`).
		WithArgs(status, seconds, ownDiagram, commandID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

func TestReconcileRecordsAFinishedRun(t *testing.T) {
	for _, status := range []string{"SUCCESS", "FAILED"} {
		t.Run(status, func(t *testing.T) {
			mock, err := pgxmock.NewConn()
			if err != nil {
				t.Fatal(err)
			}
			files := &statusFiles{objects: map[string]string{
				statusKey("abcd1234"): `{"status":"` + status + `","command_id":"abcd1234","duration_seconds":95}`,
			}}
			expectRunning(mock, "abcd1234")
			expectRecorded(mock, status, 95, "abcd1234")

			if err := generic.ReconcileDeployRuns(context.Background(), mock, store(files), deployIDs); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			if len(files.reads) != 1 || files.reads[0] != deployBucket+"/"+statusKey("abcd1234") {
				t.Fatalf("read %v", files.reads)
			}
		})
	}
}

// writeCounter counts every Exec, expected or not. pgxmock answers an
// unexpected Exec with an error, which the helper only logs, so the count is
// what proves no write was sent.
type writeCounter struct {
	pgxmock.PgxConnIface
	writes int
}

func (w *writeCounter) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	w.writes++
	return w.PgxConnIface.Exec(ctx, sql, args...)
}

// Each of these leaves the row RUNNING.
func TestReconcileLeavesTheRowWhenTheFileIsMissingOrBad(t *testing.T) {
	cases := map[string]*statusFiles{
		"missing":          {objects: map[string]string{}},
		"not JSON":         {objects: map[string]string{statusKey("abcd1234"): `SUCCESS`}},
		"unknown status":   {objects: map[string]string{statusKey("abcd1234"): `{"status":"RUNNING","command_id":"abcd1234","duration_seconds":1}`}},
		"lowercase status": {objects: map[string]string{statusKey("abcd1234"): `{"status":"success","command_id":"abcd1234","duration_seconds":1}`}},
		"another run":      {objects: map[string]string{statusKey("abcd1234"): `{"status":"SUCCESS","command_id":"zzzz9999","duration_seconds":1}`}},
		"no command id":    {objects: map[string]string{statusKey("abcd1234"): `{"status":"SUCCESS","duration_seconds":1}`}},
		"negative time":    {objects: map[string]string{statusKey("abcd1234"): `{"status":"FAILED","command_id":"abcd1234","duration_seconds":-5}`}},
		"string duration":  {objects: map[string]string{statusKey("abcd1234"): `{"status":"FAILED","command_id":"abcd1234","duration_seconds":"5; DROP TABLE"}`}},
		"too large":        {objects: map[string]string{statusKey("abcd1234"): `{"status":"SUCCESS","command_id":"abcd1234","duration_seconds":1,"pad":"` + strings.Repeat("x", 8192) + `"}`}},
		"S3 error":         {err: errors.New("access denied")},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			mock, err := pgxmock.NewConn()
			if err != nil {
				t.Fatal(err)
			}
			expectRunning(mock, "abcd1234")
			conn := &writeCounter{PgxConnIface: mock}

			if err := generic.ReconcileDeployRuns(context.Background(), conn, store(files), deployIDs); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			if conn.writes != 0 {
				t.Fatalf("%d writes, want none", conn.writes)
			}
			if len(files.reads) != 1 {
				t.Fatalf("read %v", files.reads)
			}
		})
	}
}

// A bad file for one run does not stop the next run's good file.
func TestReconcileGoesOnPastABadFile(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	files := &statusFiles{objects: map[string]string{
		statusKey("aaaa1111"): `{`,
		statusKey("bbbb2222"): `{"status":"FAILED","command_id":"bbbb2222","duration_seconds":7}`,
	}}
	expectRunning(mock, "aaaa1111", "cccc3333", "bbbb2222")
	expectRecorded(mock, "FAILED", 7, "bbbb2222")
	conn := &writeCounter{PgxConnIface: mock}

	if err := generic.ReconcileDeployRuns(context.Background(), conn, store(files), deployIDs); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if conn.writes != 1 {
		t.Fatalf("%d writes, want 1", conn.writes)
	}
	if len(files.reads) != 3 {
		t.Fatalf("read %v", files.reads)
	}
}

func TestReconcileReturnsAFailedRowList(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT command_id`).WillReturnError(errors.New("connection reset"))
	files := &statusFiles{}

	if err := generic.ReconcileDeployRuns(context.Background(), mock, store(files), deployIDs); err == nil {
		t.Fatal("want an error")
	}
	if len(files.reads) != 0 {
		t.Fatalf("read %v", files.reads)
	}
}
