package codestore

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask/runtasktest"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// entry is one file of a test zip; link makes it a symbolic link to body.
type entry struct {
	name, body string
	link       bool
}

// makeZip builds a zip from entries.
func makeZip(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.link {
			h.SetMode(os.ModeSymlink | 0o777)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestReceiveAcceptsAZipAndHashesIt(t *testing.T) {
	body := makeZip(t, entry{name: "index.js", body: "exports.handler = async () => 'hi'"},
		entry{name: "node_modules/.bin/tool", body: "../tool/cli.js", link: true},
		entry{name: "./lib/util.js", body: "x"})
	r, err := Receive(t.TempDir(), bytes.NewReader(body), MaxSize)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	defer r.Close()
	sum := sha256.Sum256(body)
	if r.SHA256 != hex.EncodeToString(sum[:]) || r.Size != int64(len(body)) {
		t.Fatalf("got %s/%d, want the content's hash and size", r.SHA256, r.Size)
	}
	got, err := io.ReadAll(r.File)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatal("the received file does not hold the upload from its start")
	}
}

func TestReceiveRefusesUnsafeContent(t *testing.T) {
	cases := map[string]struct {
		body []byte
		want string
	}{
		"not a zip":            {[]byte("#!/bin/sh\necho hi\n"), "not a .zip"},
		"parent step":          {makeZip(t, entry{name: "../evil.sh", body: "x"}), "leaves the archive"},
		"nested parent step":   {makeZip(t, entry{name: "a/../../evil.sh", body: "x"}), "leaves the archive"},
		"absolute path":        {makeZip(t, entry{name: "/etc/cron.d/evil", body: "x"}), "leaves the archive"},
		"drive letter":         {makeZip(t, entry{name: "C:/evil", body: "x"}), "leaves the archive"},
		"backslash":            {makeZip(t, entry{name: `..\evil`, body: "x"}), "leaves the archive"},
		"link outside":         {makeZip(t, entry{name: "a/link", body: "../../etc/passwd", link: true}), "points outside"},
		"absolute link":        {makeZip(t, entry{name: "link", body: "/etc/passwd", link: true}), "points outside"},
		"folders only":         {makeZip(t, entry{name: "empty/"}), "no files"},
		"control in a name":    {makeZip(t, entry{name: "a\nb", body: "x"}), "leaves the archive"},
		"empty upload":         {nil, "empty"},
		"zip with no entries":  {makeZip(t), "no files"},
		"link climbing to top": {makeZip(t, entry{name: "link", body: "..", link: true}), "points outside"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := Receive(t.TempDir(), bytes.NewReader(c.body), MaxSize)
			if err == nil {
				r.Close()
				t.Fatal("accepted")
			}
			var re *RefusedError
			if !errors.As(err, &re) || !strings.Contains(re.Message, c.want) {
				t.Fatalf("err = %v, want a refusal containing %q", err, c.want)
			}
		})
	}
}

