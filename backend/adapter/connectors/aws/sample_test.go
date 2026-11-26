package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// linkedEnv is what a function linked to one table and two buckets gets
// (functionEnv), with a value the samples must not echo.
var linkedEnv = []string{
	"CLUTTER_ORDERS_TABLE=c1234abcd_orders",
	"CLUTTER_ORDERS_REGION=ca-central-1",
	"CLUTTER_FILES_BUCKET=c1234abcd-files-0a1b",
	"CLUTTER_LOGS_BUCKET=c1234abcd-logs-9f8e",
	"CLUTTER_BAD_TABLE=has\"quote",
}

// wantLinked is the body a sample answers with under linkedEnv.
var wantLinked = map[string]any{
	"message": sampleMessage,
	"table":   "c1234abcd_orders",
	"buckets": []any{"c1234abcd-files-0a1b", "c1234abcd-logs-9f8e"},
}

// wantAnswer checks one answer is what API Gateway's proxy integration needs:
// status 200 and a JSON body with the sample message and, when want is set,
// exactly want.
func wantAnswer(t *testing.T, raw string, want map[string]any) {
	t.Helper()
	var a struct {
		StatusCode int               `json:"statusCode"`
		Headers    map[string]string `json:"headers"`
		Body       string            `json:"body"`
	}
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("the answer is not JSON: %v\n%s", err, raw)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(a.Body), &body); err != nil {
		t.Fatalf("the answer's body is not JSON: %v\n%s", err, a.Body)
	}
	if a.StatusCode != 200 || a.Headers["Content-Type"] != "application/json" || body["message"] != sampleMessage {
		t.Fatalf("answer = %+v, body %v", a, body)
	}
	// The shell sample lists names in the environment's order.
	for _, k := range []string{"tables", "buckets"} {
		if list, ok := body[k].([]any); ok {
			sort.Slice(list, func(i, j int) bool { return fmt.Sprint(list[i]) < fmt.Sprint(list[j]) })
		}
	}
	if want != nil && !reflect.DeepEqual(body, want) {
		t.Fatalf("body = %v, want %v", body, want)
	}
}

func TestEveryOfferedRuntimeHasASampleForItsDefaultHandler(t *testing.T) {
	for _, rt := range runtimes() {
		files, ok := sampleCode(rt, generic.LambdaRuntimes[rt])
		if !ok || len(files) != 1 {
			t.Fatalf("%s: no sample for its default handler", rt)
		}
		if !strings.Contains(files[0].Content, "Clutter's sample code") {
			t.Errorf("%s: the sample does not say what it is", rt)
		}
	}
	// A custom runtime starts its bootstrap whatever the handler.
	if files, ok := sampleCode("provided.al2023", "anything"); !ok || files[0].Name != "bootstrap" || !strings.HasPrefix(files[0].Content, "#!/bin/sh\n") {
		t.Fatalf("provided.al2023 sample = %+v, %v", files, ok)
	}
	if files, ok := sampleCode("python3.12", "app.main"); !ok || files[0].Name != "app.py" || !strings.Contains(files[0].Content, "def main(event, context):") {
		t.Fatalf("python sample for app.main = %+v, %v", files, ok)
	}
	if files, ok := sampleCode("nodejs22.x", "index.handler"); !ok || files[0].Name != "index.mjs" || !strings.Contains(files[0].Content, "export const handler = ") {
		t.Fatalf("node sample = %+v, %v", files, ok)
	}
	for _, h := range []string{"src/app.main", "example.Handler::handleRequest", "main", "a.b.c"} {
		if _, ok := sampleCode("python3.12", h); ok {
			t.Errorf("a sample for the handler %q, which it cannot serve", h)
		}
	}
}

