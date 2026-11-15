package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &buf
}

func TestRecordFailureStopsAtAMissingID(t *testing.T) {
	// Past the guard, the log line would be about the S3 writer.
	logged := captureLog(t)
	recordFailure(context.Background(), "", generic.DiagramRecord{ID: testDiagram, ProjectID: testProject}, "failed")
	if !strings.Contains(logged.String(), "an id is missing") {
		t.Fatalf("log: %q", logged.String())
	}
}

func TestATemplateFailureIsLoggedWithItsCause(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error>`))
	}))
	defer server.Close()
	t.Setenv("AWS_ENDPOINT_URL", server.URL)
	t.Setenv("AWS_ACCESS_KEY_ID", "test-only")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-only")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("TEMPLATE_BUCKET_NAME", "templates-bucket")
	logged := captureLog(t)

	// The request names only the ids; the S3 node comes from the database row.
	mock := useMock(t)
	expectOwnership(mock, testOrg, true)
	expectDiagramRow(mock, `{"nodes":[{"id":"a","data":{"label":"S3","variables":{"resource_name":"uploads"}}}],"edges":[]}`)

	res, err := handler(context.Background(), signedIn(map[string]string{"projectId": testProject, "diagramId": testDiagram}))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	if !strings.Contains(logged.String(), "templates/aws/s3/main.tf.tmpl") {
		t.Fatalf("log: %q", logged.String())
	}
	// The body names neither the bucket path nor the SDK error.
	if strings.Contains(res.Body, "templates/") || strings.Contains(res.Body, "AccessDenied") {
		t.Fatalf("the body leaks the cause: %s", res.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
