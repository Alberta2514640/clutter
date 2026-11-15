package gateway

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// NoAWS is the endpoint every Lambda's AWS SDK is pointed at. S3 calls the
// local store can serve go to it. Every other call gets a non-retryable
// access-denied error in the protocol the caller speaks, and is recorded, so a
// dev run shows exactly where a route needs AWS without any request leaving
// the machine.
type NoAWS struct {
	logger *log.Logger
	s3     *LocalS3
	mu     sync.Mutex
	// calls holds the latest refused calls, at most limit of them. Call
	// indexes count from the gateway's start: calls[0] is call number
	// dropped, so an index stays the same after older calls are dropped.
	calls   []string
	dropped int
	limit   int
}

// maxRecordedCalls caps the refused calls a long dev session keeps.
const maxRecordedCalls = 10000

// NewNoAWS returns an empty recorder in front of the local S3 store.
func NewNoAWS(logger *log.Logger, s3 *LocalS3) *NoAWS {
	return &NoAWS{logger: logger, s3: s3, limit: maxRecordedCalls}
}

// record keeps op, dropping the oldest call when the record is full.
func (n *NoAWS) record(op string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.calls) >= n.limit {
		drop := len(n.calls) - n.limit + 1
		n.calls = append(n.calls[:0], n.calls[drop:]...)
		n.dropped += drop
	}
	n.calls = append(n.calls, op)
}

func (n *NoAWS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if n.s3.handles(r) {
		n.s3.ServeHTTP(w, r)
		return
	}
	target := r.Header.Get("X-Amz-Target")
	op := target
	if op == "" {
		// S3 is REST: name the call by method and bucket only.
		bucket := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)[0]
		op = "S3 " + r.Method + " " + bucket
	}
	n.record(op)
	n.logger.Printf("[no-aws] refused AWS call %s %s", op, r.URL.Path)

	const msg = "the devserver has no AWS; this call was refused locally"
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-amz-json") || target != "" {
		// JSON protocols (ECS, CloudWatch Logs, SQS).
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.Header().Set("X-Amzn-ErrorType", "AccessDeniedException")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"__type":"AccessDeniedException","message":"` + msg + `"}`))
		return
	}
	// REST-XML protocol (S3).
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>AccessDenied</Code><Message>` + msg + `</Message></Error>`))
}

// CallsHandler serves the refused calls recorded after index ?since=N, so a
// test can tell which AWS APIs one request tried. next is the index the next
// call gets. A since older than the oldest call kept starts at that call.
func (n *NoAWS) CallsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		since, _ := strconv.Atoi(r.URL.Query().Get("since"))
		n.mu.Lock()
		next := n.dropped + len(n.calls)
		switch {
		case since < 0 || since > next:
			since = next
		case since < n.dropped:
			since = n.dropped
		}
		out := struct {
			Next  int      `json:"next"`
			Calls []string `json:"calls"`
		}{Next: next, Calls: append([]string{}, n.calls[since-n.dropped:]...)}
		n.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}
