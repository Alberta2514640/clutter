package api

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/codestore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/store"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/translate"
)

const codeDoc = `{"engine":"blocks","nodes":[
 {"id":"f1","type":"function","data":{"name":"api","fields":{}}},
 {"id":"v1","type":"vm","data":{"name":"web","fields":{}}}],
 "edges":[]}`

func zipBytes(t *testing.T, names ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("body of " + n)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func shaOf(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// putCode uploads body as node's code in org's diagram, as tn.
func (e *env) putCode(t *testing.T, tn *tenant, org, diagram, node, fileName, ctype string, body io.Reader, length int64) (int, string) {
	t.Helper()
	u := e.srv.URL + "/v1/organizations/" + org + "/diagrams/" + diagram + "/code/" + node + "?file_name=" + url.QueryEscape(fileName)
	req, err := http.NewRequest(http.MethodPut, u, body)
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = length
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("Authorization", tn.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

func (e *env) putZip(t *testing.T, tn *tenant, node string, body []byte) (int, string) {
	t.Helper()
	return e.putCode(t, tn, tn.org, tn.diagram, node, "code.zip", "application/zip", bytes.NewReader(body), int64(len(body)))
}

func (e *env) listCode(t *testing.T, tn *tenant) []codeView {
	t.Helper()
	code, body := e.call(t, tn, "GET", "/v1/organizations/"+tn.org+"/diagrams/"+tn.diagram+"/code", nil)
	if code != http.StatusOK {
		t.Fatalf("list code: %d %s", code, body)
	}
	var out struct{ Items []codeView }
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatal(err)
	}
	return out.Items
}

func (e *env) hasObject(t *testing.T, key string) bool {
	t.Helper()
	rc, err := e.code.Open(context.Background(), key)
	if errors.Is(err, codestore.ErrNotFound) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	rc.Close()
	return true
}

func TestCodeUploadReplaceAndDelete(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, codeDoc)
	first := zipBytes(t, "index.js")

	code, body := e.putZip(t, &tn, "f1", first)
	if code != http.StatusOK {
		t.Fatalf("upload: %d %s", code, body)
	}
	if field(t, body, "sha256") != shaOf(first) || field(t, body, "file_name") != "code.zip" || field(t, body, "node_id") != "f1" {
		t.Fatalf("upload answered %s", body)
	}
	firstKey := codestore.Key(tn.org, tn.diagram, "f1", shaOf(first))
	if !e.hasObject(t, firstKey) {
		t.Fatal("the zip was not stored")
	}
	if items := e.listCode(t, &tn); len(items) != 1 || items[0].Size != int64(len(first)) {
		t.Fatalf("list = %+v", items)
	}

	second := zipBytes(t, "index.js", "lib.js")
	if code, body := e.putZip(t, &tn, "f1", second); code != http.StatusOK {
		t.Fatalf("replace: %d %s", code, body)
	}
	if e.hasObject(t, firstKey) {
		t.Fatal("the replaced zip was kept")
	}
	secondKey := codestore.Key(tn.org, tn.diagram, "f1", shaOf(second))
	if !e.hasObject(t, secondKey) {
		t.Fatal("the new zip was not stored")
	}
	if items := e.listCode(t, &tn); len(items) != 1 || items[0].SHA256 != shaOf(second) {
		t.Fatalf("list after replace = %+v", items)
	}
	// The same zip again keeps its object.
	if code, _ := e.putZip(t, &tn, "f1", second); code != http.StatusOK || !e.hasObject(t, secondKey) {
		t.Fatal("uploading the same zip again lost it")
	}

	// A deployment's graph carries the code, from B's own records only.
	a := &API{Store: &store.Store{Pool: e.pool}}
	g, err := translate.Translate([]byte(codeDoc))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.attachCode(context.Background(), tn.org, tn.diagram, &g); err != nil {
		t.Fatal(err)
	}
	if c := g.Blocks["f1"].Code; c == nil || c.Key != secondKey || c.SHA256 != shaOf(second) {
		t.Fatalf("the function's code on the graph = %+v", c)
	}
	if g.Blocks["v1"].Code != nil {
		t.Fatal("a VM got code")
	}

	if code, body := e.call(t, &tn, "DELETE", "/v1/organizations/"+tn.org+"/diagrams/"+tn.diagram+"/code/f1", nil); code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", code, body)
	}
	if e.hasObject(t, secondKey) || len(e.listCode(t, &tn)) != 0 {
		t.Fatal("delete left the zip or its record")
	}
	if code, _ := e.call(t, &tn, "DELETE", "/v1/organizations/"+tn.org+"/diagrams/"+tn.diagram+"/code/f1", nil); code != http.StatusNotFound {
		t.Fatalf("a second delete: %d", code)
	}
}

func TestCodeIsRefusedAcrossOrganizations(t *testing.T) {
	e := newEnv(t)
	owner := e.tenant(t, codeDoc)
	other := e.tenant(t, codeDoc)
	body := zipBytes(t, "index.js")
	if code, b := e.putZip(t, &owner, "f1", body); code != http.StatusOK {
		t.Fatalf("owner upload: %d %s", code, b)
	}
	ownerKey := codestore.Key(owner.org, owner.diagram, "f1", shaOf(body))

	paths := map[string]struct{ org, diagram string }{
		// Another organization's diagram through the caller's own organization.
		"own org, other diagram": {other.org, owner.diagram},
		// An organization the caller is not a member of.
		"other org": {owner.org, owner.diagram},
	}
	for name, p := range paths {
		t.Run(name, func(t *testing.T) {
			want := http.StatusNotFound
			if p.org == owner.org {
				want = http.StatusForbidden
			}
			evil := zipBytes(t, "evil.js")
			if code, b := e.putCode(t, &other, p.org, p.diagram, "f1", "code.zip", "application/zip", bytes.NewReader(evil), int64(len(evil))); code != want {
				t.Fatalf("upload: %d %s, want %d", code, b, want)
			}
			if code, _ := e.call(t, &other, "GET", "/v1/organizations/"+p.org+"/diagrams/"+p.diagram+"/code", nil); code != want {
				t.Fatalf("list: %d, want %d", code, want)
			}
			if code, _ := e.call(t, &other, "DELETE", "/v1/organizations/"+p.org+"/diagrams/"+p.diagram+"/code/f1", nil); code != want {
				t.Fatalf("delete: %d, want %d", code, want)
			}
		})
	}
	if items := e.listCode(t, &owner); len(items) != 1 || items[0].SHA256 != shaOf(body) || !e.hasObject(t, ownerKey) {
		t.Fatal("another organization changed the owner's code")
	}
	// The store itself refuses another organization's id for the diagram.
	a := &API{Store: &store.Store{Pool: e.pool}}
	g, _ := translate.Translate([]byte(codeDoc))
	if err := a.attachCode(context.Background(), other.org, owner.diagram, &g); err != nil || g.Blocks["f1"].Code != nil {
		t.Fatal("another organization's graph got the owner's code")
	}
	if _, _, err := a.Store.PutFunctionCode(context.Background(), store.FunctionCode{
		DiagramID: owner.diagram, NodeID: "f1", OrganizationID: other.org, ObjectKey: ownerKey,
		FileName: "x.zip", Size: 1, SHA256: shaOf(body),
	}, func(context.Context) error {
		t.Error("ensure ran for a diagram of another organization")
		return nil
	}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("PutFunctionCode for another organization's diagram = %v, want ErrNotFound", err)
	}
}

// gatedCode is a code store whose next Put of one key can pause once the
// object is written, before the upload goes on to record it.
type gatedCode struct {
	codestore.Objects
	mu       sync.Mutex
	key      string
	written  chan struct{}
	released chan struct{}
}

// pauseAfterPut arms the gate for key. written closes when that Put has
// stored the object; the Put returns after release is called.
func (g *gatedCode) pauseAfterPut(key string) (written <-chan struct{}, release func()) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.key, g.written, g.released = key, make(chan struct{}), make(chan struct{})
	r := g.released
	return g.written, func() { close(r) }
}

