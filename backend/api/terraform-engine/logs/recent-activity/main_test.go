package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	userID    = "44444444-4444-4444-8444-444444444444"
	orgID     = "33333333-3333-4333-8333-333333333333"
	otherOrg  = "55555555-5555-4555-8555-555555555555"
	projectID = "11111111-1111-4111-8111-111111111111"
	diagramID = "22222222-2222-4222-8222-222222222222"
)

func TestRecentActivityRefusesAnotherOrganizationsDiagram(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	connect = func() (generic.Conn, error) { return mock, nil }
	t.Cleanup(func() { connect = generic.PsqlConnectConn })

	// The caller is a member of orgID, but the diagram's project is in otherOrg.
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(orgID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(orgID, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(diagramID).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(projectID).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(otherOrg))

	res, err := handler(context.Background(), events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{"orgId": orgID, "diagramId": diagramID},
		RequestContext:        events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	// The deployment logs are never queried.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
