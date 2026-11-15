// Package config reads B's settings from the environment and checks them at
// start-up, so a missing value fails the start, not the first request.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/credentials"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/s3name"
)

// Credential store kinds.
const (
	StoreLocal          = "local"
	StoreSecretsManager = "secretsmanager"
)

// Run launchers (ADAPTER_RUN_LAUNCHER): a child process per command on B's own
// machine, or one ECS task per run.
const (
	LauncherLocal = "local"
	LauncherECS   = "ecs"
)

// Log stores (ADAPTER_LOG_STORE): files under ADAPTER_DATA_DIR, or objects in a bucket.
const (
	LogStoreLocal = "local"
	LogStoreS3    = "s3"
)

// Config is B's settings. It never prints its secrets.
type Config struct {
	ListenAddr  string
	DatabaseURL string
	SchemaFile  string
	DataDir     string

	CredentialStore string
	// CredentialKey is the local store's AES key.
	CredentialKey []byte
	// SecretsPrefix starts each Secrets Manager secret name.
	SecretsPrefix string
	// JWTSecret verifies sign-in tokens; A's log-in signs them with it.
	JWTSecret []byte

	TerraformBin string
	// RunLauncher is LauncherLocal or LauncherECS. The ECS launcher's own
	// settings are read by runtask.LoadECSSettings.
	RunLauncher string
	// LogStore is LogStoreLocal or LogStoreS3; LogBucket is the S3 store's
	// bucket.
	LogStore   string
	LogBucket  string
	Connectors []string
	Workers    int
	RunTimeout time.Duration
	// AllowedOrigins are the browser origins CORS answers, matched exactly.
	// Empty means no CORS headers at all.
	AllowedOrigins []string
	// AllowPrivateTargets lets a connector reach loopback and private
	// addresses over plain http while it verifies a connection, for a
	// laptop's DevStack. Only allowed with the local credential store, so the
	// cloud can never set it.
	AllowPrivateTargets bool
	// MaxRunsPerOrg caps one organization's queued and running runs, so one
	// organization cannot fill the queue for the others. Zero means the API's
	// default (api.DefaultMaxRunsPerOrg).
	MaxRunsPerOrg int
}

// String hides the database string and the key.
func (c Config) String() string {
	return fmt.Sprintf("listen=%s schema=%s data=%s store=%s launcher=%s logs=%s log_bucket=%s connectors=%v workers=%d runs_per_org=%d private_targets=%t origins=%v",
		c.ListenAddr, c.SchemaFile, c.DataDir, c.CredentialStore, c.RunLauncher, c.LogStore, c.LogBucket, c.Connectors, c.Workers, c.MaxRunsPerOrg, c.AllowPrivateTargets, c.AllowedOrigins)
}

// GoString hides the secrets from %#v.
func (c Config) GoString() string { return c.String() }