// The shell sample runs as a custom runtime: it takes one event from a fake
// runtime API and posts the sample answer for that request.
func TestBootstrapSampleAnswersThroughTheRuntimeAPI(t *testing.T) {
	for _, tool := range []string{"sh", "curl", "mktemp", "grep"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not on this machine", tool)
		}
	}
	files, _ := sampleCode("provided.al2023", "bootstrap")
	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap")
	if err := os.WriteFile(path, []byte(files[0].Content), 0o755); err != nil {
		t.Fatal(err)
	}

	answered := make(chan string, 1)
	served := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/2018-06-01/runtime/invocation/next" && !served:
			served = true
			w.Header().Set("Lambda-Runtime-Aws-Request-Id", "req-1")
			_, _ = io.WriteString(w, `{"path":"/items"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/2018-06-01/runtime/invocation/req-1/response":
			b, _ := io.ReadAll(r.Body)
			answered <- string(b)
		default:
			// The next event never comes; the test stops the runtime.
			<-r.Context().Done()
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", path)
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), "AWS_LAMBDA_RUNTIME_API=" + strings.TrimPrefix(srv.URL, "http://")}, linkedEnv...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	select {
	case got := <-answered:
		wantAnswer(t, got, wantLinked)
	case <-ctx.Done():
		t.Fatal("the bootstrap sample posted no answer")
	}
}

// The Python sample, run by python3 when the machine has it, returns the
// sample answer.
func TestPythonSampleAnswers(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not on this machine")
	}
	files, _ := sampleCode("python3.12", "lambda_function.lambda_handler")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, files[0].Name), []byte(files[0].Content), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, env := range [][]string{nil, linkedEnv} {
		cmd := exec.Command("python3", "-I", "-c", "import sys, json; sys.path.insert(0, sys.argv[1]); import lambda_function as f; print(json.dumps(f.lambda_handler({}, None)))", dir)
		cmd.Env = append([]string{"PATH=" + os.Getenv("PATH")}, env...)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("python3: %v", err)
		}
		want := map[string]any{"message": sampleMessage}
		if env != nil {
			want = wantLinked
		}
		wantAnswer(t, strings.TrimSpace(string(out)), want)
	}
}

// The Node.js sample, run by node when the machine has it, returns the
// sample answer.
func TestNodeSampleAnswers(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not on this machine")
	}
	files, _ := sampleCode("nodejs22.x", "index.handler")
	dir := t.TempDir()
	path := filepath.Join(dir, files[0].Name)
	if err := os.WriteFile(path, []byte(files[0].Content), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", "--input-type=module", "-e", "const m = await import(process.argv[1]); console.log(JSON.stringify(await m.handler()));", "file://"+path)
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH")}, linkedEnv...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("node: %v", err)
	}
	wantAnswer(t, strings.TrimSpace(string(out)), wantLinked)
}

func TestSampleCodeRendersExecutableAndReportsItself(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	files := renderFor(t, c, fullGraph(), d)
	code := string(files["code.tf"])
	for _, want := range []string{`output_file_mode = "0755"`, `filename = "lambda_function.py"`, `def lambda_handler(event, context):`} {
		if !strings.Contains(code, want) {
			t.Errorf("code.tf lacks %q:\n%s", want, code)
		}
	}
	results := resultsFor(t, c, fullGraph(), d)
	if !strings.Contains(results, `output "result_code_fn" {`) || !strings.Contains(results, `value = "sample"`) {
		t.Errorf("results.tf does not say the function runs the sample:\n%s", results)
	}

	// The default runtime's sample is a bootstrap, its ${...} escaped for HCL.
	d.Settings = nil
	code = string(renderFor(t, c, fullGraph(), d)["code.tf"])
	if !strings.Contains(code, `filename = "bootstrap"`) || !strings.Contains(code, `$${AWS_LAMBDA_RUNTIME_API}`) {
		t.Errorf("the default runtime's sample is not an escaped bootstrap:\n%s", code)
	}

	// No sample for a handler Clutter cannot serve: a file that says so, and
	// no claim that the function answers.
	g := fullGraph()
	g.Blocks["fn"].Fields[connectors.FieldHandler] = "example.Handler::handleRequest"
	d.Settings = map[string]any{"function_runtime": "python3.12"}
	files = renderFor(t, c, g, d)
	if !strings.Contains(string(files["code.tf"]), `filename = "welcome.txt"`) {
		t.Errorf("code.tf for a handler with no sample:\n%s", files["code.tf"])
	}
	if strings.Contains(resultsFor(t, c, g, d), "result_code_fn") {
		t.Error("a function with no sample reports that it runs one")
	}

	// Uploaded code: no sample, and no claim.
	if strings.Contains(resultsFor(t, c, withCodeAndScript(uploaded(t, nil, testZip(t))), d), "result_code_") {
		t.Error("a function with uploaded code reports that it runs the sample")
	}
}

// resultsFor is the results.tf a run of g for d holds.
func resultsFor(t *testing.T, c *Connector, g connectors.Graph, d connectors.Deployment) string {
	t.Helper()
	files, err := c.render(context.Background(), newModel(g, d))
	if err != nil {
		t.Fatal(err)
	}
	return string(files["results.tf"])
}

func TestAdviceNamesTheNATGatewayCost(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	notes := c.Advice(fullGraph(), d)
	if len(notes) != 1 || notes[0].NodeID != "net1" || notes[0].Code != connectors.CodeCost || !strings.Contains(notes[0].Message, "$40 a month") {
		t.Fatalf("advice = %+v", notes)
	}
	g := fullGraph()
	g.Blocks["net1"].Fields["private_internet"] = false
	if notes := c.Advice(g, d); len(notes) != 0 {
		t.Fatalf("advice without a NAT gateway = %+v", notes)
	}
}
