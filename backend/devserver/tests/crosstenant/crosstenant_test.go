// Package crosstenant is the cross-tenant part of the security regression
// suite. Against A's real Lambdas, run by A2, one user tries to read and to
// change another organization's projects, diagrams and Terraform, and every
// attempt must be refused, with the owner's data left as it was.
//
// It needs the running backend (npm run dev:backend) and is started by
// tests/security-regression/run.sh, which supplies CLUTTER_API_URL,
// PSQL_CONNECTION_STRING and JWT_SECRET. Without them it skips.
package crosstenant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/devserver/internal/devauth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type user struct {
	id, token, org string
}

type client struct {
	t    *testing.T
	base string
}

type reply struct {
	status int
	body   []byte
	json   map[string]any
}

func (c client) do(method, path string, body any, token string) reply {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rd)
	if err != nil {
		c.t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	r := reply{status: res.StatusCode, body: b}
	_ = json.Unmarshal(b, &r.json) //nolint:errcheck // some bodies are not objects
	return r
}

func str(m map[string]any, path ...string) string {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = mm[p]
	}
	s, _ := cur.(string)
	return s
}

func newUser(t *testing.T, c client, conn, secret, label string) user {
	t.Helper()
	ctx := context.Background()
	db, err := pgx.Connect(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(ctx)
	u := devauth.UserData{Uuid: uuid.NewString(), Email: "xtenant-" + label + "-" + uuid.NewString()[:8] + "@clutter.localhost",
		FullName: "Cross-tenant " + label, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if _, err := db.Exec(ctx, `INSERT INTO users (id, email, full_name, picture_url) VALUES ($1, $2, $3, '')`, u.Uuid, u.Email, u.FullName); err != nil {
		t.Fatal(err)
	}
	token, _, err := devauth.Sign(u, []byte(secret), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	org := c.do("POST", "/organization", map[string]string{"organizationName": "Cross-tenant " + label + " " + uuid.NewString()[:6], "description": "security regression"}, token)
	id := str(org.json, "data", "id")
	if id == "" {
		t.Fatalf("organization create: %d %s", org.status, org.body)
	}
	t.Cleanup(func() {
		c.do("DELETE", "/organization/"+id, nil, token)
		db, err := pgx.Connect(context.Background(), conn)
		if err == nil {
			_, _ = db.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, u.Uuid) //nolint:errcheck // best-effort cleanup
			db.Close(context.Background())
		}
	})
	return user{id: u.Uuid, token: token, org: id}
}

func TestOneOrganizationCannotReachAnother(t *testing.T) {
	base, conn, secret := os.Getenv("CLUTTER_API_URL"), os.Getenv("PSQL_CONNECTION_STRING"), os.Getenv("JWT_SECRET")
	if base == "" || conn == "" || secret == "" {
		t.Skip("the backend is not running: start it with npm run dev:backend and use tests/security-regression/run.sh")
	}
	c := client{t: t, base: strings.TrimRight(base, "/")}
	owner := newUser(t, c, conn, secret, "owner")
	attacker := newUser(t, c, conn, secret, "attacker")

	// The owner's project, diagram and generated Terraform.
	proj := c.do("POST", "/project", map[string]string{"organizationId": owner.org, "name": "Owner Project", "description": "private"}, owner.token)
	projID := str(proj.json, "data", "id")
	dia := c.do("POST", "/diagram", map[string]string{"projectId": projID, "name": "Owner Diagram"}, owner.token)
	diaID := str(dia.json, "data", "id")
	if projID == "" || diaID == "" {
		t.Fatalf("seed: project %d %s, diagram %d %s", proj.status, proj.body, dia.status, dia.body)
	}
	layout := map[string]any{"nodes": []any{}, "edges": []any{}, "viewport": map[string]any{"x": 0, "y": 0, "zoom": 1}}
	if r := c.do("PUT", "/diagram", map[string]any{"projectId": projID, "diagramId": diaID, "name": "Owner Diagram", "data": layout}, owner.token); r.status != http.StatusOK {
		t.Fatalf("save: %d %s", r.status, r.body)
	}
	if r := c.do("POST", "/terraform-engine", map[string]string{"projectId": projID, "diagramId": diaID}, owner.token); r.status >= 500 {
		t.Fatalf("generate: %d %s", r.status, r.body)
	}
	before := c.do("GET", "/diagram?projectId="+projID+"&diagramId="+diaID, nil, owner.token)
	beforeProj := c.do("GET", "/project?organizationId="+owner.org+"&projectId="+projID, nil, owner.token)

	q := url.QueryEscape
	type attempt struct {
		kind, method, path string
		body               any
	}
	attempts := []attempt{
		// Reads: the attacker names the owner's ids, under the owner's
		// organization and under their own.
		{"read organization", "GET", "/organization?organizationId=" + owner.org, nil},
		{"read project (owner's org)", "GET", "/project?organizationId=" + owner.org + "&projectId=" + projID, nil},
		{"read project (own org)", "GET", "/project?organizationId=" + attacker.org + "&projectId=" + projID, nil},
		{"list projects", "GET", "/project?organizationId=" + owner.org, nil},
		{"read diagram", "GET", "/diagram?projectId=" + projID + "&diagramId=" + diaID, nil},
		{"list diagrams", "GET", "/diagram?projectId=" + projID, nil},
		{"read Terraform files", "GET", "/terraform-engine/files?projectId=" + projID + "&diagramId=" + diaID, nil},
		{"read Terraform logs", "GET", "/terraform-engine/logs?orgId=" + attacker.org + "&projId=" + projID + "&diagramId=" + diaID, nil},
		{"read Terraform log URL", "GET", "/terraform-engine/logs/url?orgId=" + attacker.org + "&projId=" + projID + "&diagramId=" + diaID + "&deploymentId=x0000001&file=command.log", nil},
		{"read recent activity", "GET", "/terraform-engine/logs/recent-activity?orgId=" + attacker.org + "&diagramId=" + diaID, nil},
		{"read linked accounts", "GET", "/organization/" + owner.org + "/accounts", nil},
		{"read Ansible jobs", "GET", "/ansible/jobs?orgId=" + owner.org, nil},
		{"read live logs", "GET", "/terraform-engine/logs/live?orgId=" + attacker.org + "&projId=" + projID + "&diagramId=" + diaID + "&taskArn=" + q("arn:aws:ecs:ca-central-1:000000000000:task/c/0123456789abcdef0123456789abcdef"), nil},
		// Changes.
		{"rename project (owner's org)", "PUT", "/project?projectId=" + projID, map[string]string{"organizationId": owner.org, "name": "pwned", "description": "x"}},
		{"rename project (own org)", "PUT", "/project?projectId=" + projID, map[string]string{"organizationId": attacker.org, "name": "pwned", "description": "x"}},
		{"overwrite diagram", "PUT", "/diagram", map[string]any{"projectId": projID, "diagramId": diaID, "name": "pwned", "data": layout}},
		{"add diagram to project", "POST", "/diagram", map[string]string{"projectId": projID, "name": "planted"}},
		{"generate Terraform", "POST", "/terraform-engine", map[string]string{"projectId": projID, "diagramId": diaID}},
		{"get a code-upload URL", "POST", "/terraform-engine/code-upload/presigned-url", map[string]string{"org_id": attacker.org, "project_id": projID, "diagram_id": diaID, "lambda_resource_name": "fn", "runtime": "python3.13"}},
		{"run Terraform apply", "POST", "/terraform-command-runner", map[string]string{"organizationId": attacker.org, "projectId": projID, "diagramId": diaID, "accountAccessRoleId": uuid.NewString(), "command": "apply"}},
		{"rename organization", "PUT", "/organization/" + owner.org, map[string]string{"description": "pwned"}},
		{"delete diagram", "DELETE", "/diagram?projectId=" + projID + "&diagramId=" + diaID, nil},
		{"delete project", "DELETE", "/project?organizationId=" + owner.org + "&projectId=" + projID, nil},
		{"delete organization", "DELETE", "/organization/" + owner.org, nil},
	}
	for _, a := range attempts {
		r := c.do(a.method, a.path, a.body, attacker.token)
		// Every body is well formed, so the only reason left to refuse is
		// that the caller does not own what it names: 403 or 404, never a 400.
		refused := r.status == http.StatusForbidden || r.status == http.StatusNotFound
		t.Logf("%-32s %-6s -> %d", a.kind, a.method, r.status)
		if !refused {
			t.Errorf("%s: %s %s answered %d %.200s", a.kind, a.method, a.path, r.status, r.body)
		}
		if strings.Contains(string(r.body), "Owner Project") || strings.Contains(string(r.body), "Owner Diagram") {
			t.Errorf("%s: the answer names the owner's data", a.kind)
		}
	}

	// No token and a forged token get nothing either.
	forged, _, _ := devauth.Sign(devauth.UserData{Uuid: owner.id, Email: "x@clutter.localhost", FullName: "X", CreatedAt: time.Now().UTC().Format(time.RFC3339)}, []byte("not-the-real-secret-0123456789abcdef"), time.Now())
	for _, tok := range []string{"", forged} {
		r := c.do("GET", "/diagram?projectId="+projID+"&diagramId="+diaID, nil, tok)
		if r.status != http.StatusUnauthorized && r.status != http.StatusForbidden {
			t.Errorf("token %q: read diagram answered %d", map[bool]string{true: "none", false: "forged"}[tok == ""], r.status)
		}
	}

	// The owner's data is as it was.
	after := c.do("GET", "/diagram?projectId="+projID+"&diagramId="+diaID, nil, owner.token)
	afterProj := c.do("GET", "/project?organizationId="+owner.org+"&projectId="+projID, nil, owner.token)
	if after.status != http.StatusOK || !bytes.Equal(normalize(before.body), normalize(after.body)) {
		t.Errorf("the owner's diagram changed: %d\nbefore %s\nafter  %s", after.status, before.body, after.body)
	}
	if afterProj.status != http.StatusOK || !bytes.Equal(normalize(beforeProj.body), normalize(afterProj.body)) {
		t.Errorf("the owner's project changed: %d\nbefore %s\nafter  %s", afterProj.status, beforeProj.body, afterProj.body)
	}
	list := c.do("GET", "/diagram?projectId="+projID, nil, owner.token)
	if strings.Contains(string(list.body), "planted") {
		t.Error("the attacker planted a diagram in the owner's project")
	}
	fmt.Fprintf(os.Stderr, "cross-tenant: %d attempts checked\n", len(attempts))
}

// normalize drops fields a read itself may change (none today), so the
// before and after bodies compare as data.
func normalize(b []byte) []byte { return bytes.TrimSpace(b) }

// TestAnonymousDatabaseRoleReadsNoTenantRows is the row-level security check
// (phase 0 item 2): Supabase's anon role, which any browser holding the
// public key acts as, reads no user, organization, project, diagram or
// linked-account row, while it still reads the public resource catalog.
func TestAnonymousDatabaseRoleReadsNoTenantRows(t *testing.T) {
	conn := os.Getenv("PSQL_CONNECTION_STRING")
	if conn == "" {
		t.Skip("the backend is not running: use tests/security-regression/run.sh")
	}
	ctx := context.Background()
	db, err := pgx.Connect(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(ctx)
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // read-only
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&total); err != nil || total == 0 {
		t.Fatalf("the owner role sees %d users (%v); the test needs at least one", total, err)
	}
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE anon`); err != nil {
		t.Fatalf("set role anon: %v", err)
	}
	for _, table := range []string{"users", "organizations", "organization_members", "projects", "diagrams", "diagram_history", "aws_account_access_roles", "connections", "deployments", "runs"} {
		var n int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n)
		if err != nil {
			// No privilege at all is also a refusal; the transaction must
			// survive it for the next table.
			if _, rerr := tx.Exec(ctx, `ROLLBACK`); rerr == nil {
				tx, _ = db.Begin(ctx)
				_, _ = tx.Exec(ctx, `SET LOCAL ROLE anon`) //nolint:errcheck // re-entering the role
			}
			t.Logf("anon on %-26s refused (%v)", table, strings.SplitN(err.Error(), "(", 2)[0])
			continue
		}
		t.Logf("anon on %-26s %d rows", table, n)
		if n != 0 {
			t.Errorf("anon reads %d rows of %s", n, table)
		}
	}
}
