// Command runtask is B's run task: the program in the short-lived Fargate
// task that runs one Terraform run (ADR-017, internal/runtask's ECS launcher).
//
// It reads the run's files and spec from the run bucket with the run's own
// S3 keys, runs each step with only the run's environment, keeps state in the
// S3 backend B wrote, and uploads the log and the result. It has no role, no
// database and no secret store. Terraform's output goes only to the run log
// object, which B redacts; this program's own output is short status lines.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// Where the image puts things (backend/adapter/images/runtask.Dockerfile).
const (
	terraformPath = "/usr/local/bin/terraform"
	workDir       = "/work"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Getenv))
}

func run(ctx context.Context, getenv func(string) string) int {
	runID := getenv(runtask.EnvRunID)
	bucket := getenv(runtask.EnvRunBucket)
	region := getenv(runtask.EnvRegion)
	if _, err := uuid.Parse(runID); err != nil || bucket == "" || region == "" {
		fmt.Fprintln(os.Stderr, "runtask: the run id, bucket or region is missing")
		return exitSetup
	}
	w := &worker{
		Bucket:          bucket,
		RunID:           runID,
		AccessKeyID:     getenv(runtask.EnvAccessKeyID),
		SecretAccessKey: getenv(runtask.EnvSecretAccessKey),
		SessionToken:    getenv(runtask.EnvSessionToken),
		Terraform:       terraformPath,
		WorkDir:         workDir,
		LogEvery:        5 * time.Second,
		// Under the task definition's 120-second stop timeout.
		Grace:    100 * time.Second,
		Status:   os.Stdout,
		Resolver: net.DefaultResolver,
	}
	// Only the run's keys, never a default credential chain: the task has no
	// role, and the run's environment may hold a customer's AWS keys.
	w.S3 = s3.New(s3.Options{
		Region:      region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(w.AccessKeyID, w.SecretAccessKey, w.SessionToken)),
	})
	return w.Run(ctx)
}
