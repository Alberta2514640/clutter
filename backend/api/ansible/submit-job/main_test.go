package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	userID    = "44444444-4444-4444-8444-444444444444"
	orgID     = "33333333-3333-4333-8333-333333333333"
	otherOrg  = "55555555-5555-4555-8555-555555555555"
	projectID = "11111111-1111-4111-8111-111111111111"
	diagramID = "22222222-2222-4222-8222-222222222222"
	accountID = "77777777-7777-4777-8777-777777777777"
)

func TestSubmitJobRefusesAPlaybookKeyNamingAnotherOrganizationsDiagram(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	original := connect
	connect = func() (dbConn, error) { return mock, nil }
	t.Cleanup(func() { connect = original })
	t.Setenv("JOB_QUEUE_URL", "https://example.invalid/queue")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")

	// The key starts with the caller's own organization, but the project and
	// diagram after it are another organization's.
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(orgID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(orgID, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(diagramID).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(projectID).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(otherOrg))

	raw, _ := json.Marshal(map[string]any{
		"account_access_role_id": accountID,
		"target_instance_ids":    []string{"i-0123456789abcdef0"},
		"playbook_s3_key":        orgID + "/" + projectID + "/" + diagramID + "/playbooks/site.yml",
	})
	res, err := handler(context.Background(), events.APIGatewayProxyRequest{
		Body:           string(raw),
		RequestContext: events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Refused before S3 is read, the account is looked up or a job is written.
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
