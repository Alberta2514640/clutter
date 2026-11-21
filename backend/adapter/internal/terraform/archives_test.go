package terraform

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/codestore"
	"github.com/google/uuid"
)

func zipOf(t *testing.T, names ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("code of " + n)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func hexSHA(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// storeZip puts body in a code store under a valid key and returns its archive.
func storeZip(t *testing.T, store codestore.Objects, id string, body []byte) Archive {
	t.Helper()
	sha := hexSHA(body)
	key := codestore.Key(uuid.NewString(), uuid.NewString(), "node1", sha)
	if err := store.Put(context.Background(), key, bytes.NewReader(body), int64(len(body))); err != nil {
		t.Fatal(err)
	}
	return Archive{ID: id, Key: key, SHA256: sha}
}

func codeRunner(t *testing.T, l *fakeRunLauncher) (*Runner, codestore.Objects) {
	t.Helper()
	store, err := codestore.NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := newRunner(t, "", l)
	r.Code = store
	return r, store
}

func TestRunStagesUploadedCodeUnderTheRunnersName(t *testing.T) {
	l := &fakeRunLauncher{}
	r, store := codeRunner(t, l)
	body := zipOf(t, "index.js", "lib/util.js")
	j := remoteJob()
	j.Archives = []Archive{storeZip(t, store, "c1234567_n1", body)}
	if err := r.Run(context.Background(), j); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := l.files["code-c1234567_n1.zip"]
	if !ok || got != string(body) {
		t.Fatalf("the run folder holds %v, want the uploaded zip as code-c1234567_n1.zip", keys(l.files))
	}
}

func TestRunRefusesArchivesItDidNotName(t *testing.T) {
	// A zip handed as a rendered file is refused: only .tf files pass, and
	// archives come only through Job.Archives.
	err := CheckFiles(map[string][]byte{"main.tf": []byte(""), "code-x.zip": []byte("PK")}, testProviders)
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("a rendered zip was accepted: %v", err)
	}

	sha := strings.Repeat("a", 64)
	good := codestore.Key(uuid.NewString(), uuid.NewString(), "n1", sha)
	cases := map[string][]Archive{
		"path in the id":      {{ID: "../evil", Key: good, SHA256: sha}},
		"dot in the id":       {{ID: "a.tf", Key: good, SHA256: sha}},
		"empty id":            {{ID: "", Key: good, SHA256: sha}},
		"id given twice":      {{ID: "a", Key: good, SHA256: sha}, {ID: "a", Key: good, SHA256: sha}},
		"key outside":         {{ID: "a", Key: "code/../../state/x.zip", SHA256: sha}},
		"key in another tree": {{ID: "a", Key: "runs/x/in/files/a.zip", SHA256: sha}},
		"digest not hex":      {{ID: "a", Key: good, SHA256: "nothex"}},
	}
	for name, as := range cases {
		t.Run(name, func(t *testing.T) {
			l := &fakeRunLauncher{}
			r, _ := codeRunner(t, l)
			j := remoteJob()
			j.Archives = as
			err := r.Run(context.Background(), j)
			if !errors.Is(err, ErrRefused) || !errors.Is(err, connectors.ErrNothingBuilt) {
				t.Fatalf("Run = %v, want a refusal that built nothing", err)
			}
			if l.run != nil {
				t.Fatal("the run was launched")
			}
		})
	}
}

func TestRunRefusesCodeThatIsNotWhatWasUploaded(t *testing.T) {
	body := zipOf(t, "index.js")
	t.Run("changed after upload", func(t *testing.T) {
		l := &fakeRunLauncher{}
		r, store := codeRunner(t, l)
		a := storeZip(t, store, "n1", body)
		other := zipOf(t, "other.js")
		if err := store.Put(context.Background(), a.Key, bytes.NewReader(other), int64(len(other))); err != nil {
			t.Fatal(err)
		}
		j := remoteJob()
		j.Archives = []Archive{a}
		err := r.Run(context.Background(), j)
		if connectors.UserMessage(err, "") == "" || !errors.Is(err, connectors.ErrNothingBuilt) || l.run != nil {
			t.Fatalf("Run = %v, want the user told to start again, with nothing launched", err)
		}
	})
	t.Run("removed", func(t *testing.T) {
		l := &fakeRunLauncher{}
		r, store := codeRunner(t, l)
		a := storeZip(t, store, "n1", body)
		if err := store.Delete(context.Background(), a.Key); err != nil {
			t.Fatal(err)
		}
		j := remoteJob()
		j.Archives = []Archive{a}
		if err := r.Run(context.Background(), j); connectors.UserMessage(err, "") == "" || l.run != nil {
			t.Fatalf("Run = %v, want the user told to start again", err)
		}
	})
	t.Run("traversal stored behind the upload check", func(t *testing.T) {
		l := &fakeRunLauncher{}
		r, store := codeRunner(t, l)
		evil := zipOf(t, "../../etc/cron.d/x")
		j := remoteJob()
		j.Archives = []Archive{storeZip(t, store, "n1", evil)}
		err := r.Run(context.Background(), j)
		if !errors.Is(err, ErrRefused) || l.run != nil {
			t.Fatalf("Run = %v, want the zip refused before launch", err)
		}
	})
	t.Run("no code store", func(t *testing.T) {
		l := &fakeRunLauncher{}
		r := newRunner(t, "", l)
		j := remoteJob()
		j.Archives = []Archive{{ID: "n1", Key: codestore.Key(uuid.NewString(), uuid.NewString(), "n1", hexSHA(body)), SHA256: hexSHA(body)}}
		if err := r.Run(context.Background(), j); err == nil || l.run != nil {
			t.Fatalf("Run = %v, want a refusal", err)
		}
	})
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
