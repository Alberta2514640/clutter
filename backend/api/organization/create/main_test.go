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

const userID = "44444444-4444-4444-8444-444444444444"

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

func createRequest(t *testing.T, name string) events.APIGatewayProxyRequest {
	t.Helper()
	body, err := json.Marshal(generic.OrgRequestBody{OrganizationName: name})
	if err != nil {
		t.Fatal(err)
	}
	return events.APIGatewayProxyRequest{
		Body:           string(body),
		RequestContext: events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	}
}

// A name outside 4 to 32 characters is a 400 with a plain message, and never
// reaches the database (the mock expects no call).
func TestCreateRefusesANameOutsideTheLimits(t *testing.T) {
	for name, value := range map[string]string{
		"too short":            "abc",
		"short once trimmed":   "  ab  ",
		"too long":             strings.Repeat("a", 33),
		"too long in letters":  strings.Repeat("é", 33),
		"empty":                "",
		"spaces around 32 max": " " + strings.Repeat("a", 32),
	} {
		t.Run(name, func(t *testing.T) {
			mockConn(t)
			res, err := handler(createRequest(t, value))
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusBadRequest || !strings.Contains(res.Body, generic.OrganizationNameProblem) {
				t.Fatalf("status %d: %s", res.StatusCode, res.Body)
			}
		})
	}
}

// Names at the limits pass the check and reach the insert. The insert fails
// here as a duplicate, so the test needs no overview rows.
func TestCreateAcceptsANameAtTheLimits(t *testing.T) {
	for name, value := range map[string]string{
		"4 characters":         "abcd",
		"32 characters":        strings.Repeat("a", 32),
		"32 letters, 64 bytes": strings.Repeat("é", 32),
	} {
		t.Run(name, func(t *testing.T) {
			mock := mockConn(t)
			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO organizations`).
				WithArgs(pgxmock.AnyArg(), userID, value, "").
				WillReturnError(&pgconn.PgError{Code: "23505"})
			mock.ExpectRollback()
			res, err := handler(createRequest(t, value))
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusConflict {
				t.Fatalf("status %d: %s", res.StatusCode, res.Body)
			}
		})
	}
}
