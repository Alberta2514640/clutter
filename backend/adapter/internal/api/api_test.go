package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/auth"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/codestore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/credentials"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/jobs"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/pgtest"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	testDB    *pgtest.DB
	testDBErr error
)

func TestMain(m *testing.M) {
	testDB, testDBErr = pgtest.Start()
	if testDBErr != nil && !errors.Is(testDBErr, pgtest.ErrNoDocker) {
		fmt.Fprintln(os.Stderr, "pgtest:", testDBErr)
		os.Exit(1)
	}
	code := m.Run()
	testDB.Stop()
	os.Exit(code)
}

// fakeVerifier maps a token to a user id.
type fakeVerifier map[string]string

func (f fakeVerifier) Verify(_ context.Context, token string) (auth.Identity, error) {
	id, ok := f[token]
	if !ok {
		return auth.Identity{}, errors.New("bad token")
	}
	return auth.Identity{UserID: id}, nil
}

// tenant is one user with one organization, project and blocks diagram.
type tenant struct {
	token, user, org, project, diagram string
}

const blocksDoc = `{"engine":"blocks","nodes":[
 {"id":"n1","type":"vm","position":{"x":0,"y":0},"data":{"name":"web","fields":{"size":"small"}}},
 {"id":"n2","type":"network","data":{"name":"net"}},
 {"id":"n3","type":"bucket","data":{"name":"files"}}],
 "edges":[{"id":"e1","source":"n1","target":"n2"}]}`