// Load reads the environment through getenv (os.Getenv in main).
//
// Required: ADAPTER_LISTEN_ADDR, ADAPTER_DATABASE_URL, ADAPTER_SCHEMA_FILE, ADAPTER_DATA_DIR,
// JWT_SECRET (the same secret A's log-in signs with), ADAPTER_CREDENTIAL_STORE,
// and ADAPTER_CREDENTIAL_KEY (local) or ADAPTER_SECRETS_PREFIX
// (secretsmanager). Optional: ADAPTER_TERRAFORM_BIN (default "terraform" on PATH),
// ADAPTER_RUN_LAUNCHER ("local", the default, or "ecs"), ADAPTER_LOG_STORE ("local", the
// default, or "s3", which needs ADAPTER_LOG_BUCKET),
// ADAPTER_CONNECTORS (comma-separated ids), ADAPTER_WORKERS (default 2), ADAPTER_RUN_TIMEOUT
// (default 1h), ADAPTER_ALLOWED_ORIGINS (comma-separated browser origins such as
// http://localhost:3000; no wildcard), ADAPTER_MAX_RUNS_PER_ORG (unset means the
// API's default, 3), and
// ADAPTER_ALLOW_PRIVATE_TARGETS=1 (A2 only; refused unless ADAPTER_CREDENTIAL_STORE is
// local). Errors name the variable, never its value.
//
// Load is not the only reader. The aws connector's four settings
// (ADAPTER_AWS_TEMPLATE_DIR, ADAPTER_AWS_TEMPLATE_BUCKET, ADAPTER_AWS_STS and
// ADAPTER_AWS_ENDPOINT) are read and checked by newAWS in cmd/api/main.go, and
// the ECS launcher's by runtask.LoadECSSettings.
func Load(getenv func(string) string) (Config, error) {
	var errs []error
	req := func(name string) string {
		v := strings.TrimSpace(getenv(name))
		if v == "" {
			errs = append(errs, fmt.Errorf("%s is not set", name))
		}
		return v
	}
	c := Config{
		ListenAddr:      req("ADAPTER_LISTEN_ADDR"),
		DatabaseURL:     req("ADAPTER_DATABASE_URL"),
		SchemaFile:      req("ADAPTER_SCHEMA_FILE"),
		DataDir:         req("ADAPTER_DATA_DIR"),
		JWTSecret:       []byte(req("JWT_SECRET")),
		CredentialStore: req("ADAPTER_CREDENTIAL_STORE"),
		TerraformBin:    strings.TrimSpace(getenv("ADAPTER_TERRAFORM_BIN")),
		Workers:         2,
		RunTimeout:      time.Hour,
	}
	switch c.CredentialStore {
	case StoreLocal:
		if raw := req("ADAPTER_CREDENTIAL_KEY"); raw != "" {
			key, err := credentials.ParseKey(raw)
			if err != nil {
				errs = append(errs, fmt.Errorf("ADAPTER_CREDENTIAL_KEY: %w", err))
			}
			c.CredentialKey = key
		}
	case StoreSecretsManager:
		c.SecretsPrefix = req("ADAPTER_SECRETS_PREFIX")
	case "":
	default:
		errs = append(errs, fmt.Errorf("ADAPTER_CREDENTIAL_STORE must be %q or %q", StoreLocal, StoreSecretsManager))
	}
	if c.DataDir != "" && !filepath.IsAbs(c.DataDir) {
		errs = append(errs, errors.New("ADAPTER_DATA_DIR must be an absolute path"))
	}
	if c.TerraformBin == "" {
		c.TerraformBin = "terraform"
	}
	switch c.RunLauncher = strings.TrimSpace(getenv("ADAPTER_RUN_LAUNCHER")); c.RunLauncher {
	case "":
		c.RunLauncher = LauncherLocal
	case LauncherLocal, LauncherECS:
	default:
		errs = append(errs, fmt.Errorf("ADAPTER_RUN_LAUNCHER must be %q or %q", LauncherLocal, LauncherECS))
	}
	switch c.LogStore = strings.TrimSpace(getenv("ADAPTER_LOG_STORE")); c.LogStore {
	case "":
		c.LogStore = LogStoreLocal
	case LogStoreLocal:
	case LogStoreS3:
		switch c.LogBucket = req("ADAPTER_LOG_BUCKET"); {
		case c.LogBucket == "":
		case !s3name.BucketPattern.MatchString(c.LogBucket):
			errs = append(errs, errors.New("ADAPTER_LOG_BUCKET is not a valid bucket name"))
		}
	default:
		errs = append(errs, fmt.Errorf("ADAPTER_LOG_STORE must be %q or %q", LogStoreLocal, LogStoreS3))
	}
	for _, id := range strings.Split(getenv("ADAPTER_CONNECTORS"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			c.Connectors = append(c.Connectors, id)
		}
	}
	if v := strings.TrimSpace(getenv("ADAPTER_WORKERS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 64 {
			errs = append(errs, errors.New("ADAPTER_WORKERS must be a number from 1 to 64"))
		}
		c.Workers = n
	}
	if v := strings.TrimSpace(getenv("ADAPTER_RUN_TIMEOUT")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < time.Minute {
			errs = append(errs, errors.New("ADAPTER_RUN_TIMEOUT must be a duration of at least 1m"))
		}
		c.RunTimeout = d
	}
	if v := strings.TrimSpace(getenv("ADAPTER_MAX_RUNS_PER_ORG")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 1000 {
			errs = append(errs, errors.New("ADAPTER_MAX_RUNS_PER_ORG must be a number from 1 to 1000"))
		}
		c.MaxRunsPerOrg = n
	}
	switch strings.TrimSpace(getenv("ADAPTER_ALLOW_PRIVATE_TARGETS")) {
	case "", "0":
	case "1":
		if c.CredentialStore != StoreLocal {
			errs = append(errs, fmt.Errorf("ADAPTER_ALLOW_PRIVATE_TARGETS=1 is only allowed with ADAPTER_CREDENTIAL_STORE=%s", StoreLocal))
		}
		c.AllowPrivateTargets = true
	default:
		errs = append(errs, errors.New("ADAPTER_ALLOW_PRIVATE_TARGETS must be 1 or unset"))
	}
	for _, o := range strings.Split(getenv("ADAPTER_ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o == "" {
			continue
		}
		if !isOrigin(o) {
			errs = append(errs, errors.New("ADAPTER_ALLOWED_ORIGINS must list origins like https://app.example.com, with no path and no wildcard"))
			break
		}
		c.AllowedOrigins = append(c.AllowedOrigins, o)
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	if _, err := os.Stat(c.SchemaFile); err != nil {
		return Config{}, errors.New("ADAPTER_SCHEMA_FILE does not name a readable file")
	}
	return c, nil
}

// isOrigin reports whether s is exactly scheme://host[:port], the form a
// browser sends in its Origin header.
func isOrigin(s string) bool {
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || strings.Contains(u.Host, "*") {
		return false
	}
	return u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == "" && !u.ForceQuery && u.Scheme+"://"+u.Host == s
}
