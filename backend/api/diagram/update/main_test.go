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
	otherOrg  = "55555555-5555-4555-8555-555555555555"
	projectID = "11111111-1111-4111-8111-111111111111"
	diagramID = "22222222-2222-4222-8222-222222222222"
)

func TestDiagramUpdateGivesAnotherOrganizationsDiagramTheSameNotFound(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	original := connect
	connect = func() (dbConn, error) { return mock, nil }
	t.Cleanup(func() { connect = original })

	// The diagram and project exist, but in an organization the caller is not in.
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(diagramID).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(projectID).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(otherOrg))
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(otherOrg).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(otherOrg, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	raw, _ := json.Marshal(map[string]string{"projectId": projectID, "diagramId": diagramID, "name": "renamed"})
	res, err := handler(context.Background(), events.APIGatewayProxyRequest{
		Body:           string(raw),
		RequestContext: events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var body struct{ Message string }
	_ = json.Unmarshal([]byte(res.Body), &body)
	if res.StatusCode != http.StatusNotFound || body.Message != "not found" {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	// No transaction starts and nothing is written.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