func newTenant(t *testing.T, pool *pgxpool.Pool, doc string) tenant {
	t.Helper()
	ctx := context.Background()
	tn := tenant{token: "tok-" + uuid.NewString(), user: uuid.NewString(), org: uuid.NewString(), project: uuid.NewString(), diagram: uuid.NewString()}
	stmts := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users (id, email, full_name, picture_url) VALUES ($1, $2, 'T', '')`, []any{tn.user, tn.user + "@example.com"}},
		{`INSERT INTO organizations (id, name, created_by) VALUES ($1, $2, $3)`, []any{tn.org, tn.org[:8], tn.user}},
		{`INSERT INTO organization_members (organization_id, member_id) VALUES ($1, $2)`, []any{tn.org, tn.user}},
		{`INSERT INTO projects (id, organization_id, created_by, name) VALUES ($1, $2, $3, 'p')`, []any{tn.project, tn.org, tn.user}},
		{`INSERT INTO diagrams (id, project_id, created_by, name, data) VALUES ($1, $2, $3, 'd', $4::jsonb)`, []any{tn.diagram, tn.project, tn.user, doc}},
		{`INSERT INTO diagram_history (diagram_id, version, updated_by, name, data) VALUES ($1, 3, $2, 'd', $3::jsonb)`, []any{tn.diagram, tn.user, doc}},
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return tn
}

type env struct {
	srv     *httptest.Server
	pool    *pgxpool.Pool
	verify  fakeVerifier
	logs    *logstore.Disk
	creds   *credentials.Local
	secrets string
	logBuf  *syncBuffer
	runner  *jobs.Runner
	// code is the API's code store, and codeTemp the folder uploads wait in.
	// The API reaches code through gate, which can pause an upload.
	code     *codestore.Disk
	gate     *gatedCode
	codeTemp string
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func newEnv(t *testing.T) *env {
	t.Helper()
	return newEnvWith(t, mock.New())
}

// newEnvWith is newEnv with c as the only connector.
func newEnvWith(t *testing.T, c connectors.Connector) *env {
	t.Helper()
	if testDB == nil {
		t.Skip("docker is not available")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testDB.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	dir := t.TempDir()
	key := make([]byte, credentials.KeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	creds, err := credentials.NewLocal(filepath.Join(dir, "secrets"), key)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := logstore.NewDisk(filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatal(err)
	}
	reg := connectors.NewRegistry()
	if err := reg.Register(c); err != nil {
		t.Fatal(err)
	}
	logBuf := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	st := &store.Store{Pool: pool}
	runner := jobs.New(jobs.Config{Registry: reg, Store: st, Broker: brokerFunc(creds.Get), Logs: logs, Logger: logger, Workers: 2})
	runner.Start()
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = runner.Stop(c) //nolint:errcheck // test cleanup
	})
	code, err := codestore.NewDisk(filepath.Join(dir, "code"))
	if err != nil {
		t.Fatal(err)
	}
	gate := &gatedCode{Objects: code}
	v := fakeVerifier{}
	a := &API{Store: st, Registry: reg, Credentials: creds, Jobs: runner, Logs: logs, Verifier: v, Logger: logger,
		Code: gate, CodeTempDir: filepath.Join(dir, "tmp")}
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)
	return &env{srv: srv, pool: pool, verify: v, logs: logs, creds: creds, secrets: filepath.Join(dir, "secrets"), logBuf: logBuf, runner: runner,
		code: code, gate: gate, codeTemp: filepath.Join(dir, "tmp")}
}

type brokerFunc func(context.Context, string) (connectors.Secrets, error)

func (b brokerFunc) Credentials(ctx context.Context, ref string) (connectors.Secrets, error) {
	return b(ctx, ref)
}

func (e *env) tenant(t *testing.T, doc string) tenant {
	tn := newTenant(t, e.pool, doc)
	e.verify[tn.token] = tn.user
	return tn
}

// call sends a request as tn (no token when tn is nil) and returns the status and body.
func (e *env) call(t *testing.T, tn *tenant, method, path string, body any) (int, string) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, rd)
	if err != nil {
		t.Fatal(err)
	}
	if tn != nil {
		req.Header.Set("Authorization", tn.token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(b)
}

func field(t *testing.T, body, name string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("body %q: %v", body, err)
	}
	s, _ := m[name].(string)
	return s
}

const secretToken = mock.ValidToken

func (e *env) connect(t *testing.T, tn *tenant, name string) string {
	t.Helper()
	code, body := e.call(t, tn, "POST", "/v1/organizations/"+tn.org+"/connections",
		map[string]any{"connector": "mock", "name": name, "region": "mock-1", "credentials": map[string]string{"token": secretToken}})
	if code != http.StatusCreated {
		t.Fatalf("create connection: %d %s", code, body)
	}
	return field(t, body, "id")
}

func (e *env) deploy(t *testing.T, tn *tenant, connID string, exclusions []string) string {
	t.Helper()
	code, body := e.call(t, tn, "POST", "/v1/organizations/"+tn.org+"/diagrams/"+tn.diagram+"/deployments",
		map[string]any{"connection_id": connID, "settings": map[string]any{"image": "debian-12"}, "exclusions": exclusions})
	if code != http.StatusCreated {
		t.Fatalf("create deployment: %d %s", code, body)
	}
	return field(t, body, "id")
}

// runAndWait starts a run and polls it to an end state, recording each status seen.
func (e *env) runAndWait(t *testing.T, tn *tenant, depID string, action string) (string, []string) {
	t.Helper()
	code, body := e.call(t, tn, "POST", "/v1/organizations/"+tn.org+"/deployments/"+depID+"/runs", map[string]string{"action": action})
	if code != http.StatusAccepted {
		t.Fatalf("%s: %d %s", action, code, body)
	}
	runID := field(t, body, "id")
	seen := []string{field(t, body, "status")}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		_, b := e.call(t, tn, "GET", "/v1/organizations/"+tn.org+"/runs/"+runID, nil)
		s := field(t, b, "status")
		if s != seen[len(seen)-1] {
			seen = append(seen, s)
		}
		if s == "completed" || s == "failed" {
			return runID, seen
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s did not finish; saw %v", action, seen)
	return "", nil
}

func TestSchemaAppliesTwice(t *testing.T) {
	e := newEnv(t)
	sql, err := os.ReadFile(pgtest.RepoFile("storage-sql/adapter-tables.sql"))
	if err != nil {
		t.Fatal(err)
	}
	st := &store.Store{Pool: e.pool}
	for i := 0; i < 2; i++ {
		if err := st.ApplySchema(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply %d: %v", i+1, err)
		}
	}
	var rls int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM pg_class
		WHERE relname IN ('connections', 'deployments', 'runs') AND relrowsecurity`).Scan(&rls); err != nil || rls != 3 {
		t.Fatalf("row-level security on %d of 3 tables (%v)", rls, err)
	}
}

func TestSignInIsRequired(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)
	code, _ := e.call(t, nil, "GET", "/v1/organizations/"+tn.org+"/connections", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", code)
	}
	bad := tn
	bad.token = "not-a-token"
	if code, _ := e.call(t, &bad, "GET", "/v1/catalog", nil); code != http.StatusUnauthorized {
		t.Fatalf("bad token: %d", code)
	}
	if code, _ := e.call(t, nil, "GET", "/healthz", nil); code != http.StatusOK {
		t.Fatalf("healthz: %d", code)
	}
	code, body := e.call(t, &tn, "GET", "/v1/catalog", nil)
	if code != http.StatusOK || !strings.Contains(body, `"connector":"mock"`) {
		t.Fatalf("catalog: %d %s", code, body)
	}
}

