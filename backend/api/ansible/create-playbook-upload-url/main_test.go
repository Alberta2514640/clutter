package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	userID    = "44444444-4444-4444-8444-444444444444"
	otherOrg  = "55555555-5555-4555-8555-555555555555"
	projectID = "11111111-1111-4111-8111-111111111111"
	diagramID = "22222222-2222-4222-8222-222222222222"
)

func TestPlaybookUploadRefusesAnotherOrganizationsDiagram(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	connect = func() (generic.Conn, error) { return mock, nil }
	t.Cleanup(func() { connect = generic.PsqlConnectConn })

	// The diagram and project exist, but in an organization the caller is not in.
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(diagramID).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(projectID).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(otherOrg))
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(otherOrg).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(otherOrg, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	raw, _ := json.Marshal(map[string]string{
		"file_name": "site.yml", "project_id": projectID, "diagram_id": diagramID,
	})
	res, err := handler(context.Background(), events.APIGatewayProxyRequest{
		Body:           string(raw),
		RequestContext: events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	// No playbook row is written.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
