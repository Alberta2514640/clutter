package generic_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	ownUser    = "44444444-4444-4444-8444-444444444444"
	ownOrg     = "33333333-3333-4333-8333-333333333333"
	strangeOrg = "55555555-5555-4555-8555-555555555555"
	ownProject = "11111111-1111-4111-8111-111111111111"
	ownDiagram = "22222222-2222-4222-8222-222222222222"
)

func ownershipMock(t *testing.T) pgxmock.PgxConnIface {
	t.Helper()
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	return mock
}

func expectMember(mock pgxmock.PgxConnIface, org string, member bool) {
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(org).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(org, ownUser).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(member))
}

func expectDiagramIn(mock pgxmock.PgxConnIface, project string) {
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(ownDiagram).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(project))
}

func expectProjectIn(mock pgxmock.PgxConnIface, org string) {
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(ownProject).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(org))
}

func wantStatus(t *testing.T, err error, status int) {
	t.Helper()
	var authErr *generic.AuthorizationError
	if !errors.As(err, &authErr) || authErr.StatusCode != status {
		t.Fatalf("want a %d AuthorizationError, got %v", status, err)
	}
	// One message per status, holding no id, so the answer never tells a
	// caller which of another tenant's ids exist.
	want := map[int]string{
		404: "not found",
		403: "user does not have access to this organization",
	}[status]
	if authErr.Message != want {
		t.Fatalf("message %q, want %q", authErr.Message, want)
	}
}

func TestOwnershipAcceptsAMembersOwnDiagram(t *testing.T) {
	mock := ownershipMock(t)
	expectMember(mock, ownOrg, true)
	expectDiagramIn(mock, ownProject)
	expectProjectIn(mock, ownOrg)
	org, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, ownOrg, ownProject, ownDiagram)
	if err != nil || org != ownOrg {
		t.Fatalf("org %q, err %v", org, err)
	}
}

func TestOwnershipRefusesAProjectOfAnotherOrganization(t *testing.T) {
	mock := ownershipMock(t)
	// A member of ownOrg names their own organization and a stranger's project.
	expectMember(mock, ownOrg, true)
	expectProjectIn(mock, strangeOrg)
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, ownOrg, ownProject, "")
	wantStatus(t, err, 404)
}

func TestOwnershipRefusesADiagramOfAnotherProject(t *testing.T) {
	mock := ownershipMock(t)
	expectMember(mock, ownOrg, true)
	expectDiagramIn(mock, "99999999-9999-4999-8999-999999999999")
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, ownOrg, ownProject, ownDiagram)
	wantStatus(t, err, 404)
}

func TestOwnershipRefusesANonMemberBeforeAnyChildLookup(t *testing.T) {
	mock := ownershipMock(t)
	expectMember(mock, strangeOrg, false)
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, strangeOrg, ownProject, ownDiagram)
	wantStatus(t, err, 403)
}

func TestOwnershipFindsTheOrganizationFromTheProject(t *testing.T) {
	mock := ownershipMock(t)
	expectDiagramIn(mock, ownProject)
	expectProjectIn(mock, strangeOrg)
	expectMember(mock, strangeOrg, false)
	// The caller named no organization, so a stranger learns nothing: 404.
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, "", ownProject, ownDiagram)
	wantStatus(t, err, 404)
}

func TestOwnershipRefusesAProjectThatDoesNotExist(t *testing.T) {
	mock := ownershipMock(t)
	expectMember(mock, ownOrg, true)
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(ownProject).
		WillReturnError(pgx.ErrNoRows)
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, ownOrg, ownProject, "")
	wantStatus(t, err, 404)
}

func TestOwnershipRefusesADiagramThatDoesNotExist(t *testing.T) {
	mock := ownershipMock(t)
	expectMember(mock, ownOrg, true)
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(ownDiagram).
		WillReturnError(pgx.ErrNoRows)
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, ownOrg, ownProject, ownDiagram)
	wantStatus(t, err, 404)
}

func TestOwnershipRefusesAnOrganizationThatDoesNotExist(t *testing.T) {
	mock := ownershipMock(t)
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(strangeOrg).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, strangeOrg, ownProject, "")
	wantStatus(t, err, 404)
}

func TestMembershipMessagesNameNoID(t *testing.T) {
	mock := ownershipMock(t)
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(strangeOrg).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	expectMember(mock, ownOrg, false)
	for _, org := range []string{strangeOrg, ownOrg} {
		err := generic.CheckOrganizationMembershipPSQL(context.Background(), mock, ownUser, org)
		var authErr *generic.AuthorizationError
		if !errors.As(err, &authErr) {
			t.Fatalf("want an AuthorizationError, got %v", err)
		}
		if strings.Contains(authErr.Message, org) || strings.Contains(authErr.Message, ownUser) {
			t.Fatalf("message %q names an id", authErr.Message)
		}
	}
}

func TestOwnershipRefusesIDsThatAreNotUUIDs(t *testing.T) {
	mock := ownershipMock(t)
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, ownOrg, "../other", "")
	wantStatus(t, err, 404)
}

func TestOwnershipRefusesACallerWithNoID(t *testing.T) {
	mock := ownershipMock(t)
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, "", ownOrg, "", "")
	wantStatus(t, err, 403)
}

func TestOwnershipRefusesARequestThatNamesNothing(t *testing.T) {
	mock := ownershipMock(t)
	_, err := generic.CheckOwnershipPSQL(context.Background(), mock, ownUser, "", "", "")
	wantStatus(t, err, 403)
}
