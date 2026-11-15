// Command api is the adapter API (B API): the adapter layer's one HTTP service
// beside A (docs/architecture/system-b-adapter-layer.md).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/aws"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/generalopenstack"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/api"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/auth"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/codestore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/config"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/credentials"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/examples"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/jobs"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/store"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/jackc/pgx/v5/pgxpool"
)

// available lists every connector this build can offer, by id. ADAPTER_CONNECTORS
// picks which ones run. This composition root is the only place that names
// connectors.
var available = map[string]func(ctx context.Context, cfg config.Config, tf *terraform.Runner, pool *pgxpool.Pool) (connectors.Connector, error){
	mock.ID: func(context.Context, config.Config, *terraform.Runner, *pgxpool.Pool) (connectors.Connector, error) {
		return mock.New(), nil
	},
	generalopenstack.ID: func(_ context.Context, cfg config.Config, tf *terraform.Runner, _ *pgxpool.Pool) (connectors.Connector, error) {
		var opts []generalopenstack.Option
		if cfg.AllowPrivateTargets {
			opts = append(opts, generalopenstack.AllowPrivateTargets())
		}
		return generalopenstack.New(tf, opts...), nil
	},
	aws.ID: newAWS,
}

// newAWS builds the AWS connector. Its templates come from a folder on disk
// (ADAPTER_AWS_TEMPLATE_DIR, the repo's backend/infra/modules/s3/uploads/aws locally)
// or from the platform's template bucket (ADAPTER_AWS_TEMPLATE_BUCKET, in the cloud).
// ADAPTER_AWS_STS is "sts" in the cloud. On a laptop (A2 only, refused unless
// credentials are stored locally) it is "local", which verifies a well-formed role
// and stops every run with "needs a real AWS account", or "localstack" with
// ADAPTER_AWS_ENDPOINT, which applies for real against LocalStack.
func newAWS(ctx context.Context, cfg config.Config, tf *terraform.Runner, pool *pgxpool.Pool) (connectors.Connector, error) {
	var templates aws.TemplateSource
	switch dir, bucket := os.Getenv("ADAPTER_AWS_TEMPLATE_DIR"), os.Getenv("ADAPTER_AWS_TEMPLATE_BUCKET"); {
	case dir != "" && bucket == "":
		loader, err := tftemplate.NewTemplateLoaderFromDir(dir)
		if err != nil {
			return nil, fmt.Errorf("ADAPTER_AWS_TEMPLATE_DIR: %w", err)
		}
		templates = loader
	case bucket != "" && dir == "":
		loader, err := tftemplate.NewTemplateLoader(ctx, bucket)
		if err != nil {
			return nil, fmt.Errorf("ADAPTER_AWS_TEMPLATE_BUCKET: %w", err)
		}
		templates = loader
	default:
		return nil, errors.New("the aws connector needs exactly one of ADAPTER_AWS_TEMPLATE_DIR or ADAPTER_AWS_TEMPLATE_BUCKET")
	}

	var assumer aws.Assumer
	switch os.Getenv("ADAPTER_AWS_STS") {
	case "sts":
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("aws config: %w", err)
		}
		assumer = aws.STSAssumer{Client: sts.NewFromConfig(awsCfg)}
	case "local":
		if cfg.CredentialStore != "local" {
			return nil, errors.New(`ADAPTER_AWS_STS=local is only allowed with ADAPTER_CREDENTIAL_STORE=local`)
		}
		assumer = aws.LocalAssumer{}
	case "localstack":
		if cfg.CredentialStore != "local" {
			return nil, errors.New(`ADAPTER_AWS_STS=localstack is only allowed with ADAPTER_CREDENTIAL_STORE=local`)
		}
		a, err := aws.NewLocalStackAssumer(os.Getenv("ADAPTER_AWS_ENDPOINT"))
		if err != nil {
			return nil, fmt.Errorf("ADAPTER_AWS_ENDPOINT: %w", err)
		}
		assumer = a
	default:
		return nil, errors.New(`the aws connector needs ADAPTER_AWS_STS set to "sts", "local" or "localstack"`)
	}
	endpoint := os.Getenv("ADAPTER_AWS_ENDPOINT")
	if endpoint != "" && os.Getenv("ADAPTER_AWS_STS") != "localstack" {
		return nil, errors.New(`ADAPTER_AWS_ENDPOINT is only allowed with ADAPTER_AWS_STS=localstack`)
	}
	// The linked-roles reader is the one place a connector reads Clutter's
	// database: A's linked AWS roles become one-click connections.
	c := aws.New(tf, templates, assumer).WithLinkedRoles(aws.LinkedRoles{DB: pool})
	if endpoint != "" {
		return c.WithEndpoint(endpoint)
	}
	return c, nil
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(log)
	if err := run(log); err != nil {
		log.Error("adapter api stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	log.Info("starting", "config", cfg.String())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return errors.New("the database string is not valid")
	}
	defer pool.Close()
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err = pool.Ping(pingCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("the database is unreachable: %w", err)
	}

	st := &store.Store{Pool: pool}
	schema, err := os.ReadFile(cfg.SchemaFile)
	if err != nil {
		return err
	}
	if err := st.ApplySchema(ctx, string(schema)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	if n, err := st.FailInterrupted(ctx, jobs.FailInterruptedMessage); err != nil {
		return fmt.Errorf("fail interrupted runs: %w", err)
	} else if n > 0 {
		log.Warn("failed runs a previous stop left behind", "runs", n)
	}

	creds, err := credentialStore(ctx, cfg)
	if err != nil {
		return err
	}
	logs, err := logStore(ctx, cfg)
	if err != nil {
		return err
	}
	code, err := codeStore(ctx, cfg)
	if err != nil {
		return err
	}
	launcher, err := runLauncher(ctx, cfg)
	if err != nil {
		return err
	}

	tf := &terraform.Runner{
		WorkDir:        filepath.Join(cfg.DataDir, "work"),
		StateDir:       filepath.Join(cfg.DataDir, "state"),
		PluginCacheDir: filepath.Join(cfg.DataDir, "plugin-cache"),
		Launcher:       launcher,
		Code:           code,
		PassEnv:        []string{"HTTPS_PROXY", "HTTP_PROXY", "NO_PROXY", "SSL_CERT_FILE", "SSL_CERT_DIR"},
	}
	// The ECS launcher runs Terraform in the run task; B API's image has none.
	if cfg.RunLauncher == config.LauncherLocal {
		if bin, err := exec.LookPath(cfg.TerraformBin); err != nil {
			log.Warn("terraform not found; Terraform runs will fail", "bin", cfg.TerraformBin)
		} else if tf.Bin, err = filepath.Abs(bin); err != nil {
			return err
		}
	}

	reg := connectors.NewRegistry()
	for _, id := range cfg.Connectors {
		build, ok := available[id]
		if !ok {
			return fmt.Errorf("ADAPTER_CONNECTORS names %q, which this build does not have", id)
		}
		c, err := build(ctx, cfg, tf, pool)
		if err != nil {
			return fmt.Errorf("connector %s: %w", id, err)
		}
		if err := reg.Register(c); err != nil {
			return err
		}
	}

	// A broken example fails the start, not the first "New diagram".
	if _, err := examples.All(); err != nil {
		return err
	}

	runner := jobs.New(jobs.Config{
		Registry: reg, Store: st, Broker: broker{creds}, Logs: logs, Logger: log,
		Workers: cfg.Workers, RunTimeout: cfg.RunTimeout,
	})
	runner.Start()

	// Code objects whose rows went with a diagram or an organization deleted
	// in A: once at start-up, then every hour, until B stops.
	sweeper := &codestore.Sweeper{Objects: code, Records: st, Logger: log}
	go sweeper.Run(ctx, codestore.SweepEvery)

	verifier := auth.JWTVerifier{Secret: cfg.JWTSecret}
	a := &api.API{Store: st, Registry: reg, Credentials: creds, Jobs: runner, Logs: logs, Verifier: verifier, Logger: log, AllowedOrigins: cfg.AllowedOrigins, MaxRunsPerOrg: cfg.MaxRunsPerOrg,
		Code: code, CodeTempDir: filepath.Join(cfg.DataDir, "uploads")}
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           a.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.ListenAddr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	log.Info("stopping")
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("http shutdown", "err", err)
	}
	if err := runner.Stop(shutCtx); err != nil {
		log.Error("job runner shutdown", "err", err)
	}
	return nil
}

