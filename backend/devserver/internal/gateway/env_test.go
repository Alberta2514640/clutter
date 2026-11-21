package gateway

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The log-in Lambda accepts only Google tokens minted for the client id it is
// given, so the devserver must give it the one the frontend signs in with.
// Locally that is the NEXT_PUBLIC_GOOGLE_CLIENT_ID dev.sh starts the frontend
// with, and the frontend must read that value rather than hold its own.
func TestGoogleClientIDMatchesTheFrontend(t *testing.T) {
	devSh, err := os.ReadFile(filepath.Join("..", "..", "dev.sh"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`NEXT_PUBLIC_GOOGLE_CLIENT_ID="([^"]+)"`).FindAllSubmatch(devSh, -1)
	if len(m) != 1 {
		t.Fatalf("dev.sh sets NEXT_PUBLIC_GOOGLE_CLIENT_ID %d times; want once", len(m))
	}
	if got := string(m[0][1]); got != FrontendGoogleClientID {
		t.Fatalf("dev.sh passes %s to the frontend, but the log-in Lambda gets %s; update FrontendGoogleClientID in env.go", got, FrontendGoogleClientID)
	}

	provider := filepath.Join("..", "..", "..", "..", "frontend", "clutter", "src", "providers", "GoogleAuthProvider.tsx")
	src, err := os.ReadFile(provider)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID") {
		t.Fatalf("%s does not read NEXT_PUBLIC_GOOGLE_CLIENT_ID", provider)
	}
	if strings.Contains(string(src), ".apps.googleusercontent.com") {
		t.Fatalf("%s holds a client id of its own; it must read NEXT_PUBLIC_GOOGLE_CLIENT_ID only", provider)
	}

	got, err := lambdaEnvValue("GOOGLE_CLIENT_ID", Config{})
	if err != nil || got != FrontendGoogleClientID {
		t.Fatalf("GOOGLE_CLIENT_ID = %q, %v", got, err)
	}
}

// Real AWS from this machine (real-aws.sh): the account-link flow names
// Clutter's real account and template only when both are given, in their
// exact forms; otherwise it keeps the local stand-ins.
func TestRealAWSAccountAndTemplate(t *testing.T) {
	const account = "123456789012"
	const tmpl = "https://clutter-dev-templates-123456789012-ca-central-1.s3.ca-central-1.amazonaws.com/templates/cloudformation/client_side_terraform_deployer_role.yaml"
	base := Config{
		JWTSecret:      LocalSecretPrefix + strings.Repeat("a", 64),
		PSQLConnString: "postgres://u:p@127.0.0.1:54329/clutter?sslmode=disable",
		AWSEndpoint:    "http://127.0.0.1:9",
		HomeDir:        "/tmp/x",
	}
	for key, want := range map[string]string{"CLUTTER_ACCOUNT_ID": localAccountID, "CLOUDFORMATION_TEMPLATE_URL": "https://devserver.invalid/client_side_terraform_deployer_role.yaml"} {
		if got, err := lambdaEnvValue(key, base); err != nil || got != want {
			t.Fatalf("%s without real AWS = %q (%v), want %q", key, got, err, want)
		}
	}

	real := base
	real.ClutterAccountID, real.TemplateURL = account, tmpl
	if err := ValidateConfig(real); err != nil {
		t.Fatalf("real AWS config refused: %v", err)
	}
	for key, want := range map[string]string{"CLUTTER_ACCOUNT_ID": account, "CLOUDFORMATION_TEMPLATE_URL": tmpl} {
		if got, err := lambdaEnvValue(key, real); err != nil || got != want {
			t.Fatalf("%s with real AWS = %q (%v), want %q", key, got, err, want)
		}
	}

	for name, c := range map[string]Config{
		"account alone":       func() Config { c := base; c.ClutterAccountID = account; return c }(),
		"template alone":      func() Config { c := base; c.TemplateURL = tmpl; return c }(),
		"short account":       func() Config { c := real; c.ClutterAccountID = "12345"; return c }(),
		"plain http template": func() Config { c := real; c.TemplateURL = strings.Replace(tmpl, "https://", "http://", 1); return c }(),
		"another host": func() Config {
			c := real
			c.TemplateURL = "https://example.com/templates/cloudformation/client_side_terraform_deployer_role.yaml"
			return c
		}(),
		"another file": func() Config { c := real; c.TemplateURL = strings.Replace(tmpl, "client_side", "other", 1); return c }(),
	} {
		if err := ValidateConfig(c); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