func (g *gatedCode) Put(ctx context.Context, key string, body io.ReadSeeker, size int64) error {
	if err := g.Objects.Put(ctx, key, body, size); err != nil {
		return err
	}
	g.mu.Lock()
	armed := g.key == key
	written, released := g.written, g.released
	if armed {
		g.key = ""
	}
	g.mu.Unlock()
	if armed {
		close(written)
		<-released
	}
	return nil
}

// putZipAsync uploads body as node's code in a goroutine and sends the
// status, or -1 when the request itself failed.
func (e *env) putZipAsync(tn *tenant, node string, body []byte) <-chan int {
	out := make(chan int, 1)
	go func() {
		u := e.srv.URL + "/v1/organizations/" + tn.org + "/diagrams/" + tn.diagram + "/code/" + node + "?file_name=code.zip"
		req, err := http.NewRequest(http.MethodPut, u, bytes.NewReader(body))
		if err != nil {
			out <- -1
			return
		}
		req.Header.Set("Content-Type", "application/zip")
		req.Header.Set("Authorization", tn.token)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			out <- -1
			return
		}
		res.Body.Close()
		out <- res.StatusCode
	}()
	return out
}

func waitFor[T any](t *testing.T, c <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-c:
		return v
	case <-time.After(30 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		panic("unreachable")
	}
}