func TestRunMovesThroughStatesWithLogs(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)
	connID := e.connect(t, &tn, "lab")

	// The bucket (n3) is not supported by the mock cloud.
	notReady := e.deploy(t, &tn, connID, nil)
	code, body := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/deployments/"+notReady+"/compatibility", nil)
	if code != http.StatusOK || !strings.Contains(body, `"ready":false`) || !strings.Contains(body, `"node_id":"n3"`) {
		t.Fatalf("compatibility: %d %s", code, body)
	}
	if code, body := e.call(t, &tn, "POST", "/v1/organizations/"+tn.org+"/deployments/"+notReady+"/runs", map[string]string{"action": "apply"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("apply while not ready: %d %s", code, body)
	}
	if _, err := e.pool.Exec(context.Background(), `DELETE FROM deployments WHERE id = $1`, notReady); err != nil {
		t.Fatalf("a new deployment could not be deleted: %v", err)
	}

	depID := e.deploy(t, &tn, connID, []string{"n3"})
	code, body = e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/deployments/"+depID+"/compatibility", nil)
	if code != http.StatusOK || !strings.Contains(body, `"ready":true`) {
		t.Fatalf("compatibility with exclusion: %d %s", code, body)
	}

	for _, action := range []string{"plan", "apply", "destroy"} {
		runID, seen := e.runAndWait(t, &tn, depID, action)
		if seen[0] != "queued" || seen[len(seen)-1] != "completed" {
			t.Fatalf("%s moved through %v", action, seen)
		}
		_, b := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/runs/"+runID, nil)
		// A destroy does not read the diagram, so it records no version.
		wantVersion := `"diagram_version":3`
		if action == "destroy" {
			wantVersion = `"diagram_version":null`
		}
		if !strings.Contains(b, wantVersion) || !strings.Contains(b, `"started_at":"`) || !strings.Contains(b, `"finished_at":"`) {
			t.Fatalf("run record: %s", b)
		}
		code, logText := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/runs/"+runID+"/logs", nil)
		if code != http.StatusOK || !strings.Contains(logText, "Run "+runID) {
			t.Fatalf("%s log: %d %q", action, code, logText)
		}
		if strings.Contains(logText, secretToken) {
			t.Fatalf("%s log holds the credential", action)
		}
	}

	var status string
	if err := e.pool.QueryRow(context.Background(), `SELECT status FROM deployments WHERE id = $1`, depID).Scan(&status); err != nil || status != "destroyed" {
		t.Fatalf("deployment status %q (%v)", status, err)
	}
}

