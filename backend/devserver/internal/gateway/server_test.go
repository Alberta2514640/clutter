package gateway

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda/messages"
)

// fakeInvoker records proxy events and plays the authorizer and handlers.
type fakeInvoker struct {
	allow    bool
	lastFn   string
	lastProx events.APIGatewayProxyRequest
	authHits int
	fnErr    *messages.InvokeResponse_Error
}

func (f *fakeInvoker) Invoke(_ context.Context, fn string, payload []byte) ([]byte, *messages.InvokeResponse_Error, error) {
	if fn == "authorizer" {
		f.authHits++
		var ev events.APIGatewayCustomAuthorizerRequestTypeRequest
		_ = json.Unmarshal(payload, &ev)
		effect, principal := "Deny", "Unauthorized"
		var ctx map[string]any
		if f.allow {
			effect, principal = "Allow", "user-1"
			ctx = map[string]any{"id": "user-1", "email": "a@b.c"}
		}
		out, _ := json.Marshal(events.APIGatewayCustomAuthorizerResponse{
			PrincipalID: principal,
			PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
				Version:   "2012-10-17",
				Statement: []events.IAMPolicyStatement{{Action: []string{"execute-api:Invoke"}, Effect: effect, Resource: []string{ev.MethodArn}}},
			},
			Context: ctx,
		})
		return out, nil, nil
	}
	f.lastFn = fn
	_ = json.Unmarshal(payload, &f.lastProx)
	if f.fnErr != nil {
		return nil, f.fnErr, nil
	}
	out, _ := json.Marshal(events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers:    map[string]string{"Access-Control-Allow-Origin": "*", "Content-Type": "application/json"},
		Body:       `{"ok":true}`,
	})
	return out, nil, nil
}