func credentialStore(ctx context.Context, cfg config.Config) (credentials.Store, error) {
	switch cfg.CredentialStore {
	case config.StoreLocal:
		return credentials.NewLocal(filepath.Join(cfg.DataDir, "secrets"), cfg.CredentialKey)
	case config.StoreSecretsManager:
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("aws config: %w", err)
		}
		return credentials.SecretsManager{Client: secretsmanager.NewFromConfig(awsCfg), Prefix: cfg.SecretsPrefix}, nil
	}
	return nil, fmt.Errorf("unknown credential store %q", cfg.CredentialStore)
}

// logStore is where run logs live: files under ADAPTER_DATA_DIR, or the S3 bucket
// ADAPTER_LOG_BUCKET in the cloud, where B's disk is lost when its task stops.
func logStore(ctx context.Context, cfg config.Config) (logstore.Store, error) {
	switch cfg.LogStore {
	case config.LogStoreLocal:
		return logstore.NewDisk(filepath.Join(cfg.DataDir, "logs"))
	case config.LogStoreS3:
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("aws config: %w", err)
		}
		return logstore.NewS3(s3.NewFromConfig(awsCfg), cfg.LogBucket)
	}
	return nil, fmt.Errorf("unknown log store %q", cfg.LogStore)
}

// codeStore is where uploaded function code lives. It follows the log store:
// files under ADAPTER_DATA_DIR locally, and objects under code/ in the same bucket
// (ADAPTER_LOG_BUCKET, the run bucket) in the cloud, which no lifecycle rule
// expires.
func codeStore(ctx context.Context, cfg config.Config) (codestore.Objects, error) {
	switch cfg.LogStore {
	case config.LogStoreLocal:
		return codestore.NewDisk(filepath.Join(cfg.DataDir, "code"))
	case config.LogStoreS3:
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("aws config: %w", err)
		}
		return codestore.NewS3(s3.NewFromConfig(awsCfg), cfg.LogBucket)
	}
	return nil, fmt.Errorf("unknown log store %q", cfg.LogStore)
}

// runLauncher is how runs start: a child process per command, or one ECS task
// per run with the settings runtask.LoadECSSettings reads.
func runLauncher(ctx context.Context, cfg config.Config) (runtask.Launcher, error) {
	switch cfg.RunLauncher {
	case config.LauncherLocal:
		return runtask.Local{}, nil
	case config.LauncherECS:
		settings, err := runtask.LoadECSSettings(os.Getenv)
		if err != nil {
			return nil, fmt.Errorf("ADAPTER_RUN_LAUNCHER=ecs: %w", err)
		}
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(settings.Region))
		if err != nil {
			return nil, fmt.Errorf("aws config: %w", err)
		}
		return &runtask.ECS{
			Settings: settings,
			ECS:      ecs.NewFromConfig(awsCfg),
			S3:       s3.NewFromConfig(awsCfg),
			STS:      sts.NewFromConfig(awsCfg),
		}, nil
	}
	return nil, fmt.Errorf("unknown run launcher %q", cfg.RunLauncher)
}

// broker hands a run its connection's credentials from the credential store.
type broker struct{ store credentials.Store }

func (b broker) Credentials(ctx context.Context, ref string) (connectors.Secrets, error) {
	return b.store.Get(ctx, ref)
}
