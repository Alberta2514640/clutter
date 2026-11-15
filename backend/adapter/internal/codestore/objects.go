package codestore

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/s3name"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

func checkKey(key string) error {
	if !ValidKey(key) {
		return errors.New("the code key is not valid")
	}
	return nil
}

// Disk keeps zips as files under one folder, locally and in A2.
type Disk struct{ dir string }

// NewDisk returns a Disk store rooted at dir, creating it.
func NewDisk(dir string) (*Disk, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Disk{dir: dir}, nil
}

func (d *Disk) path(key string) string { return filepath.Join(d.dir, filepath.FromSlash(key)) }

// Put implements Objects. It writes a temporary file beside the object and
// renames it, so a reader never sees half a zip.
func (d *Disk) Put(_ context.Context, key string, body io.ReadSeeker, size int64) error {
	if err := checkKey(key); err != nil {
		return err
	}
	p := d.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(p), ".put-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	n, err := io.Copy(f, io.LimitReader(body, size+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n != size {
		err = errors.New("the code's size changed while it was stored")
	}
	if err == nil {
		err = os.Rename(tmp, p)
	}
	if err != nil {
		_ = os.Remove(tmp) //nolint:errcheck // the put already failed
	}
	return err
}

// Open implements Objects.
func (d *Disk) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, err
	}
	f, err := os.Open(d.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

// Exists implements Objects.
func (d *Disk) Exists(_ context.Context, key string) (bool, error) {
	if err := checkKey(key); err != nil {
		return false, err
	}
	_, err := os.Stat(d.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Delete implements Objects.
func (d *Disk) Delete(_ context.Context, key string) error {
	if err := checkKey(key); err != nil {
		return err
	}
	err := os.Remove(d.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// List implements Objects. It reads the whole folder and sorts the keys, which
// suits the small stores Disk is for. Temporary files from Put are not keys.
func (d *Disk) List(_ context.Context, after string, limit int) ([]string, error) {
	root := d.path(Prefix)
	var keys []string
	err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && p == root {
				return fs.SkipAll
			}
			return err
		}
		if e.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(d.dir, p)
		if err != nil {
			return err
		}
		if key := filepath.ToSlash(rel); ValidKey(key) && key > after {
			keys = append(keys, key)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)
	if len(keys) > limit {
		keys = keys[:limit]
	}
	return keys, nil
}

// S3API is the part of the S3 client the S3 store uses.
type S3API interface {
	PutObject(ctx context.Context, in *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, in *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, in *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	DeleteObject(ctx context.Context, in *s3.DeleteObjectInput, opts ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, in *s3.ListObjectsV2Input, opts ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// S3 keeps zips as objects under Prefix in one bucket: B's run bucket in the
// cloud, where no lifecycle rule expires that prefix. The bucket encrypts
// every object by default.
type S3 struct {
	client S3API
	bucket string
}

// NewS3 returns an S3 store in bucket.
func NewS3(client S3API, bucket string) (*S3, error) {
	if client == nil {
		return nil, errors.New("the code store needs an S3 client")
	}
	if !s3name.BucketPattern.MatchString(bucket) {
		return nil, errors.New("the code bucket name is not valid")
	}
	return &S3{client: client, bucket: bucket}, nil
}

// Put implements Objects.
func (s *S3) Put(ctx context.Context, key string, body io.ReadSeeker, size int64) error {
	if err := checkKey(key); err != nil {
		return err
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String("application/zip"),
	})
	return err
}

// Open implements Objects.
func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, err
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		if notFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return out.Body, nil
}

// Exists implements Objects, with a HEAD request.
func (s *S3) Exists(ctx context.Context, key string) (bool, error) {
	if err := checkKey(key); err != nil {
		return false, err
	}
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		if notFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Delete implements Objects. S3 answers a delete of a missing key with
// success.
func (s *S3) Delete(ctx context.Context, key string) error {
	if err := checkKey(key); err != nil {
		return err
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return err
}

// List implements Objects, a page of S3's listing at a time. S3 lists keys in
// byte order, the same order Go compares strings in.
func (s *S3) List(ctx context.Context, after string, limit int) ([]string, error) {
	keys := []string{}
	in := &s3.ListObjectsV2Input{Bucket: aws.String(s.bucket), Prefix: aws.String(Prefix)}
	if after != "" {
		in.StartAfter = aws.String(after)
	}
	for len(keys) < limit {
		in.MaxKeys = aws.Int32(int32(min(limit-len(keys), 1000)))
		out, err := s.client.ListObjectsV2(ctx, in)
		if err != nil {
			return nil, err
		}
		for _, o := range out.Contents {
			if k := aws.ToString(o.Key); ValidKey(k) && k > after && len(keys) < limit {
				keys = append(keys, k)
			}
		}
		if !aws.ToBool(out.IsTruncated) || out.NextContinuationToken == nil {
			break
		}
		in.ContinuationToken, in.StartAfter = out.NextContinuationToken, nil
	}
	return keys, nil
}

func notFound(err error) bool {
	var nsk *s3types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	// A HEAD of a missing key answers 404 with no body, so the SDK names it
	// NotFound.
	var nf *s3types.NotFound
	if errors.As(err, &nf) {
		return true
	}
	var ae smithy.APIError
	return errors.As(err, &ae) && (ae.ErrorCode() == "NoSuchKey" || ae.ErrorCode() == "NotFound")
}