func TestCrossTenantIsRefused(t *testing.T) {
	e := newEnv(t)
	a := e.tenant(t, blocksDoc)
	b := e.tenant(t, blocksDoc)
	connA := e.connect(t, &a, "a-cloud")
	depA := e.deploy(t, &a, connA, []string{"n3"})
	runA, _ := e.runAndWait(t, &a, depA, "plan")
	connB := e.connect(t, &b, "b-cloud")

	refused := func(code int) bool { return code == http.StatusForbidden || code == http.StatusNotFound }
	cases := []struct {
		name, method, path string
		body               any
	}{
		{"list A's connections", "GET", "/v1/organizations/" + a.org + "/connections", nil},
		{"create in A's org", "POST", "/v1/organizations/" + a.org + "/connections",
			map[string]any{"connector": "mock", "name": "x", "region": "mock-1", "credentials": map[string]string{"token": secretToken}}},
		{"verify A's connection through A's org", "POST", "/v1/organizations/" + a.org + "/connections/" + connA + "/verify", nil},
		{"verify A's connection through B's org", "POST", "/v1/organizations/" + b.org + "/connections/" + connA + "/verify", nil},
		{"delete A's connection through B's org", "DELETE", "/v1/organizations/" + b.org + "/connections/" + connA, nil},
		{"deploy A's diagram on B's connection", "POST", "/v1/organizations/" + b.org + "/diagrams/" + a.diagram + "/deployments",
			map[string]any{"connection_id": connB}},
		{"deploy A's diagram through A's org", "POST", "/v1/organizations/" + a.org + "/diagrams/" + a.diagram + "/deployments",
			map[string]any{"connection_id": connB}},
		{"deploy B's diagram on A's connection", "POST", "/v1/organizations/" + b.org + "/diagrams/" + b.diagram + "/deployments",
			map[string]any{"connection_id": connA}},
		{"A's compatibility through B's org", "GET", "/v1/organizations/" + b.org + "/deployments/" + depA + "/compatibility", nil},
		{"run A's deployment through B's org", "POST", "/v1/organizations/" + b.org + "/deployments/" + depA + "/runs", map[string]string{"action": "apply"}},
		{"remove A's deployment through B's org", "DELETE", "/v1/organizations/" + b.org + "/deployments/" + depA, nil},
		{"remove A's deployment through A's org", "DELETE", "/v1/organizations/" + a.org + "/deployments/" + depA, nil},
		{"A's run through B's org", "GET", "/v1/organizations/" + b.org + "/runs/" + runA, nil},
		{"A's run log through B's org", "GET", "/v1/organizations/" + b.org + "/runs/" + runA + "/logs", nil},
	}
	for _, c := range cases {
		code, body := e.call(t, &b, c.method, c.path, c.body)
		if !refused(code) {
			t.Errorf("%s: %d %s", c.name, code, body)
		}
		if strings.Contains(body, a.org) || strings.Contains(body, connA) {
			t.Errorf("%s: the refusal names A's ids: %s", c.name, body)
		}
	}

	// A deployment that mixes A's diagram with B's connection, written behind
	// the API's back, is still invisible to both through the API.
	mixed := uuid.NewString()
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO deployments (id, diagram_id, connection_id) VALUES ($1, $2, $3)`, mixed, a.diagram, connB); err != nil {
		t.Fatal(err)
	}
	for _, tn := range []*tenant{&a, &b} {
		if code, _ := e.call(t, tn, "GET", "/v1/organizations/"+tn.org+"/deployments/"+mixed+"/compatibility", nil); code != http.StatusNotFound {
			t.Errorf("mixed deployment through %s's org: %d", tn.org, code)
		}
		if code, _ := e.call(t, tn, "DELETE", "/v1/organizations/"+tn.org+"/deployments/"+mixed, nil); code != http.StatusNotFound {
			t.Errorf("remove the mixed deployment through %s's org: %d", tn.org, code)
		}
	}

	// A's own calls still work.
	if code, _ := e.call(t, &a, "GET", "/v1/organizations/"+a.org+"/runs/"+runA, nil); code != http.StatusOK {
		t.Fatalf("A's own run: %d", code)
	}
}

func TestCredentialsNeverLeave(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)

	code, body := e.call(t, &tn, "POST", "/v1/organizations/"+tn.org+"/connections",
		map[string]any{"connector": "mock", "name": "bad", "region": "mock-1", "credentials": map[string]string{"token": "wrong-token-55aa"}})
	if code != http.StatusUnprocessableEntity || strings.Contains(body, "wrong-token-55aa") {
		t.Fatalf("bad credentials: %d %s", code, body)
	}

	connID := e.connect(t, &tn, "good")
	_, list := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/connections", nil)
	_, verified := e.call(t, &tn, "POST", "/v1/organizations/"+tn.org+"/connections/"+connID+"/verify", nil)
	for _, b := range []string{list, verified} {
		if strings.Contains(b, secretToken) || strings.Contains(b, "secret_ref") || strings.Contains(b, "local:") {
			t.Fatalf("a response carries the credential or its reference: %s", b)
		}
	}
	if field(t, verified, "status") != "working" {
		t.Fatalf("verify: %s", verified)
	}

	var ref string
	if err := e.pool.QueryRow(context.Background(), `SELECT secret_ref FROM connections WHERE id = $1`, connID).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	var row string
	if err := e.pool.QueryRow(context.Background(), `SELECT row_to_json(c)::text FROM connections c WHERE id = $1`, connID).Scan(&row); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(row, secretToken) {
		t.Fatal("the database row holds the credential")
	}
	files, err := os.ReadDir(e.secrets)
	if err != nil || len(files) == 0 {
		t.Fatalf("no secret files (%v)", err)
	}
	for _, f := range files {
		b, _ := os.ReadFile(filepath.Join(e.secrets, f.Name())) //nolint:errcheck // test read
		if bytes.Contains(b, []byte(secretToken)) {
			t.Fatal("a secret file holds the credential in clear")
		}
	}

	depID := e.deploy(t, &tn, connID, []string{"n3"})
	e.runAndWait(t, &tn, depID, "apply")
	if strings.Contains(e.logBuf.String(), secretToken) {
		t.Fatal("the service log holds the credential")
	}

	// Live deployments block the delete; after destroy it goes through and the secret is gone.
	if code, body := e.call(t, &tn, "DELETE", "/v1/organizations/"+tn.org+"/connections/"+connID, nil); code != http.StatusConflict {
		t.Fatalf("delete while live: %d %s", code, body)
	}
	if _, err := e.pool.Exec(context.Background(), `DELETE FROM diagrams WHERE id = $1`, tn.diagram); err == nil {
		t.Fatal("a diagram with a live deployment was deleted")
	}
	e.runAndWait(t, &tn, depID, "destroy")
	if code, body := e.call(t, &tn, "DELETE", "/v1/organizations/"+tn.org+"/connections/"+connID, nil); code != http.StatusNoContent {
		t.Fatalf("delete after destroy: %d %s", code, body)
	}
	if _, err := e.creds.Get(context.Background(), ref); !errors.Is(err, credentials.ErrNotFound) {
		t.Fatalf("the secret survived the delete: %v", err)
	}
	if code, _ := e.call(t, &tn, "POST", "/v1/organizations/"+tn.org+"/connections/"+connID+"/verify", nil); code != http.StatusNotFound {
		t.Fatalf("verify after delete: %d", code)
	}
}

func TestOneActiveRunPerDeployment(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)
	connID := e.connect(t, &tn, "busy")
	depID := e.deploy(t, &tn, connID, []string{"n3"})
	runID := uuid.NewString()
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO runs (id, deployment_id, action, log_key) VALUES ($1, $2, 'apply', $3)`,
		runID, depID, logstore.KeyForRun(runID)); err != nil {
		t.Fatal(err)
	}
	if code, body := e.call(t, &tn, "POST", "/v1/organizations/"+tn.org+"/deployments/"+depID+"/runs", map[string]string{"action": "plan"}); code != http.StatusConflict {
		t.Fatalf("second active run: %d %s", code, body)
	}
	st := &store.Store{Pool: e.pool}
	if n, err := st.FailInterrupted(context.Background(), jobs.FailInterruptedMessage); err != nil || n < 1 {
		t.Fatalf("fail interrupted: %d %v", n, err)
	}
	_, body := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/runs/"+runID, nil)
	if field(t, body, "status") != "failed" {
		t.Fatalf("interrupted run: %s", body)
	}
}

