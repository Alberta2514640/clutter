// Package codestore keeps the function code users upload through B: one zip
// per organization, diagram and function block. Receive checks an upload
// (size, zip, paths) into a temporary file before anything is stored; Disk
// keeps zips under ADAPTER_DATA_DIR locally, and S3 keeps them in the run bucket in
// the cloud, behind one interface. The Terraform runner (internal/terraform)
// stages a stored zip into a run's folder; nothing in B ever unpacks one.
package codestore

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
)

// Limits on one upload.
const (
	// MaxSize caps a zip's size: 50 MB, AWS Lambda's limit for a zip
	// uploaded directly.
	MaxSize = 50 << 20
	// MaxUnpacked caps the sum of the files' sizes inside a zip: 250 MB, AWS
	// Lambda's limit for unpacked code.
	MaxUnpacked = 250 << 20
	// MaxEntries caps how many files and folders a zip holds.
	MaxEntries = 20000
	// maxLinkTarget caps a symbolic link's target.
	maxLinkTarget = 4096
)

// Errors.
var (
	// ErrNotFound means no object has this key.
	ErrNotFound = errors.New("code not found")
	// ErrTooLarge means the upload is over MaxSize.
	ErrTooLarge = errors.New("the code is larger than the limit")
)

// RefusedError is an upload whose content Clutter refuses. Its message is
// safe to show the user.
type RefusedError struct{ Message string }

func (e *RefusedError) Error() string { return e.Message }

func refused(format string, args ...any) error {
	return &RefusedError{Message: fmt.Sprintf(format, args...)}
}

// Objects keeps zips by key.
type Objects interface {
	// Put stores size bytes from body at key, replacing any object there.
	Put(ctx context.Context, key string, body io.ReadSeeker, size int64) error
	// Open reads the object at key, or returns ErrNotFound.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Exists reports whether an object is at key, without reading it.
	Exists(ctx context.Context, key string) (bool, error)
	// Delete removes the object at key. A missing object is not an error.
	Delete(ctx context.Context, key string) error
	// List returns up to limit keys that sort after after ("" for the
	// first), in byte order. It returns only keys ValidKey accepts.
	List(ctx context.Context, after string, limit int) ([]string, error)
}

// Prefix starts every key. In the cloud it is the folder of the run bucket
// that B API's role may write code to.
const Prefix = "code/"

var (
	uuidPattern = `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`
	keyPattern  = regexp.MustCompile(`^code/` + uuidPattern + `/` + uuidPattern + `/[A-Za-z0-9_-]{1,64}/[0-9a-f]{64}\.zip$`)
)

// Key is the object key of one upload: by organization, diagram, node and
// content, so a replaced zip never overwrites one a run may be staging.
func Key(orgID, diagramID, nodeID, sha string) string {
	return Prefix + orgID + "/" + diagramID + "/" + nodeID + "/" + sha + ".zip"
}

// ValidKey reports whether key has the shape Key gives, so no key can name a
// path outside the store.
func ValidKey(key string) bool { return keyPattern.MatchString(key) }

// KeyNode returns the diagram and node a valid key belongs to.
func KeyNode(key string) (diagramID, nodeID string, ok bool) {
	if !ValidKey(key) {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(key, Prefix), "/")
	return parts[1], parts[2], true
}

// Received is an upload Receive accepted, held in a temporary file.
type Received struct {
	File *os.File
	Size int64
	// SHA256 is the content's digest in lowercase hex.
	SHA256 string
}

// Close removes the temporary file.
func (r *Received) Close() error {
	name := r.File.Name()
	cerr := r.File.Close()
	if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return cerr
}

// Receive copies body into a temporary file in dir, at most max bytes,
// hashing it as it goes, and then checks it with CheckZip. It returns
// ErrTooLarge for a body over max, a *RefusedError for content Clutter
// refuses, and any other error for a fault of its own. On success the caller
// closes the result.
func Receive(dir string, body io.Reader, max int64) (*Received, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(dir, "upload-*.zip")
	if err != nil {
		return nil, err
	}
	r := &Received{File: f}
	ok := false
	defer func() {
		if !ok {
			_ = r.Close() //nolint:errcheck // the upload already failed
		}
	}()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(body, max+1))
	var mbe *http.MaxBytesError
	switch {
	case errors.As(err, &mbe):
		return nil, ErrTooLarge
	case err != nil:
		return nil, err
	case n > max:
		return nil, ErrTooLarge
	case n == 0:
		return nil, refused("The upload is empty. Choose a .zip file with your function's code.")
	}
	if err := CheckZip(f, n); err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	r.Size, r.SHA256 = n, hex.EncodeToString(h.Sum(nil))
	ok = true
	return r, nil
}

