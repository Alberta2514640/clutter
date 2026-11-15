// Package runtasktest holds an in-memory S3 for the run launcher's and the
// run task's tests.
package runtasktest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3 is an in-memory bucket set that implements runtask.S3API.
type S3 struct {
	mu      sync.Mutex
	objects map[string][]byte // "bucket/key"
	// Deleted lists every "bucket/key" deleted, in order.
	Deleted []string
	// FailPut, when set, fails PutObject for keys with this suffix.
	FailPut string
	// OnPut, when set, sees every object put, before it is stored.
	OnPut func(bucket, key string, body []byte)
}

// New returns an empty S3.
func New() *S3 { return &S3{objects: map[string][]byte{}} }

// Set stores an object.
func (m *S3) Set(bucket, key string, body []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[bucket+"/"+key] = append([]byte(nil), body...)
}

// Object returns an object and whether it exists.
func (m *S3) Object(bucket, key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.objects[bucket+"/"+key]
	return b, ok
}

// Keys lists every "bucket/key", sorted.
func (m *S3) Keys() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.objects))
	for k := range m.objects {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func apiErr(code string) error { return &smithy.GenericAPIError{Code: code, Message: code} }

// PutObject implements runtask.S3API.
func (m *S3) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.FailPut != "" && strings.HasSuffix(aws.ToString(in.Key), m.FailPut) {
		return nil, apiErr("AccessDenied")
	}
	body, err := io.ReadAll(in.Body)
	if err != nil {
		return nil, err
	}
	if m.OnPut != nil {
		m.OnPut(aws.ToString(in.Bucket), aws.ToString(in.Key), body)
	}
	m.Set(aws.ToString(in.Bucket), aws.ToString(in.Key), body)
	return &s3.PutObjectOutput{}, nil
}

// GetObject implements runtask.S3API, with "bytes=N-" ranges.
func (m *S3) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	body, ok := m.Object(aws.ToString(in.Bucket), aws.ToString(in.Key))
	if !ok {
		return nil, apiErr("NoSuchKey")
	}
	if r := aws.ToString(in.Range); r != "" {
		n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(r, "bytes="), "-"))
		if err != nil {
			return nil, fmt.Errorf("range %q", r)
		}
		if n >= len(body) {
			return nil, apiErr("InvalidRange")
		}
		body = body[n:]
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(body))}, nil
}

// ListObjectsV2 implements runtask.S3API, in one page.
func (m *S3) ListObjectsV2(_ context.Context, in *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	want := aws.ToString(in.Bucket) + "/" + aws.ToString(in.Prefix)
	out := &s3.ListObjectsV2Output{IsTruncated: aws.Bool(false)}
	for _, k := range m.Keys() {
		if strings.HasPrefix(k, want) {
			out.Contents = append(out.Contents, s3types.Object{Key: aws.String(strings.TrimPrefix(k, aws.ToString(in.Bucket)+"/"))})
		}
	}
	return out, nil
}

// DeleteObject implements runtask.S3API.
func (m *S3) DeleteObject(_ context.Context, in *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := aws.ToString(in.Bucket) + "/" + aws.ToString(in.Key)
	delete(m.objects, k)
	m.Deleted = append(m.Deleted, k)
	return &s3.DeleteObjectOutput{}, nil
}
