package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	userID    = "44444444-4444-4444-8444-444444444444"
	orgID     = "33333333-3333-4333-8333-333333333333"
	otherOrg  = "55555555-5555-4555-8555-555555555555"
	projectID = "11111111-1111-4111-8111-111111111111"
	diagramID = "22222222-2222-4222-8222-222222222222"
)

func mockConn(t *testing.T) pgxmock.PgxConnIface {
	t.Helper()
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	connect = func() (generic.Conn, error) { return mock, nil }
	t.Cleanup(func() {
		connect = generic.PsqlConnectConn
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	return mock
}

func expectDiagramIn(mock pgxmock.PgxConnIface, org string, member bool) {
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(diagramID).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(projectID).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(org))
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(org).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(org, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(member))
}

func deleteRequest() events.APIGatewayProxyRequest {
	return events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{"projectId": projectID, "diagramId": diagramID},
		RequestContext:        events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	}
}

func TestDiagramDeleteRefusesAnotherOrganizationsDiagram(t *testing.T) {
	mock := mockConn(t)
	expectDiagramIn(mock, otherOrg, false)

	res, err := handler(context.Background(), deleteRequest())
	if err != nil {
		t.Fatal(err)
	}
	// Nothing is deleted.
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
}

func TestDiagramDeleteWithLiveDeploymentsIsAConflict(t *testing.T) {
	mock := mockConn(t)
	expectDiagramIn(mock, orgID, true)
	mock.ExpectExec(`DELETE FROM diagrams`).WithArgs(diagramID, projectID).
		WillReturnError(&pgconn.PgError{Code: "23001", Message: "deployment 99 is live; destroy it before deleting"})

	res, err := handler(context.Background(), deleteRequest())
	if err != nil {
		t.Fatal(err)
	}
	var body struct{ Message string }
	_ = json.Unmarshal([]byte(res.Body), &body)
	if res.StatusCode != http.StatusConflict || body.Message != "this has live deployments; destroy them first" {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
}
