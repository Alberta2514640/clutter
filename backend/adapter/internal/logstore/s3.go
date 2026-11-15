package logstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/s3name"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3Prefix starts every log's object key: a log with key k is the object
// S3Prefix + k. B API's role may read and write only under it.
const S3Prefix = "logs/"

// S3API is the part of the S3 client the S3 store uses.
type S3API interface {
	PutObject(ctx context.Context, in *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, in *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// S3 keeps logs as objects in one bucket, for the cloud, where B's disk is
// lost when its task stops. S3 cannot append, so a log being written is held
// in memory and the whole object is uploaded every Every while it grows, and
// once more when it is closed. Open in the same process reads the log in
// memory, which is newer than the object.
type S3 struct {
	client S3API
	bucket string
	// Every is how often a growing log is uploaded. Zero means 3 seconds.
	Every time.Duration

	mu   sync.Mutex
	live map[string]*s3Log
}

// NewS3 returns an S3 store in bucket.
func NewS3(client S3API, bucket string) (*S3, error) {
	if client == nil {
		return nil, errors.New("the log store needs an S3 client")
	}
	if !s3name.BucketPattern.MatchString(bucket) {
		return nil, errors.New("the log bucket name is not valid")
	}
	return &S3{client: client, bucket: bucket, live: map[string]*s3Log{}}, nil
}

// Create implements Store. It uploads an empty log at once, so a missing or
// unwritable bucket fails the run's start, not its end.
func (s *S3) Create(key string) (io.WriteCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, err
	}
	l := &s3Log{s: s, key: key, done: make(chan struct{}), stopped: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.put(ctx, key, nil); err != nil {
		return nil, fmt.Errorf("start run log: %w", err)
	}
	s.mu.Lock()
	s.live[key] = l
	s.mu.Unlock()
	every := s.Every
	if every <= 0 {
		every = 3 * time.Second
	}
	go l.uploadEvery(every)
	return l, nil
}

// Open implements Store.
func (s *S3) Open(key string) (io.ReadCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, err
	}
	s.mu.Lock()
	l, ok := s.live[key]
	s.mu.Unlock()
	if ok {
		return io.NopCloser(bytes.NewReader(l.snapshot())), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(S3Prefix + key)})
	if err != nil {
		var nsk *s3types.NoSuchKey
		var ae smithy.APIError
		if errors.As(err, &nsk) || (errors.As(err, &ae) && ae.ErrorCode() == "NoSuchKey") {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read run log: %w", err)
	}
	// Read it whole, so the caller's copy does not hold the connection open
	// while it writes the response.
	defer out.Body.Close()
	body, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read run log: %w", err)
	}
	return io.NopCloser(bytes.NewReader(body)), nil
}

func (s *S3) put(ctx context.Context, key string, body []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(S3Prefix + key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("text/plain; charset=utf-8"),
	})
	return err
}

// s3Log is one log being written.
type s3Log struct {
	s   *S3
	key string

	mu     sync.Mutex
	buf    bytes.Buffer
	sent   int // length of the last upload
	closed bool

	// upMu keeps uploads in order, so an older body never replaces a newer one.
	upMu    sync.Mutex
	done    chan struct{}
	stopped chan struct{}
}

func (l *s3Log) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return 0, errors.New("the run log is closed")
	}
	return l.buf.Write(p)
}

func (l *s3Log) snapshot() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	return bytes.Clone(l.buf.Bytes())
}

// upload sends the whole log when it has grown since the last upload.
func (l *s3Log) upload(ctx context.Context) error {
	l.upMu.Lock()
	defer l.upMu.Unlock()
	l.mu.Lock()
	if l.buf.Len() == l.sent {
		l.mu.Unlock()
		return nil
	}
	body := bytes.Clone(l.buf.Bytes())
	l.mu.Unlock()
	if err := l.s.put(ctx, l.key, body); err != nil {
		return err
	}
	l.mu.Lock()
	l.sent = len(body)
	l.mu.Unlock()
	return nil
}

// uploadEvery uploads the log every d until Close. A failed upload is tried
// again at the next tick; Close reports the last one's error.
func (l *s3Log) uploadEvery(d time.Duration) {
	defer close(l.stopped)
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-l.done:
			return
		case <-t.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = l.upload(ctx)
			cancel()
		}
	}
}

// Close uploads the rest of the log. The log stays readable from memory until
// that upload ends, then from the object.
func (l *s3Log) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()
	close(l.done)
	<-l.stopped
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := l.upload(ctx)
	l.s.mu.Lock()
	if l.s.live[l.key] == l {
		delete(l.s.live, l.key)
	}
	l.s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("upload run log: %w", err)
	}
	return nil
}
