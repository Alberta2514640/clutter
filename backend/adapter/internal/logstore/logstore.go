// Package logstore keeps run logs. The job runner writes them; the API reads
// them. Disk is the local implementation; S3 takes its place in the cloud
// behind the same interface (ADAPTER_LOG_STORE).
package logstore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// ErrNotFound means no log has this key.
var ErrNotFound = errors.New("log not found")

// Store keeps run logs by key.
type Store interface {
	// Create starts the log at key, replacing any earlier one.
	Create(key string) (io.WriteCloser, error)
	// Open reads the log at key.
	Open(key string) (io.ReadCloser, error)
}

var keyPattern = regexp.MustCompile(`^runs/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.log$`)

// KeyForRun returns the log key of a run.
func KeyForRun(runID string) string { return "runs/" + runID + ".log" }

func checkKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("log key %q is not valid", key)
	}
	return nil
}

// Disk keeps logs as files under one directory.
type Disk struct{ dir string }

// NewDisk returns a Disk store rooted at dir, creating it.
func NewDisk(dir string) (*Disk, error) {
	if err := os.MkdirAll(filepath.Join(dir, "runs"), 0o700); err != nil {
		return nil, err
	}
	return &Disk{dir: dir}, nil
}

// Create implements Store.
func (d *Disk) Create(key string) (io.WriteCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(d.dir, filepath.FromSlash(key)), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
}

// Open implements Store.
func (d *Disk) Open(key string) (io.ReadCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(d.dir, filepath.FromSlash(key)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

// Memory keeps logs in memory, for tests.
type Memory struct {
	mu   sync.Mutex
	logs map[string]*bytes.Buffer
}

// NewMemory returns an empty Memory store.
func NewMemory() *Memory { return &Memory{logs: map[string]*bytes.Buffer{}} }

type memWriter struct {
	m   *Memory
	buf *bytes.Buffer
}

func (w memWriter) Write(p []byte) (int, error) {
	w.m.mu.Lock()
	defer w.m.mu.Unlock()
	return w.buf.Write(p)
}

func (memWriter) Close() error { return nil }

// Create implements Store.
func (m *Memory) Create(key string) (io.WriteCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	buf := &bytes.Buffer{}
	m.logs[key] = buf
	return memWriter{m: m, buf: buf}, nil
}

// Open implements Store.
func (m *Memory) Open(key string) (io.ReadCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	buf, ok := m.logs[key]
	if !ok {
		return nil, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(bytes.Clone(buf.Bytes()))), nil
}