// TestOneOrganizationCannotFillTheQueue: an organization with
// DefaultMaxRunsPerOrg runs queued or running is refused another with 429,
// and another organization can still start one.
func TestOneOrganizationCannotFillTheQueue(t *testing.T) {
	e := newEnv(t)
	busy := e.tenant(t, blocksDoc)
	ctx := context.Background()
	var queued []string
	for i := 0; i < DefaultMaxRunsPerOrg; i++ {
		depID := e.deploy(t, &busy, e.connect(t, &busy, fmt.Sprintf("c%d", i)), []string{"n3"})
		runID := uuid.NewString()
		if _, err := e.pool.Exec(ctx, `INSERT INTO runs (id, deployment_id, action, log_key) VALUES ($1, $2, 'apply', $3)`,
			runID, depID, logstore.KeyForRun(runID)); err != nil {
			t.Fatal(err)
		}
		queued = append(queued, runID)
	}
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `UPDATE runs SET status = 'failed' WHERE id = ANY($1)`, queued) //nolint:errcheck // test cleanup
	})
	depID := e.deploy(t, &busy, e.connect(t, &busy, "one-more"), []string{"n3"})
	code, body := e.call(t, &busy, "POST", "/v1/organizations/"+busy.org+"/deployments/"+depID+"/runs", map[string]string{"action": "plan"})
	if code != http.StatusTooManyRequests || !strings.Contains(body, "too_many_runs") {
		t.Fatalf("a run over the organization's cap: %d %s", code, body)
	}

	other := e.tenant(t, blocksDoc)
	otherDep := e.deploy(t, &other, e.connect(t, &other, "free"), []string{"n3"})
	if _, seen := e.runAndWait(t, &other, otherDep, "plan"); seen[len(seen)-1] != "completed" {
		t.Fatalf("another organization's run: %v", seen)
	}
}

func TestADiagramThatIsNotBlocksIsRefused(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, `{"nodes":[],"edges":[]}`)
	connID := e.connect(t, &tn, "aws-like")
	code, body := e.call(t, &tn, "POST", "/v1/organizations/"+tn.org+"/diagrams/"+tn.diagram+"/deployments", map[string]any{"connection_id": connID})
	if code != http.StatusUnprocessableEntity || !strings.Contains(body, "not_blocks") {
		t.Fatalf("A's diagram: %d %s", code, body)
	}
}
