package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func env(t *testing.T, over map[string]string) func(string) string {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	schema := filepath.Join(t.TempDir(), "adapter-tables.sql")
	if err := os.WriteFile(schema, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := map[string]string{
		"ADAPTER_LISTEN_ADDR":      "127.0.0.1:8790",
		"ADAPTER_DATABASE_URL":     "postgres://u:db-password-123@localhost/db",
		"ADAPTER_SCHEMA_FILE":      schema,
		"ADAPTER_DATA_DIR":         t.TempDir(),
		"JWT_SECRET":               "jwt-secret-value-9",
		"ADAPTER_CREDENTIAL_STORE": "local",
		"ADAPTER_CREDENTIAL_KEY":   base64.StdEncoding.EncodeToString(key),
	}
	for k, v := range over {
		m[k] = v
	}
	return func(k string) string { return m[k] }
}

func TestLoad(t *testing.T) {
	c, err := Load(env(t, map[string]string{"ADAPTER_CONNECTORS": " mock , "}))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.CredentialKey) != 32 || c.Workers != 2 || len(c.Connectors) != 1 || c.Connectors[0] != "mock" {
		t.Fatalf("loaded %v", c)
	}
	for _, out := range []string{fmt.Sprint(c), fmt.Sprintf("%+v %#v", c, c)} {
		if strings.Contains(out, "db-password-123") || strings.Contains(out, "jwt-secret-value-9") {
			t.Fatal("config prints a secret")
		}
	}
}

func TestLoadRefusesMissingValues(t *testing.T) {
	for _, name := range []string{"ADAPTER_LISTEN_ADDR", "ADAPTER_DATABASE_URL", "ADAPTER_SCHEMA_FILE", "ADAPTER_DATA_DIR", "JWT_SECRET", "ADAPTER_CREDENTIAL_STORE", "ADAPTER_CREDENTIAL_KEY"} {
		if _, err := Load(env(t, map[string]string{name: ""})); err == nil || !strings.Contains(err.Error(), name) {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	if _, err := Load(env(t, map[string]string{"ADAPTER_CREDENTIAL_STORE": "secretsmanager"})); err == nil {
		t.Fatal("secretsmanager without a prefix was accepted")
	}
	if _, err := Load(env(t, map[string]string{"ADAPTER_CREDENTIAL_KEY": "c2hvcnQ="})); err == nil || strings.Contains(err.Error(), "c2hvcnQ=") {
		t.Fatalf("short key: %v", err)
	}
	if _, err := Load(env(t, map[string]string{"ADAPTER_DATA_DIR": "relative/dir"})); err == nil {
		t.Fatal("a relative data dir was accepted")
	}
}

func TestLoadAllowedOrigins(t *testing.T) {
	c, err := Load(env(t, nil))
	if err != nil || c.AllowedOrigins != nil {
		t.Fatalf("unset: %v %v", c.AllowedOrigins, err)
	}
	c, err = Load(env(t, map[string]string{"ADAPTER_ALLOWED_ORIGINS": " http://localhost:3000 ,http://127.0.0.1:3000,, https://app.example.com"}))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"http://localhost:3000", "http://127.0.0.1:3000", "https://app.example.com"}
	if fmt.Sprint(c.AllowedOrigins) != fmt.Sprint(want) {
		t.Fatalf("origins %v", c.AllowedOrigins)
	}
	for _, bad := range []string{"*", "https://*.example.com", "http://localhost:3000/", "http://localhost:3000/app",
		"localhost:3000", "ftp://example.com", "https://user@example.com", "https://example.com?x=1", "null"} {
		if _, err := Load(env(t, map[string]string{"ADAPTER_ALLOWED_ORIGINS": "http://localhost:3000," + bad})); err == nil {
			t.Errorf("origin %q was accepted", bad)
		}
	}
}

func TestLoadPrivateTargetsOnlyLocally(t *testing.T) {
	c, err := Load(env(t, nil))
	if err != nil || c.AllowPrivateTargets {
		t.Fatalf("unset: %v %v", c.AllowPrivateTargets, err)
	}
	if c, err := Load(env(t, map[string]string{"ADAPTER_ALLOW_PRIVATE_TARGETS": "1"})); err != nil || !c.AllowPrivateTargets {
		t.Fatalf("local: %v %v", c.AllowPrivateTargets, err)
	}
	cloud := map[string]string{"ADAPTER_ALLOW_PRIVATE_TARGETS": "1", "ADAPTER_CREDENTIAL_STORE": "secretsmanager", "ADAPTER_SECRETS_PREFIX": "clutter/adapter/"}
	if _, err := Load(env(t, cloud)); err == nil || !strings.Contains(err.Error(), "ADAPTER_ALLOW_PRIVATE_TARGETS") {
		t.Fatalf("with Secrets Manager: %v", err)
	}
	if _, err := Load(env(t, map[string]string{"ADAPTER_ALLOW_PRIVATE_TARGETS": "yes"})); err == nil {
		t.Fatal("a value other than 1 was accepted")
	}
}

func TestLoadMaxRunsPerOrg(t *testing.T) {
	if c, err := Load(env(t, nil)); err != nil || c.MaxRunsPerOrg != 0 {
		t.Fatalf("unset: %d %v", c.MaxRunsPerOrg, err)
	}
	if c, err := Load(env(t, map[string]string{"ADAPTER_MAX_RUNS_PER_ORG": "5"})); err != nil || c.MaxRunsPerOrg != 5 {
		t.Fatalf("five: %d %v", c.MaxRunsPerOrg, err)
	}
	for _, bad := range []string{"0", "-1", "x", "1001"} {
		if _, err := Load(env(t, map[string]string{"ADAPTER_MAX_RUNS_PER_ORG": bad})); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestLoadRunLauncherAndLogStore(t *testing.T) {
	c, err := Load(env(t, nil))
	if err != nil || c.RunLauncher != LauncherLocal || c.LogStore != LogStoreLocal || c.LogBucket != "" {
		t.Fatalf("unset: %q %q %q %v", c.RunLauncher, c.LogStore, c.LogBucket, err)
	}
	c, err = Load(env(t, map[string]string{"ADAPTER_RUN_LAUNCHER": "ecs", "ADAPTER_LOG_STORE": "s3", "ADAPTER_LOG_BUCKET": "clutter-adapter-runs-x"}))
	if err != nil || c.RunLauncher != LauncherECS || c.LogStore != LogStoreS3 || c.LogBucket != "clutter-adapter-runs-x" {
		t.Fatalf("cloud: %q %q %q %v", c.RunLauncher, c.LogStore, c.LogBucket, err)
	}
	if !strings.Contains(c.String(), "launcher=ecs") || !strings.Contains(c.String(), "logs=s3") {
		t.Fatalf("String() %s", c)
	}
	for name, over := range map[string]map[string]string{
		"ADAPTER_RUN_LAUNCHER": {"ADAPTER_RUN_LAUNCHER": "fargate"},
		"ADAPTER_LOG_STORE":    {"ADAPTER_LOG_STORE": "dynamodb"},
		"ADAPTER_LOG_BUCKET":   {"ADAPTER_LOG_STORE": "s3"},
	} {
		if _, err := Load(env(t, over)); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%v: %v", over, err)
		}
	}
	if _, err := Load(env(t, map[string]string{"ADAPTER_LOG_STORE": "s3", "ADAPTER_LOG_BUCKET": "Not_A_Bucket"})); err == nil ||
		strings.Contains(err.Error(), "Not_A_Bucket") {
		t.Fatalf("bad bucket: %v", err)
	}
}
