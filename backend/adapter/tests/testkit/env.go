package testkit

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
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/api"
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

// DB is the suite's throwaway Postgres, or nil without Docker.
var DB *pgtest.DB

// Main starts a throwaway Postgres for the package's tests and runs them.
// Without Docker the tests that need it skip (Env does), as B's own do.
func Main(m *testing.M) {
	var err error
	DB, err = pgtest.Start()
	if err != nil && !errors.Is(err, pgtest.ErrNoDocker) {
		fmt.Fprintln(os.Stderr, "pgtest:", err)
		os.Exit(1)
	}
	code := m.Run()
	DB.Stop()
	os.Exit(code)
}

// SyncBuffer is a bytes.Buffer safe for concurrent writes: B's service log.
type SyncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *SyncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *SyncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

type verifier struct {
	mu sync.Mutex
	m  map[string]string
}

func (v *verifier) Verify(_ context.Context, token string) (auth.Identity, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	id, ok := v.m[token]
	if !ok {
		return auth.Identity{}, errors.New("bad token")
	}
	return auth.Identity{UserID: id}, nil
}

// Env is B API in-process: the real routes, store, job runner, credential
// store and log store, on the throwaway Postgres.
type Env struct {
	URL     string
	Pool    *pgxpool.Pool
	Dir     string // B's data folder: secrets, logs, code
	Service *SyncBuffer
	Runner  *jobs.Runner
	verify  *verifier
}

// Options tune NewEnv.
type Options struct {
	Workers       int
	MaxRunsPerOrg int
}

// NewEnv starts B API with these connectors. It skips tb without Docker.
func NewEnv(tb testing.TB, opts Options, cs ...connectors.Connector) *Env {
	tb.Helper()
	if DB == nil {
		tb.Skip("docker is not available, so there is no Postgres for B API")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, DB.URL)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(pool.Close)

	dir := tb.TempDir()
	key := make([]byte, credentials.KeySize)
	if _, err := rand.Read(key); err != nil {
		tb.Fatal(err)
	}
	creds, err := credentials.NewLocal(filepath.Join(dir, "secrets"), key)
	if err != nil {
		tb.Fatal(err)
	}
	logs, err := logstore.NewDisk(filepath.Join(dir, "logs"))
	if err != nil {
		tb.Fatal(err)
	}
	reg := connectors.NewRegistry()
	for _, c := range cs {
		if err := reg.Register(c); err != nil {
			tb.Fatal(err)
		}
	}
	service := &SyncBuffer{}
	logger := slog.New(slog.NewTextHandler(service, &slog.HandlerOptions{Level: slog.LevelDebug}))
	st := &store.Store{Pool: pool}
	workers := opts.Workers
	if workers == 0 {
		workers = 2
	}
	runner := jobs.New(jobs.Config{Registry: reg, Store: st, Broker: brokerFunc(creds.Get), Logs: logs, Logger: logger, Workers: workers})
	runner.Start()
	tb.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = runner.Stop(c) //nolint:errcheck // test cleanup
	})
	code, err := codestore.NewDisk(filepath.Join(dir, "code"))
	if err != nil {
		tb.Fatal(err)
	}
	v := &verifier{m: map[string]string{}}
	a := &api.API{Store: st, Registry: reg, Credentials: creds, Jobs: runner, Logs: logs, Verifier: v, Logger: logger,
		Code: code, CodeTempDir: filepath.Join(dir, "tmp"), MaxRunsPerOrg: opts.MaxRunsPerOrg}
	srv := httptest.NewServer(a.Handler())
	tb.Cleanup(srv.Close)
	return &Env{URL: srv.URL, Pool: pool, Dir: dir, Service: service, Runner: runner, verify: v}
}

type brokerFunc func(context.Context, string) (connectors.Secrets, error)

func (b brokerFunc) Credentials(ctx context.Context, ref string) (connectors.Secrets, error) {
	return b(ctx, ref)
}

// Tenant is one user with one organization, project and blocks diagram.
type Tenant struct {
	Token, User, Org, Project, Diagram string
}