// An upload's object is stored before the node's lock, so another upload, a
// code delete or the sweep can delete it before the upload records it. The
// record must still name an object that exists.
func TestOverlappingCodeWritesKeepTheRecordedObject(t *testing.T) {
	first := func(t *testing.T) []byte { return zipBytes(t, "index.js") }
	cases := map[string]func(t *testing.T, e *env, tn *tenant){
		// The reported case: while the first zip is uploaded again, an upload
		// of a second zip replaces it and deletes the first zip's object.
		"a replacing upload": func(t *testing.T, e *env, tn *tenant) {
			if code, body := e.putZip(t, tn, "f1", zipBytes(t, "index.js", "lib.js")); code != http.StatusOK {
				t.Fatalf("the overlapping upload: %d %s", code, body)
			}
		},
		"a code delete": func(t *testing.T, e *env, tn *tenant) {
			if code, body := e.call(t, tn, "DELETE", "/v1/organizations/"+tn.org+"/diagrams/"+tn.diagram+"/code/f1", nil); code != http.StatusNoContent {
				t.Fatalf("the overlapping delete: %d %s", code, body)
			}
		},
	}
	for name, overlap := range cases {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			tn := e.tenant(t, codeDoc)
			body := first(t)
			key := codestore.Key(tn.org, tn.diagram, "f1", shaOf(body))
			if code, b := e.putZip(t, &tn, "f1", body); code != http.StatusOK {
				t.Fatalf("upload: %d %s", code, b)
			}

			written, release := e.gate.pauseAfterPut(key)
			done := e.putZipAsync(&tn, "f1", body)
			waitFor(t, written, "the paused upload's put")
			overlap(t, e, &tn)
			if e.hasObject(t, key) {
				t.Fatal("the overlap did not delete the first zip's object, so the test proves nothing")
			}
			release()
			if code := waitFor(t, done, "the paused upload"); code != http.StatusOK {
				t.Fatalf("the paused upload: %d", code)
			}

			items := e.listCode(t, &tn)
			if len(items) != 1 || items[0].SHA256 != shaOf(body) {
				t.Fatalf("list = %+v, want the paused upload's zip", items)
			}
			if !e.hasObject(t, key) {
				t.Fatal("the record names an object that was deleted")
			}
			second := codestore.Key(tn.org, tn.diagram, "f1", shaOf(zipBytes(t, "index.js", "lib.js")))
			if e.hasObject(t, second) {
				t.Fatal("the replaced zip's object was kept")
			}
		})
	}
}

