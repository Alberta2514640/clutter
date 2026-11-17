package logstore

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask/runtasktest"
	"github.com/google/uuid"
)

const testBucket = "clutter-adapter-runs-x"

func read(t *testing.T, s Store, key string) string {
	t.Helper()
	rc, err := s.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestS3StoreWritesUnderTheLogsPrefix(t *testing.T) {
	m := runtasktest.New()
	s, err := NewS3(m, testBucket)
	if err != nil {
		t.Fatal(err)
	}
	s.Every = time.Hour // only Create and Close upload in this test
	key := KeyForRun(uuid.NewString())

	if _, err := s.Open(key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing log: %v", err)
	}
	w, err := s.Create(key)
	if err != nil {
		t.Fatal(err)
	}
	// Create puts an empty object at once, so a bad bucket fails the start.
	if body, ok := m.Object(testBucket, "logs/"+key); !ok || len(body) != 0 {
		t.Fatalf("no empty log after Create: %q %t", body, ok)
	}
	if _, err := io.WriteString(w, "init done\n"); err != nil {
		t.Fatal(err)
	}
	// While the log is open, this process reads it from memory.
	if got := read(t, s, key); got != "init done\n" {
		t.Fatalf("live log %q", got)
	}
	if _, err := io.WriteString(w, "apply done\n"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if body, _ := m.Object(testBucket, "logs/"+key); string(body) != "init done\napply done\n" {
		t.Fatalf("object %q", body)
	}
	// After Close the log is read from the object, as another process would.
	if got := read(t, s, key); got != "init done\napply done\n" {
		t.Fatalf("closed log %q", got)
	}
	if _, err := io.WriteString(w, "late"); err == nil {
		t.Fatal("a write after Close was accepted")
	}
	for _, k := range m.Keys() {
		if k != testBucket+"/logs/"+key {
			t.Fatalf("object outside the logs prefix: %s", k)
		}
	}
}

func TestS3StoreUploadsAGrowingLog(t *testing.T) {
	m := runtasktest.New()
	s, err := NewS3(m, testBucket)
	if err != nil {
		t.Fatal(err)
	}
	s.Every = time.Millisecond
	key := KeyForRun(uuid.NewString())
	w, err := s.Create(key)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, _ = io.WriteString(w, "line\n")
		}
	}()
	wg.Wait()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if body, _ := m.Object(testBucket, "logs/"+key); len(body) == 250 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the growing log was not uploaded before Close")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestS3StoreRefusesBadInput(t *testing.T) {
	m := runtasktest.New()
	if _, err := NewS3(m, "Not A Bucket"); err == nil {
		t.Fatal("a bad bucket was accepted")
	}
	if _, err := NewS3(nil, testBucket); err == nil {
		t.Fatal("a nil client was accepted")
	}
	s, err := NewS3(m, testBucket)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../x.log", "runs/x.log", "logs/runs/" + uuid.NewString() + ".log"} {
		if _, err := s.Create(key); err == nil {
			t.Errorf("key %q accepted by Create", key)
		}
		if _, err := s.Open(key); err == nil {
			t.Errorf("key %q accepted by Open", key)
		}
	}
	// A bucket B cannot write to fails Create, so the run fails at its start.
	m.FailPut = ".log"
	if _, err := s.Create(KeyForRun(uuid.NewString())); err == nil {
		t.Fatal("Create succeeded without an upload")
	}
}
