package main

import (
	"encoding/json"
	"net/http"
	"strings"
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

func mockConn(t *testing.T) pgxmock.PgxConnIface {
	t.Helper()
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	saved := connect
	connect = func() (dbConn, error) { return mock, nil }
	t.Cleanup(func() {
		connect = saved
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	return mock
}

func updateRequest(t *testing.T, name, description string) events.APIGatewayProxyRequest {
	t.Helper()
	body, err := json.Marshal(generic.OrgRequestBody{OrganizationName: name, Description: description})
	if err != nil {
		t.Fatal(err)
	}
	return events.APIGatewayProxyRequest{
		Body:           string(body),
		PathParameters: map[string]string{"organizationId": orgID},
		RequestContext: events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	}
}

func expectMemberWithName(mock pgxmock.PgxConnIface, current string) {
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(orgID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(orgID, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT name, description\s+FROM organizations`).WithArgs(orgID).
		WillReturnRows(pgxmock.NewRows([]string{"name", "description"}).AddRow(current, ""))
}

// A new name outside 4 to 32 characters is a 400 with a plain message, and
// never reaches the database (the mock expects no call).
func TestUpdateRefusesANameOutsideTheLimits(t *testing.T) {
	for name, value := range map[string]string{
		"too short":          "abc",
		"short once trimmed": "  ab  ",
		"too long":           strings.Repeat("a", 33),
	} {
		t.Run(name, func(t *testing.T) {
			mockConn(t)
			res, err := handler(updateRequest(t, value, ""))
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusBadRequest || !strings.Contains(res.Body, generic.OrganizationNameProblem) {
				t.Fatalf("status %d: %s", res.StatusCode, res.Body)
			}
		})
	}
}

// A 32-character name passes the check and reaches the update, which fails
// here as a duplicate, so the test needs no overview rows.
func TestUpdateAcceptsANameAtTheLimit(t *testing.T) {
	value := strings.Repeat("é", 32)
	mock := mockConn(t)
	expectMemberWithName(mock, "Old name")
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE organizations\s+SET name`).WithArgs(value, orgID).
		WillReturnError(&pgconn.PgError{Code: "23505"})
	mock.ExpectRollback()
	res, err := handler(updateRequest(t, value, ""))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
}

// An empty name means the name does not change, so a description-only update
// is not refused by the name rule.
func TestUpdateWithoutANameSkipsTheNameRule(t *testing.T) {
	mock := mockConn(t)
	expectMemberWithName(mock, "Same name")
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE organizations\s+SET description`).WithArgs("New text", orgID).
		WillReturnError(&pgconn.PgError{Code: "57014"})
	mock.ExpectRollback()
	res, err := handler(updateRequest(t, "", "New text"))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
}