func serve(t *testing.T, f *fakeInvoker, method, target, auth, body string) *httptest.ResponseRecorder {
	t.Helper()
	s := NewServer(f, log.New(io.Discard, "", 0), nil)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func TestAuthorizedRouteTranslatesEvent(t *testing.T) {
	f := &fakeInvoker{allow: true}
	rec := serve(t, f, "POST", "/organization/11111111-1111-4111-8111-111111111111/accounts/22222222-2222-4222-8222-222222222222?x=1&x=2", "tok", `{"role_arn":"a"}`)
	if rec.Code != 201 || rec.Body.String() != `{"ok":true}` {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if f.lastFn != "organization-submit-account-role-arn" {
		t.Fatalf("routed to %q", f.lastFn)
	}
	ev := f.lastProx
	if ev.Resource != "/organization/{organizationId}/accounts/{accountId}" {
		t.Errorf("resource %q", ev.Resource)
	}
	if ev.PathParameters["organizationId"] != "11111111-1111-4111-8111-111111111111" || ev.PathParameters["accountId"] != "22222222-2222-4222-8222-222222222222" {
		t.Errorf("path params %v", ev.PathParameters)
	}
	if ev.QueryStringParameters["x"] != "2" || len(ev.MultiValueQueryStringParameters["x"]) != 2 {
		t.Errorf("query %v %v", ev.QueryStringParameters, ev.MultiValueQueryStringParameters)
	}
	if ev.Body != `{"role_arn":"a"}` || ev.HTTPMethod != "POST" {
		t.Errorf("body %q method %q", ev.Body, ev.HTTPMethod)
	}
	if ev.RequestContext.Authorizer["id"] != "user-1" || ev.RequestContext.Authorizer["principalId"] != "user-1" {
		t.Errorf("authorizer context %v", ev.RequestContext.Authorizer)
	}
}

func TestMissingAuthorizationIs401WithoutCallingAuthorizer(t *testing.T) {
	f := &fakeInvoker{allow: true}
	rec := serve(t, f, "GET", "/organization", "", "")
	if rec.Code != 401 || f.authHits != 0 || f.lastFn != "" {
		t.Fatalf("got %d, authorizer calls %d, handler %q", rec.Code, f.authHits, f.lastFn)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("gateway errors carry no CORS headers in AWS")
	}
}

func TestDenyIs403(t *testing.T) {
	f := &fakeInvoker{allow: false}
	rec := serve(t, f, "GET", "/organization", "tok", "")
	if rec.Code != 403 || f.lastFn != "" {
		t.Fatalf("got %d, handler %q", rec.Code, f.lastFn)
	}
}

func TestPublicRouteSkipsAuthorizer(t *testing.T) {
	f := &fakeInvoker{}
	rec := serve(t, f, "GET", "/resources", "", "")
	if rec.Code != 201 || f.authHits != 0 || f.lastFn != "resources-get" {
		t.Fatalf("got %d, authorizer calls %d, handler %q", rec.Code, f.authHits, f.lastFn)
	}
	if f.lastProx.PathParameters != nil || f.lastProx.QueryStringParameters != nil {
		t.Errorf("empty params must be null, as in API Gateway")
	}
}

func TestLiteralBeatsParam(t *testing.T) {
	f := &fakeInvoker{allow: true}
	serve(t, f, "GET", "/terraform-engine/logs/recent-activity", "tok", "")
	if f.lastFn != "terraform-command-runner-logs-recent-activity" {
		t.Fatalf("routed to %q", f.lastFn)
	}
	serve(t, f, "GET", "/ansible/jobs/abc/logs", "tok", "")
	if f.lastFn != "ansible-get-job-logs" || f.lastProx.PathParameters["jobId"] != "abc" {
		t.Fatalf("routed to %q with %v", f.lastFn, f.lastProx.PathParameters)
	}
}

func TestOptionsPreflight(t *testing.T) {
	f := &fakeInvoker{}
	rec := serve(t, f, "OPTIONS", "/organization/abc", "", "")
	if rec.Code != 200 || rec.Header().Get("Access-Control-Allow-Methods") != "PUT,DELETE,OPTIONS" || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("got %d %v", rec.Code, rec.Header())
	}
	if f.authHits != 0 || f.lastFn != "" {
		t.Fatalf("preflight must not invoke anything")
	}
	// The bare /cloudformation-stack-url resource has no OPTIONS method.
	if rec := serve(t, f, "OPTIONS", "/cloudformation-stack-url", "", ""); rec.Code != 403 {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestUnknownRouteAndMethod(t *testing.T) {
	f := &fakeInvoker{allow: true}
	if rec := serve(t, f, "GET", "/nope", "tok", ""); rec.Code != 403 || !strings.Contains(rec.Body.String(), "Missing Authentication Token") {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if rec := serve(t, f, "PATCH", "/project", "tok", ""); rec.Code != 403 {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestFunctionErrorIs502(t *testing.T) {
	f := &fakeInvoker{allow: true, fnErr: &messages.InvokeResponse_Error{Message: "boom", Type: "panic"}}
	rec := serve(t, f, "GET", "/organization", "tok", "")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestValidateConfig(t *testing.T) {
	good := Config{
		JWTSecret:      LocalSecretPrefix + strings.Repeat("a", 64),
		PSQLConnString: "postgres://u:p@127.0.0.1:54329/clutter?sslmode=disable",
		AWSEndpoint:    "http://127.0.0.1:9",
		HomeDir:        "/tmp/x",
	}
	if err := ValidateConfig(good); err != nil {
		t.Fatalf("good config refused: %v", err)
	}
	bad := good
	bad.JWTSecret = strings.Repeat("a", 64)
	if ValidateConfig(bad) == nil {
		t.Error("accepted a secret without the local prefix")
	}
	bad = good
	bad.PSQLConnString = "postgres://u:p@db.example.supabase.co:5432/postgres"
	if ValidateConfig(bad) == nil {
		t.Error("accepted a remote Postgres")
	}
	for _, addr := range []string{"0.0.0.0:8787", ":8787", "192.168.1.5:8787"} {
		if RequireLoopbackAddr(addr) == nil {
			t.Errorf("accepted listen address %s", addr)
		}
	}
	if err := RequireLoopbackAddr("127.0.0.1:8787"); err != nil {
		t.Errorf("refused loopback: %v", err)
	}
}

func TestProcessEnvIsClosed(t *testing.T) {
	c := Config{JWTSecret: "s", PSQLConnString: "p", FrontendOrigin: "o", AWSEndpoint: "http://127.0.0.1:9", HomeDir: "/h"}
	env, err := processEnv("x", []string{"PSQL_CONNECTION_STRING"}, "1234", c)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(env, "\n")
	for _, want := range []string{"_LAMBDA_SERVER_PORT=1234", "AWS_ENDPOINT_URL=http://127.0.0.1:9", "AWS_CONFIG_FILE=/dev/null", "PSQL_CONNECTION_STRING=p"} {
		if !strings.Contains(joined, want) {
			t.Errorf("env lacks %s", want)
		}
	}
	if strings.Contains(joined, "JWT_SECRET") {
		t.Error("a Lambda got a variable main.tf does not give it")
	}
	if _, err := processEnv("x", []string{"UNKNOWN_VAR"}, "1", c); err == nil {
		t.Error("an unknown variable must fail start-up")
	}
}
