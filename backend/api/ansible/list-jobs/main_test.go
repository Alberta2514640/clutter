package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	userID   = "44444444-4444-4444-8444-444444444444"
	orgID    = "33333333-3333-4333-8333-333333333333"
	otherOrg = "55555555-5555-4555-8555-555555555555"
)

func useMock(t *testing.T) pgxmock.PgxConnIface {
	t.Helper()
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	connect = func() (generic.Conn, error) { return mock, nil }
	t.Cleanup(func() { connect = generic.PsqlConnectConn })
	return mock
}

func list(t *testing.T, query map[string]string) events.APIGatewayProxyResponse {
	t.Helper()
	res, err := handler(context.Background(), events.APIGatewayProxyRequest{
		QueryStringParameters: query,
		RequestContext:        events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestListJobsRefusesAnotherOrganization(t *testing.T) {
	mock := useMock(t)
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(otherOrg).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(otherOrg, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	res := list(t, map[string]string{"orgId": otherOrg})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	// The jobs table is never read.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListJobsIsScopedToTheOrganization(t *testing.T) {
	mock := useMock(t)
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(orgID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(orgID, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`WHERE created_by = \$1\s+AND split_part\(COALESCE\(NULLIF\(playbook_s3_key, ''\), terraform_directory\), '/', 1\) = \$2`).
		WithArgs(userID, orgID).
		WillReturnRows(pgxmock.NewRows([]string{"id"}))

	res := list(t, map[string]string{"orgId": orgID})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// oneFile is a bucket holding one status file.
type oneFile struct{ key, body string }

func (f oneFile) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if aws.ToString(in.Key) != f.key {
		return nil, &s3types.NoSuchKey{}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(f.body))}, nil
}

func TestListJobsShowsTheRunnersStatus(t *testing.T) {
	const (
		projectID = "11111111-1111-4111-8111-111111111111"
		diagramID = "22222222-2222-4222-8222-222222222222"
		running   = "66666666-6666-4666-8666-666666666666"
		done      = "77777777-7777-4777-8777-777777777777"
	)
	folder := orgID + "/" + projectID + "/" + diagramID + "/playbooks/"
	logKey := folder + "logs/" + running + ".log"
	file := oneFile{
		key:  folder + "logs/" + running + ".status.json",
		body: `{"status":"FAILED","job_id":"` + running + `","error_message":"Playbook failed on one or more instances","log_s3_key":"` + logKey + `"}`,
	}
	newStatusStore = func(context.Context) *generic.StatusStore {
		return &generic.StatusStore{Reader: file, Bucket: "clutter"}
	}
	t.Cleanup(func() { newStatusStore = generic.StatusStoreFromEnv })

	for _, filter := range []string{"", "RUNNING"} {
		t.Run("filter "+filter, func(t *testing.T) {
			mock := useMock(t)
			mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(orgID).
				WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
			mock.ExpectQuery(`FROM organization_members`).WithArgs(orgID, userID).
				WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
			now := time.Now()
			rows := pgxmock.NewRows([]string{"id", "job_type", "status", "created_at", "updated_at", "extra_vars", "task_arn", "error_message", "target_instance_ids", "playbook_s3_key", "terraform_directory", "role_arn"}).
				AddRow(running, "ansible", "RUNNING", now, now, []byte("{}"), nil, nil, []byte(`["i-0123456789abcdef0"]`), ptr(folder+"a-site.yml"), nil, nil)
			if filter == "" {
				rows.AddRow(done, "ansible", "COMPLETED", now, now, []byte("{}"), nil, nil, []byte(`["i-0123456789abcdef0"]`), ptr(folder+"a-site.yml"), nil, nil)
			}
			query := mock.ExpectQuery(`FROM jobs`).WithArgs(userID, orgID)
			if filter != "" {
				query = query.WithArgs(userID, orgID, filter)
			}
			query.WillReturnRows(rows)
			mock.ExpectExec(`UPDATE jobs\s+SET status = \$1`).
				WithArgs("FAILED", "Playbook failed on one or more instances", logKey, running).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))

			query2 := map[string]string{"orgId": orgID}
			if filter != "" {
				query2["status"] = filter
			}
			res := list(t, query2)
			if res.StatusCode != http.StatusOK {
				t.Fatalf("status %d: %s", res.StatusCode, res.Body)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			var body struct {
				Data []map[string]any `json:"data"`
			}
			if err := json.Unmarshal([]byte(res.Body), &body); err != nil {
				t.Fatal(err)
			}
			if filter != "" {
				// The job finished, so it no longer matches RUNNING.
				if len(body.Data) != 0 {
					t.Fatalf("listed %v", body.Data)
				}
				return
			}
			if len(body.Data) != 2 || body.Data[0]["status"] != "FAILED" || body.Data[0]["error_message"] != "Playbook failed on one or more instances" || body.Data[1]["status"] != "COMPLETED" {
				t.Fatalf("listed %v", body.Data)
			}
		})
	}
}

func ptr(s string) *string { return &s }

func TestListJobsNeedsAnOrganization(t *testing.T) {
	connect = func() (generic.Conn, error) {
		t.Error("the handler opened the database")
		return nil, context.Canceled
	}
	t.Cleanup(func() { connect = generic.PsqlConnectConn })

	res := list(t, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
}