// The sweep deletes the objects no row names: those whose rows went with a
// diagram deleted in A, and strays. It keeps every recorded object, and an
// upload it overlaps still ends with its object stored.
func TestCodeSweepRemovesObjectsNoRowNames(t *testing.T) {
	e := newEnv(t)
	kept := e.tenant(t, codeDoc)
	gone := e.tenant(t, codeDoc)
	keptZip, goneZip := zipBytes(t, "index.js"), zipBytes(t, "main.py")
	for _, u := range []struct {
		tn   *tenant
		body []byte
	}{{&kept, keptZip}, {&gone, goneZip}} {
		if code, b := e.putZip(t, u.tn, "f1", u.body); code != http.StatusOK {
			t.Fatalf("upload: %d %s", code, b)
		}
	}
	keptKey := codestore.Key(kept.org, kept.diagram, "f1", shaOf(keptZip))
	goneKey := codestore.Key(gone.org, gone.diagram, "f1", shaOf(goneZip))
	// A deletes the diagram; the cascade takes the row and leaves the object.
	if _, err := e.pool.Exec(context.Background(), `DELETE FROM diagrams WHERE id = $1`, gone.diagram); err != nil {
		t.Fatal(err)
	}
	stray := codestore.Key(kept.org, kept.diagram, "f2", shaOf([]byte("stray")))
	if err := e.code.Put(context.Background(), stray, bytes.NewReader([]byte("stray")), 5); err != nil {
		t.Fatal(err)
	}

	s := &codestore.Sweeper{Objects: e.code, Records: &store.Store{Pool: e.pool}, PerPass: 1}
	total := 0
	// Three keys, one a pass, then a pass past the end.
	for range 4 {
		n, err := s.Pass(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		total += n
	}
	if total != 2 || e.hasObject(t, goneKey) || e.hasObject(t, stray) {
		t.Fatalf("the sweep deleted %d objects; the deleted diagram's left: %v, the stray left: %v",
			total, e.hasObject(t, goneKey), e.hasObject(t, stray))
	}
	if !e.hasObject(t, keptKey) {
		t.Fatal("the sweep deleted a recorded object")
	}

	// A sweep between an upload's put and its record deletes the new object;
	// the upload stores it again before it records it.
	next := zipBytes(t, "index.js", "next.js")
	nextKey := codestore.Key(kept.org, kept.diagram, "f1", shaOf(next))
	written, release := e.gate.pauseAfterPut(nextKey)
	done := e.putZipAsync(&kept, "f1", next)
	waitFor(t, written, "the paused upload's put")
	s = &codestore.Sweeper{Objects: e.code, Records: &store.Store{Pool: e.pool}}
	if n, err := s.Pass(context.Background()); err != nil || n != 1 || e.hasObject(t, nextKey) {
		t.Fatalf("the sweep during the upload deleted %d, %v; the new object left: %v", n, err, e.hasObject(t, nextKey))
	}
	release()
	if code := waitFor(t, done, "the paused upload"); code != http.StatusOK {
		t.Fatalf("the paused upload: %d", code)
	}
	if !e.hasObject(t, nextKey) || e.hasObject(t, keptKey) {
		t.Fatalf("after the upload: new object %v, replaced object %v", e.hasObject(t, nextKey), e.hasObject(t, keptKey))
	}
}

func TestCodeUploadRefusesWhatIsNotFunctionCode(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, codeDoc)
	good := zipBytes(t, "index.js")
	cases := map[string]struct {
		node, name, ctype string
		body              []byte
		want              int
		code              string
	}{
		"a VM block":                    {"v1", "code.zip", "application/zip", good, http.StatusUnprocessableEntity, "not_a_function"},
		"a block not saved yet":         {"f9", "code.zip", "application/zip", good, http.StatusUnprocessableEntity, "unknown_block"},
		"a node id not in the alphabet": {"f1.zip", "code.zip", "application/zip", good, http.StatusNotFound, "not_found"},
		"not a zip":                     {"f1", "code.zip", "application/zip", []byte("#!/bin/sh\necho hi\n"), http.StatusUnprocessableEntity, "invalid_code"},
		"a tar name":                    {"f1", "code.tar.gz", "application/zip", good, http.StatusUnprocessableEntity, "invalid_file_name"},
		"a name with a folder":          {"f1", "../code.zip", "application/zip", good, http.StatusUnprocessableEntity, "invalid_file_name"},
		"text content type":             {"f1", "code.zip", "text/plain", good, http.StatusUnsupportedMediaType, "not_zip"},
		"traversal inside the zip":      {"f1", "code.zip", "application/zip", zipBytes(t, "../../etc/cron.d/x"), http.StatusUnprocessableEntity, "invalid_code"},
		"an empty body":                 {"f1", "code.zip", "application/zip", nil, http.StatusUnprocessableEntity, "invalid_code"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			code, body := e.putCode(t, &tn, tn.org, tn.diagram, c.node, c.name, c.ctype, bytes.NewReader(c.body), int64(len(c.body)))
			if code != c.want || !bytes.Contains([]byte(body), []byte(`"code":"`+c.code+`"`)) {
				t.Fatalf("%d %s, want %d %s", code, body, c.want, c.code)
			}
		})
	}
	if len(e.listCode(t, &tn)) != 0 {
		t.Fatal("a refused upload was recorded")
	}
}

// zeros is an endless reader of zero bytes.
type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestCodeUploadRefusesTooLarge(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, codeDoc)
	// Announced as too large: refused before the body is read.
	code, body := e.putCode(t, &tn, tn.org, tn.diagram, "f1", "code.zip", "application/zip",
		io.LimitReader(zeros{}, codestore.MaxSize+1), codestore.MaxSize+1)
	if code != http.StatusRequestEntityTooLarge {
		t.Fatalf("announced: %d %s", code, body)
	}
	// Sent without a length: refused once the body passes the limit.
	code, body = e.putCode(t, &tn, tn.org, tn.diagram, "f1", "code.zip", "application/zip",
		io.LimitReader(zeros{}, codestore.MaxSize+1), -1)
	if code != http.StatusRequestEntityTooLarge {
		t.Fatalf("streamed: %d %s", code, body)
	}
	if len(e.listCode(t, &tn)) != 0 {
		t.Fatal("a refused upload was recorded")
	}
}
