package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/pgtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestConnectionsComeOnlyFromLinkedRoles: the external id is in the stack
// link every member of the organization sees, so B makes an AWS connection
// only from a role linked to the caller's organization (ADR-019).
func TestConnectionsComeOnlyFromLinkedRoles(t *testing.T) {
	c := New(nil, nil, nil)
	if !c.Credentials().LinkedOnly {
		t.Fatal("the AWS connector accepts pasted credentials")
	}
	var _ connectors.Linker = c
}

// TestLinkedRolesReadsOnlyTheOrganizationsCompletedRoles runs against a
// throwaway Postgres with A's schema. Without Docker it skips.
func TestLinkedRolesReadsOnlyTheOrganizationsCompletedRoles(t *testing.T) {
	db, err := pgtest.Start()
	if errors.Is(err, pgtest.ErrNoDocker) {
		t.Skip("docker is not available")
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Stop)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, db.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	user, orgA, orgB := uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO public.users (id, email, full_name, picture_url) VALUES ($1, 'a@example.com', 'A', '')`, user)
	for _, org := range []string{orgA, orgB} {
		exec(`INSERT INTO public.organizations (id, name, created_by) VALUES ($1, $2, $3)`, org, "org-"+org[:8], user)
	}
	externals := map[string]string{}
	role := func(org, name, status string, arn any) string {
		id := uuid.NewString()
		externals[id] = uuid.NewString() // A's external id is a UUID
		exec(`INSERT INTO public.aws_account_access_roles (id, organization_id, unique_id, account_name, role_arn, external_id, status, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, id, org, id[:8], name, arn, externals[id], status, user)
		return id
	}
	done := role(orgA, "prod", "complete", goodRole)
	pending := role(orgA, "half", "incomplete", nil)
	revoked := role(orgA, "old", "revoked", goodRole)
	other := role(orgB, "theirs", "complete", goodRole)

	l := LinkedRoles{DB: pool}
	got, err := l.List(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != done || got[0].RoleARN != goodRole || got[0].AccountName != "prod" {
		t.Fatalf("listed %+v", got)
	}

	creds, err := l.Credentials(ctx, orgA, done)
	if err != nil {
		t.Fatal(err)
	}
	if creds[CredRoleARN] != goodRole || creds[CredExternalID] != externals[done] {
		t.Fatal("wrong credentials")
	}
	if err := (&Connector{}).Credentials().CheckSupplied(creds); err != nil {
		t.Fatalf("a linked role does not fit the credential spec: %v", err)
	}
	for name, id := range map[string]string{"incomplete": pending, "revoked": revoked, "another organization's": other, "not a uuid": "x"} {
		if _, err := l.Credentials(ctx, orgA, id); !errors.Is(err, ErrLinkedRoleNotFound) {
			t.Errorf("%s role: %v", name, err)
		}
	}

	// The connector offers the same roles through connectors.Linker, by name only.
	c := New(nil, nil, nil).WithLinkedRoles(l)
	accounts, err := c.Linked(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].ID != done || accounts[0].Name != "prod" {
		t.Fatalf("linked accounts %+v", accounts)
	}
	linked, err := c.LinkedSecrets(ctx, orgA, done)
	if err != nil || linked.Secrets[CredExternalID] != externals[done] {
		t.Fatalf("linked secrets: %v", err)
	}
	// A's link flow does not record the stack's template version.
	if linked.TemplateVersion != "" {
		t.Fatalf("template version %q from a row that does not record one", linked.TemplateVersion)
	}
	if _, err := c.LinkedSecrets(ctx, orgB, done); !errors.Is(err, connectors.ErrLinkedNotFound) {
		t.Errorf("another organization's role through the connector: %v", err)
	}
	if accounts, err := New(nil, nil, nil).Linked(ctx, orgA); err != nil || len(accounts) != 0 {
		t.Errorf("without linked roles: %v %v", accounts, err)
	}
	if _, err := New(nil, nil, nil).LinkedSecrets(ctx, orgA, done); !errors.Is(err, connectors.ErrLinkedNotFound) {
		t.Errorf("secrets without linked roles: %v", err)
	}
}
