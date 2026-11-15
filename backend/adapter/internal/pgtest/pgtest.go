// Package pgtest starts a throwaway Postgres in Docker with A's schema and
// B's tables, for tests. Without Docker it returns ErrNoDocker, and the tests
// that need it skip.
package pgtest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Image is the Postgres image A2 uses, pinned by digest.
const Image = "postgres:17-alpine@sha256:266d3c257594e195b309ae9d1805e54c788f1d7b375f56bf6c367aac6c917e7b"

// ErrNoDocker means Docker is not installed or not running.
var ErrNoDocker = errors.New("docker is not available")

// DB is one running container.
type DB struct {
	URL       string
	container string
}

// RepoFile returns the absolute path of a file under backend/, such as
// "storage-sql/adapter-tables.sql".
func RepoFile(rel string) string {
	_, here, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(here), "..", "..", "..", filepath.FromSlash(rel))
}

// SchemaFiles are loaded in this order, as A2 loads them.
var SchemaFiles = []string{
	"devserver/sql/00-supabase-roles.sql",
	"storage-sql/application-tables.sql",
	"storage-sql/seed-resources.sql",
	"storage-sql/adapter-tables.sql",
}

// Start runs a container on a free local port and loads SchemaFiles.
func Start() (*DB, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, ErrNoDocker
	}
	if err := exec.Command("docker", "info", "--format", "{{.ServerVersion}}").Run(); err != nil {
		return nil, ErrNoDocker
	}
	out, err := exec.Command("docker", "run", "--rm", "-d",
		"-e", "POSTGRES_PASSWORD=pgtest", "-e", "POSTGRES_DB=clutter",
		"-p", "127.0.0.1::5432", Image).Output()
	if err != nil {
		return nil, fmt.Errorf("docker run: %w", err)
	}
	db := &DB{container: strings.TrimSpace(string(out))}
	port, err := exec.Command("docker", "port", db.container, "5432/tcp").Output()
	if err != nil {
		db.Stop()
		return nil, fmt.Errorf("docker port: %w", err)
	}
	addr := strings.TrimSpace(strings.Split(string(port), "\n")[0])
	db.URL = "postgres://postgres:pgtest@" + addr + "/clutter?sslmode=disable"

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	conn, err := waitConnect(ctx, db.URL)
	if err != nil {
		db.Stop()
		return nil, err
	}
	defer conn.Close(context.Background())
	for _, f := range SchemaFiles {
		sql, err := os.ReadFile(RepoFile(f))
		if err != nil {
			db.Stop()
			return nil, err
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			db.Stop()
			return nil, fmt.Errorf("load %s: %w", f, err)
		}
	}
	return db, nil
}

// waitConnect waits for the server to accept connections. The image's
// first-start server listens on a socket only, so a TCP connection that works
// is the final server.
func waitConnect(ctx context.Context, url string) (*pgx.Conn, error) {
	for {
		conn, err := pgx.Connect(ctx, url)
		if err == nil {
			if err = conn.Ping(ctx); err == nil {
				return conn, nil
			}
			conn.Close(ctx)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("postgres did not start: %w", err)
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// Stop removes the container.
func (d *DB) Stop() {
	if d == nil || d.container == "" {
		return
	}
	_ = exec.Command("docker", "rm", "-f", d.container).Run() //nolint:errcheck // best effort
}