func TestReceiveRefusesTooLarge(t *testing.T) {
	body := makeZip(t, entry{name: "big.bin", body: strings.Repeat("x", 4096)})
	if _, err := Receive(t.TempDir(), bytes.NewReader(body), int64(len(body)-1)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
	// Behind http.MaxBytesReader, as the API reads it.
	rec := httptest.NewRecorder()
	limited := http.MaxBytesReader(rec, io.NopCloser(bytes.NewReader(body)), int64(len(body)-1))
	if _, err := Receive(t.TempDir(), limited, int64(len(body)+10)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge behind MaxBytesReader", err)
	}
}

// lyingZip is a zip of one entry that unpacks to realSize zero bytes while its
// header claims claimedSize.
func lyingZip(t *testing.T, realSize int, claimedSize uint64) []byte {
	t.Helper()
	var comp bytes.Buffer
	fw, err := flate.NewWriter(&comp, flate.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	zero := make([]byte, realSize)
	if _, err := fw.Write(zero); err != nil {
		t.Fatal(err)
	}
	if err := fw.Close(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	w, err := zw.CreateRaw(&zip.FileHeader{Name: "a.bin", Method: zip.Deflate, CRC32: crc32.ChecksumIEEE(zero),
		CompressedSize64: uint64(comp.Len()), UncompressedSize64: claimedSize})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(w, &comp); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// A header can understate an entry's size; CheckZip counts what the entry
// really unpacks to.
func TestCheckZipRefusesAHeaderThatUnderstatesTheSize(t *testing.T) {
	refusedWith := func(t *testing.T, err error, want string) {
		t.Helper()
		var re *RefusedError
		if !errors.As(err, &re) || !strings.Contains(re.Message, want) {
			t.Fatalf("err = %v, want a refusal containing %q", err, want)
		}
	}
	// The reported case at a small scale: a few KB that claims 10 bytes
	// and unpacks to 4 MB.
	bomb := lyingZip(t, 4<<20, 10)
	refusedWith(t, CheckZip(bytes.NewReader(bomb), int64(len(bomb))), "could not unpack")
	// Through Receive, as an upload arrives.
	if _, err := Receive(t.TempDir(), bytes.NewReader(bomb), MaxSize); err == nil {
		t.Fatal("Receive accepted the zip")
	}

	// An honest header over the limit is refused before anything unpacks.
	honest := lyingZip(t, 2<<20, 2<<20)
	refusedWith(t, checkZip(bytes.NewReader(honest), int64(len(honest)), 1<<20), "more than 1 MB unpacked")
	if err := checkZip(bytes.NewReader(honest), int64(len(honest)), 3<<20); err != nil {
		t.Fatalf("a zip within the limit: %v", err)
	}
	// A header that claims more than the content holds is refused too.
	overstated := lyingZip(t, 1000, 5000)
	refusedWith(t, CheckZip(bytes.NewReader(overstated), int64(len(overstated))), "could not unpack")
}

// The count is the guard that does not trust archive/zip's own size check: it
// stops reading one byte past the limit.
func TestUnpackedSizeStopsAtTheLimit(t *testing.T) {
	body := lyingZip(t, 2<<20, 2<<20)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unpackedSize(zr.File[0], 1<<20); !errors.Is(err, errOverLimit) {
		t.Fatalf("err = %v, want errOverLimit", err)
	}
	if n, err := unpackedSize(zr.File[0], 2<<20); err != nil || n != 2<<20 {
		t.Fatalf("unpackedSize at the exact limit = %d, %v", n, err)
	}
}

func TestReceiveLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := Receive(dir, strings.NewReader("not a zip"), MaxSize); err == nil {
		t.Fatal("accepted")
	}
	r, err := Receive(dir, bytes.NewReader(makeZip(t, entry{name: "a", body: "b"})), MaxSize)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	left, _ := os.ReadDir(dir)
	if len(left) != 0 {
		t.Fatalf("left %d files behind", len(left))
	}
}

func TestKeys(t *testing.T) {
	sha := strings.Repeat("a", 64)
	k := Key("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222", "node_1-A", sha)
	if !ValidKey(k) {
		t.Fatalf("Key gave %q, which ValidKey refuses", k)
	}
	for _, bad := range []string{
		"code/../x.zip",
		"logs/runs/x.log",
		Key("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222", "../n", sha),
		Key("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222", "n", "zz"),
	} {
		if ValidKey(bad) {
			t.Errorf("ValidKey(%q) = true", bad)
		}
	}
}

func TestValidFileName(t *testing.T) {
	for name, want := range map[string]bool{
		"code.zip": true, "My Function.ZIP": true, "code.tar.gz": false, ".zip": false,
		"a/b.zip": false, `a\b.zip`: false, "a\nb.zip": false, strings.Repeat("a", 252) + ".zip": false,
	} {
		if got := ValidFileName(name); got != want {
			t.Errorf("ValidFileName(%q) = %v, want %v", name, got, want)
		}
	}
}

func testObjects(t *testing.T, o Objects) {
	ctx := context.Background()
	key := Key("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222", "n1", strings.Repeat("b", 64))
	if _, err := o.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Open of a missing key = %v, want ErrNotFound", err)
	}
	if ok, err := o.Exists(ctx, key); ok || err != nil {
		t.Fatalf("Exists of a missing key = %v, %v", ok, err)
	}
	if err := o.Put(ctx, key, strings.NewReader("zip bytes"), 9); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if ok, err := o.Exists(ctx, key); !ok || err != nil {
		t.Fatalf("Exists after Put = %v, %v", ok, err)
	}
	rc, err := o.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "zip bytes" {
		t.Fatalf("read %q", got)
	}
	if err := o.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := o.Delete(ctx, key); err != nil {
		t.Fatalf("a second Delete: %v", err)
	}
	if _, err := o.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Open after Delete = %v", err)
	}
	if err := o.Put(ctx, "code/../../etc/passwd", strings.NewReader("x"), 1); err == nil {
		t.Fatal("Put took a key outside the store")
	}

	// List, in byte order, from a key and up to a limit.
	if keys, err := o.List(ctx, "", 10); err != nil || len(keys) != 0 {
		t.Fatalf("List of an empty store = %v, %v", keys, err)
	}
	org, dia := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	want := []string{Key(org, dia, "n", strings.Repeat("c", 64)), Key(org, dia, "n-1", strings.Repeat("c", 64)), Key(org, dia, "n", strings.Repeat("d", 64))}
	for _, k := range want {
		if err := o.Put(ctx, k, strings.NewReader("z"), 1); err != nil {
			t.Fatal(err)
		}
	}
	slices.Sort(want)
	if keys, err := o.List(ctx, "", 10); err != nil || !slices.Equal(keys, want) {
		t.Fatalf("List = %v, %v; want %v", keys, err, want)
	}
	if keys, err := o.List(ctx, want[0], 1); err != nil || !slices.Equal(keys, want[1:2]) {
		t.Fatalf("List after the first, one key = %v, %v; want %v", keys, err, want[1:2])
	}
	if d, n, ok := KeyNode(want[0]); !ok || d != dia || n == "" {
		t.Fatalf("KeyNode(%q) = %q, %q, %v", want[0], d, n, ok)
	}
}

func TestDisk(t *testing.T) {
	d, err := NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	testObjects(t, d)
}

// headS3 adds HeadObject to the shared S3 fake, answering a missing key the
// way S3 does: 404 with no body, which the SDK names NotFound.
type headS3 struct{ *runtasktest.S3 }

func (h headS3) HeadObject(_ context.Context, in *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if _, ok := h.Object(aws.ToString(in.Bucket), aws.ToString(in.Key)); !ok {
		return nil, &s3types.NotFound{}
	}
	return &s3.HeadObjectOutput{}, nil
}

func TestS3(t *testing.T) {
	if _, err := NewS3(headS3{runtasktest.New()}, "Not A Bucket"); err == nil {
		t.Fatal("a bad bucket name was accepted")
	}
	s, err := NewS3(headS3{runtasktest.New()}, "clutter-adapter-runs-test")
	if err != nil {
		t.Fatal(err)
	}
	testObjects(t, s)
}