// NewTenant seeds a user, organization, membership, project and diagram
// holding doc, as A's routes would have made them.
func (e *Env) NewTenant(tb testing.TB, doc string) Tenant {
	tb.Helper()
	ctx := context.Background()
	tn := Tenant{Token: "tok-" + uuid.NewString(), User: uuid.NewString(), Org: uuid.NewString(), Project: uuid.NewString(), Diagram: uuid.NewString()}
	stmts := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users (id, email, full_name, picture_url) VALUES ($1, $2, 'T', '')`, []any{tn.User, tn.User + "@example.com"}},
		{`INSERT INTO organizations (id, name, created_by) VALUES ($1, $2, $3)`, []any{tn.Org, tn.Org[:8], tn.User}},
		{`INSERT INTO organization_members (organization_id, member_id) VALUES ($1, $2)`, []any{tn.Org, tn.User}},
		{`INSERT INTO projects (id, organization_id, created_by, name) VALUES ($1, $2, $3, 'p')`, []any{tn.Project, tn.Org, tn.User}},
		{`INSERT INTO diagrams (id, project_id, created_by, name, data) VALUES ($1, $2, $3, 'd', $4::jsonb)`, []any{tn.Diagram, tn.Project, tn.User, doc}},
		{`INSERT INTO diagram_history (diagram_id, version, updated_by, name, data) VALUES ($1, 1, $2, 'd', $3::jsonb)`, []any{tn.Diagram, tn.User, doc}},
	}
	for _, s := range stmts {
		if _, err := e.Pool.Exec(ctx, s.sql, s.args...); err != nil {
			tb.Fatalf("seed: %v", err)
		}
	}
	e.verify.mu.Lock()
	e.verify.m[tn.Token] = tn.User
	e.verify.mu.Unlock()
	return tn
}

// Call sends a request as tn (no token when tn is nil) and returns the
// status and body.
func (e *Env) Call(tb testing.TB, tn *Tenant, method, path string, body any) (int, string) {
	tb.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			tb.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.URL+path, rd)
	if err != nil {
		tb.Fatal(err)
	}
	if tn != nil {
		req.Header.Set("Authorization", tn.Token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		tb.Fatal(err)
	}
	defer res.Body.Close()
	return res.StatusCode, ReadAll(tb, res.Body)
}

// Field reads a top-level string field of a JSON body.
func Field(tb testing.TB, body, name string) string {
	tb.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		tb.Fatalf("body %q: %v", body, err)
	}
	s, _ := m[name].(string)
	return s
}

// Connect creates a connection for tn and returns its id.
func (e *Env) Connect(tb testing.TB, tn *Tenant, connector, region string, creds map[string]string) string {
	tb.Helper()
	code, body := e.Call(tb, tn, "POST", "/v1/organizations/"+tn.Org+"/connections",
		map[string]any{"connector": connector, "name": "c-" + uuid.NewString()[:8], "region": region, "credentials": creds})
	if code != http.StatusCreated {
		tb.Fatalf("create connection: %d %s", code, body)
	}
	return Field(tb, body, "id")
}

// Deploy creates a deployment of tn's diagram on a connection.
func (e *Env) Deploy(tb testing.TB, tn *Tenant, connID string, settings map[string]any, exclusions []string) string {
	tb.Helper()
	code, body := e.Call(tb, tn, "POST", "/v1/organizations/"+tn.Org+"/diagrams/"+tn.Diagram+"/deployments",
		map[string]any{"connection_id": connID, "settings": settings, "exclusions": exclusions})
	if code != http.StatusCreated {
		tb.Fatalf("create deployment: %d %s", code, body)
	}
	return Field(tb, body, "id")
}

// StartRun starts a run and returns its id.
func (e *Env) StartRun(tb testing.TB, tn *Tenant, depID, action string) string {
	tb.Helper()
	code, body := e.Call(tb, tn, "POST", "/v1/organizations/"+tn.Org+"/deployments/"+depID+"/runs", map[string]string{"action": action})
	if code != http.StatusAccepted {
		tb.Fatalf("%s: %d %s", action, code, body)
	}
	return Field(tb, body, "id")
}

// WaitRun polls a run to an end state and returns every status seen.
func (e *Env) WaitRun(tb testing.TB, tn *Tenant, runID string, timeout time.Duration) []string {
	tb.Helper()
	var seen []string
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, b := e.Call(tb, tn, "GET", "/v1/organizations/"+tn.Org+"/runs/"+runID, nil)
		s := Field(tb, b, "status")
		if len(seen) == 0 || s != seen[len(seen)-1] {
			seen = append(seen, s)
		}
		if s == "completed" || s == "failed" {
			return seen
		}
		time.Sleep(10 * time.Millisecond)
	}
	tb.Fatalf("run %s did not finish; saw %v", runID, seen)
	return nil
}

// RunLog returns a run's log through the API.
func (e *Env) RunLog(tb testing.TB, tn *Tenant, runID string) string {
	tb.Helper()
	code, body := e.Call(tb, tn, "GET", "/v1/organizations/"+tn.Org+"/runs/"+runID+"/logs", nil)
	if code != http.StatusOK {
		tb.Fatalf("logs: %d %s", code, body)
	}
	return body
}

// MockDoc is a blocks diagram the mock connector can build.
const MockDoc = `{"engine":"blocks","nodes":[
 {"id":"n1","type":"vm","position":{"x":0,"y":0},"data":{"name":"web","fields":{"size":"small"}}},
 {"id":"n2","type":"network","data":{"name":"net","fields":{"cidr":"10.20.0.0/24"}}},
 {"id":"n3","type":"volume","data":{"name":"data","fields":{"size_gb":5}}},
 {"id":"n4","type":"firewall","data":{"name":"fw","fields":{"ingress":"tcp/22"}}}],
 "edges":[{"id":"e1","source":"n1","target":"n2"},{"id":"e2","source":"n3","target":"n1"},{"id":"e3","source":"n4","target":"n1"}]}`
