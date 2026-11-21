package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	userID = "44444444-4444-4444-8444-444444444444"
	orgID  = "33333333-3333-4333-8333-333333333333"
)

func TestOrganizationDeleteWithLiveDeploymentsIsAConflict(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	connect = func() (generic.Conn, error) { return mock, nil }
	t.Cleanup(func() { connect = generic.PsqlConnectConn })

	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(orgID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(orgID, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	// The cascade reaches a live deployment and the trigger refuses it.
	mock.ExpectQuery(`DELETE FROM organizations`).WithArgs(orgID).
		WillReturnError(&pgconn.PgError{Code: "23001", Message: "deployment 99 is live; destroy it before deleting"})

	res, err := handler(events.APIGatewayProxyRequest{
		PathParameters: map[string]string{"organizationId": orgID},
		RequestContext: events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var body struct{ Message string }
	_ = json.Unmarshal([]byte(res.Body), &body)
	if res.StatusCode != http.StatusConflict || body.Message != "this has live deployments; destroy them first" {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