// CheckZip refuses content that is not a zip, a zip with no files, more than
// MaxEntries entries or more than MaxUnpacked bytes unpacked, and any entry
// whose path leaves the zip's own folder: an absolute path, a drive letter, a
// ".." step, a backslash or a control character, or a symbolic link whose
// target does any of these. A link inside the zip (node_modules/.bin links,
// for one) is accepted. The errors are *RefusedError.
func CheckZip(r io.ReaderAt, size int64) error {
	return checkZip(r, size, MaxUnpacked)
}

// checkZip is CheckZip with maxUnpacked as the unpacked limit, so a test can
// use a small one.
func checkZip(r io.ReaderAt, size int64, maxUnpacked uint64) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return refused("The file is not a .zip archive. Upload your function's code as one .zip file.")
	}
	if len(zr.File) > MaxEntries {
		return refused("The .zip holds more than %d files and folders.", MaxEntries)
	}
	tooBig := refused("The .zip's files add up to more than %d MB unpacked, the most a function can hold.", maxUnpacked>>20)
	// declared sums the sizes the entries' headers claim, which refuses an
	// honest large zip before anything is unpacked. A header can lie, so
	// unpacked counts the bytes each entry really unpacks to.
	var declared, unpacked uint64
	files := 0
	for _, f := range zr.File {
		name := f.Name
		if err := checkEntryPath(name); err != nil {
			return refused("The .zip holds %s, a path that leaves the archive's own folder. Zip the code from inside its folder.", quoteName(name))
		}
		declared += f.UncompressedSize64
		if declared > maxUnpacked {
			return tooBig
		}
		n, err := unpackedSize(f, maxUnpacked-unpacked)
		switch {
		case errors.Is(err, errOverLimit):
			return tooBig
		case err != nil:
			return refused("The .zip holds %s, a file Clutter could not unpack. Zip the code again.", quoteName(name))
		}
		unpacked += n
		mode := f.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			target, err := linkTarget(f)
			if err != nil {
				return refused("The .zip holds a link, %s, that Clutter could not read.", quoteName(name))
			}
			// A relative target may climb ("../lib/x") as long as it lands
			// inside the archive's folder.
			if strings.HasPrefix(target, "/") || (len(target) >= 2 && target[1] == ':') ||
				!insideRoot(path.Join(path.Dir(name), target)) {
				return refused("The .zip holds a link, %s, that points outside the archive's own folder.", quoteName(name))
			}
			files++
		case mode.IsDir():
		default:
			files++
		}
	}
	if files == 0 {
		return refused("The .zip holds no files.")
	}
	return nil
}

// checkEntryPath refuses a path that is absolute, names a drive, holds a
// backslash or a control character, or has a ".." step. A "./" step, which
// some zip tools write, is harmless and accepted.
func checkEntryPath(name string) error {
	bad := errors.New("unsafe path")
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, `\`) {
		return bad
	}
	if len(name) >= 2 && name[1] == ':' {
		return bad
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return bad
		}
	}
	for _, part := range strings.Split(strings.TrimSuffix(name, "/"), "/") {
		if part == ".." {
			return bad
		}
	}
	return nil
}

// insideRoot reports whether a cleaned, slash-separated relative path stays
// inside the archive's folder.
func insideRoot(p string) bool {
	c := path.Clean(p)
	return c != ".." && !strings.HasPrefix(c, "../") && !strings.HasPrefix(c, "/")
}

var errOverLimit = errors.New("the entry unpacks to more than the limit")

// unpackedSize unpacks one entry into nothing and returns how many bytes it
// held. It stops and returns errOverLimit as soon as the entry passes limit
// bytes, so a zip whose headers understate its size cannot make B unpack
// more than the limit. A folder unpacks to nothing (archive/zip refuses one
// that claims content).
func unpackedSize(f *zip.File, limit uint64) (uint64, error) {
	rc, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer rc.Close()
	n, err := io.Copy(io.Discard, io.LimitReader(rc, int64(limit)+1))
	if err != nil {
		return 0, err
	}
	if uint64(n) > limit {
		return 0, errOverLimit
	}
	return uint64(n), nil
}

func linkTarget(f *zip.File) (string, error) {
	if f.UncompressedSize64 > maxLinkTarget {
		return "", errors.New("link target too long")
	}
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	b, err := io.ReadAll(io.LimitReader(rc, maxLinkTarget+1))
	if err != nil || len(b) == 0 || len(b) > maxLinkTarget {
		return "", errors.New("link target not readable")
	}
	t := string(b)
	if strings.ContainsAny(t, "\\\x00") {
		return "", errors.New("link target not plain")
	}
	return t, nil
}

// quoteName shows an entry's name in a message, cut and quoted so it cannot
// break the message.
func quoteName(name string) string {
	if len(name) > 80 {
		name = name[:80] + "..."
	}
	return fmt.Sprintf("%q", name)
}

// ValidFileName reports whether name can be shown as an upload's file name: a
// plain name ending in .zip, at most 255 bytes, with no folder and no control
// character.
func ValidFileName(name string) bool {
	if len(name) < 5 || len(name) > 255 || !strings.HasSuffix(strings.ToLower(name), ".zip") {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}
