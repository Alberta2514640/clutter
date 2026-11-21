package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/netguard"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// exitSetup is the exit code when the run could not be prepared or reported,
// so Terraform's own codes stay distinct.
const exitSetup = 70

// worker runs one run inside the run task.
type worker struct {
	S3     runtask.S3API
	Bucket string
	RunID  string
	// AccessKeyID, SecretAccessKey and SessionToken are the run's S3 keys,
	// given to the state backend at init so they win over any AWS keys in
	// the run's own environment.
	AccessKeyID, SecretAccessKey, SessionToken string
	// Terraform is the absolute path of the terraform binary.
	Terraform string
	// WorkDir is an empty folder the task owns.
	WorkDir string
	// LogEvery is how often the log is uploaded while Terraform runs.
	LogEvery time.Duration
	// Grace is how long Terraform has to stop after an interrupt.
	Grace time.Duration
	// Status receives the task's own short status lines (CloudWatch). It
	// never receives Terraform's output.
	Status io.Writer
	// Resolver looks up the run's public hosts (RunSpec.PublicHosts).
	Resolver netguard.Resolver
}

// Run prepares the run, runs its steps, uploads the log and the result, and
// returns the task's exit code.
func (w *worker) Run(ctx context.Context) int {
	keys := runtask.KeysFor(w.RunID)
	runDir := filepath.Join(w.WorkDir, "run")
	logPath := filepath.Join(w.WorkDir, "log.txt")
	for _, d := range []string{runDir, filepath.Join(w.WorkDir, "plugin-cache")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			fmt.Fprintf(w.Status, "prepare: %v\n", err)
			return exitSetup
		}
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		fmt.Fprintf(w.Status, "prepare: %v\n", err)
		return exitSetup
	}
	defer logFile.Close()

	up := &logUploader{w: w, key: keys.Log, path: logPath}
	stopUploads := up.every(ctx, w.LogEvery)

	res := w.steps(ctx, keys, runDir, logFile)

	stopUploads()
	// The run's context may have ended; the last upload gets its own.
	finalCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	code := res.ExitCode
	if err := up.upload(finalCtx); err != nil {
		fmt.Fprintf(w.Status, "upload log: %v\n", err)
		code = exitSetup
	}
	body, _ := json.Marshal(res)
	if err := w.put(finalCtx, keys.Result, body); err != nil {
		fmt.Fprintf(w.Status, "upload result: %v\n", err)
		code = exitSetup
	}
	fmt.Fprintf(w.Status, "run %s ended: step=%q exit=%d\n", w.RunID, res.Step, code)
	return code
}

// steps prepares the folder and runs each step, stopping at the first failure.
func (w *worker) steps(ctx context.Context, keys runtask.RunKeys, runDir string, log *os.File) runtask.Result {
	fail := func(msg string) runtask.Result {
		// Plain words only: these lines reach the user's run log.
		fmt.Fprintf(log, "clutter: %s\n", msg)
		fmt.Fprintf(w.Status, "prepare: %s\n", msg)
		return runtask.Result{Step: runtask.PrepareStep, ExitCode: exitSetup}
	}
	// task.env was read by ECS before the task started; it is not needed again.
	if _, err := w.S3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &w.Bucket, Key: aws.String(keys.TaskEnv)}); err != nil {
		fmt.Fprintf(w.Status, "delete task keys: %v\n", err)
	}
	specBody, err := w.get(ctx, keys.Spec)
	if err != nil {
		return fail("the run could not be read")
	}
	if _, err := w.S3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &w.Bucket, Key: aws.String(keys.Spec)}); err != nil {
		fmt.Fprintf(w.Status, "delete run spec: %v\n", err)
	}
	var spec runtask.RunSpec
	if err := json.Unmarshal(specBody, &spec); err != nil {
		return fail("the run is damaged")
	}
	if err := runtask.CheckSteps(spec.Steps); err != nil {
		return fail("the run asks for a command that is not allowed")
	}
	// A cloud address the connection gave (an OpenStack auth URL) must be
	// public now, before Terraform connects to it. This catches a name that
	// resolves inside Clutter's network at the start of the run; a name whose
	// DNS changes during the run is stopped by the run subnets' network ACL
	// (backend/infra/modules/adapter-service/network.tf), not here.
	for _, host := range spec.PublicHosts {
		if err := netguard.CheckHost(ctx, w.Resolver, host); err != nil {
			if errors.Is(err, netguard.ErrBlockedHost) {
				return fail("the cloud's address resolves to a private, loopback, link-local or metadata address, which Clutter does not connect to")
			}
			return fail("the cloud's address could not be looked up")
		}
	}
	if err := w.download(ctx, keys.FilesPrefix, runDir); err != nil {
		return fail(err.Error())
	}
	backendConfig := filepath.Join(w.WorkDir, "state.s3.tfbackend")
	if err := w.writeBackendConfig(backendConfig); err != nil {
		return fail("the run's state keys are not valid")
	}

	// The run spec was built for B's machine. A value inside B's run folder
	// (HOME, TF_CLI_CONFIG_FILE) points at this task's copy instead, and the
	// paths outside it point at this task's own.
	env := map[string]string{}
	for k, v := range spec.Env {
		env[k] = remapDir(v, spec.Dir, runDir)
	}
	env["PATH"] = "/usr/local/bin:/usr/bin:/bin"
	env["TF_PLUGIN_CACHE_DIR"] = filepath.Join(w.WorkDir, "plugin-cache")
	envList, err := runtask.EnvList(env)
	if err != nil {
		return fail("the run's environment is not valid")
	}

	for _, step := range spec.Steps {
		if ctx.Err() != nil {
			return runtask.Result{Step: step[0], ExitCode: 130}
		}
		args := append([]string(nil), step...)
		if step[0] == "init" {
			args = append(args, "-backend-config="+backendConfig)
		}
		cmd := exec.CommandContext(ctx, w.Terraform, args...)
		cmd.Dir = runDir
		cmd.Env = envList
		cmd.Stdin = nil
		cmd.Stdout = log
		cmd.Stderr = log
		// The output step's stdout holds sensitive values: it goes to its
		// own object, never to the log B copies to the user.
		var outputs *capped
		if step[0] == runtask.OutputStep {
			outputs = &capped{max: runtask.MaxOutputs}
			cmd.Stdout = outputs
		}
		cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
		cmd.WaitDelay = w.Grace
		if err := cmd.Run(); err != nil {
			code := 1
			var ee *exec.ExitError
			if errors.As(err, &ee) && ee.ExitCode() > 0 {
				code = ee.ExitCode()
			}
			return runtask.Result{Step: step[0], ExitCode: code}
		}
		if outputs != nil {
			if outputs.over {
				fmt.Fprintln(log, "clutter: the Terraform outputs are too large")
				return runtask.Result{Step: step[0], ExitCode: exitSetup}
			}
			if err := w.put(ctx, keys.Outputs, outputs.buf.Bytes()); err != nil {
				fmt.Fprintf(w.Status, "upload outputs: %v\n", err)
				return runtask.Result{Step: step[0], ExitCode: exitSetup}
			}
		}
	}
	return runtask.Result{}
}

// capped keeps at most max bytes and notes when more came.
type capped struct {
	buf  bytes.Buffer
	max  int
	over bool
}

func (c *capped) Write(p []byte) (int, error) {
	if c.buf.Len()+len(p) > c.max {
		c.over = true
		return len(p), nil
	}
	return c.buf.Write(p)
}

// remapDir rewrites v when it is from, or a path inside it, to the same path
// under to. Any other value is returned as it is.
func remapDir(v, from, to string) string {
	if from == "" || !filepath.IsAbs(from) {
		return v
	}
	from = filepath.Clean(from)
	switch {
	case v == from:
		return to
	case strings.HasPrefix(v, from+string(filepath.Separator)):
		return filepath.Join(to, strings.TrimPrefix(v, from+string(filepath.Separator)))
	}
	return v
}

// download copies the run folder's files into dir. Only plain names are
// accepted, so no object can write outside the folder.
func (w *worker) download(ctx context.Context, prefix, dir string) error {
	keys, err := runtask.ListKeys(ctx, w.S3, w.Bucket, prefix)
	if err != nil {
		return errors.New("the run files could not be listed")
	}
	if len(keys) == 0 {
		return errors.New("the run has no files")
	}
	for _, k := range keys {
		name := strings.TrimPrefix(k, prefix)
		if !runtask.FileNamePattern.MatchString(name) {
			return fmt.Errorf("the run file name %q is not allowed", name)
		}
		body, err := w.get(ctx, k)
		if err != nil {
			return errors.New("the run files could not be read")
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			return errors.New("the run files could not be written")
		}
	}
	return nil
}

// writeBackendConfig writes the run's S3 keys for terraform init. The file
// is outside the run folder, and nothing uploads it.
func (w *worker) writeBackendConfig(path string) error {
	for _, v := range []string{w.AccessKeyID, w.SecretAccessKey, w.SessionToken} {
		if !runtask.ValidCredential(v) {
			return errors.New("not valid")
		}
	}
	body := fmt.Sprintf("access_key = %q\nsecret_key = %q\ntoken      = %q\n", w.AccessKeyID, w.SecretAccessKey, w.SessionToken)
	return os.WriteFile(path, []byte(body), 0o600)
}

func (w *worker) get(ctx context.Context, key string) ([]byte, error) {
	out, err := w.S3.GetObject(ctx, &s3.GetObjectInput{Bucket: &w.Bucket, Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (w *worker) put(ctx context.Context, key string, body []byte) error {
	_, err := w.S3.PutObject(ctx, &s3.PutObjectInput{Bucket: &w.Bucket, Key: aws.String(key), Body: bytes.NewReader(body)})
	return err
}

// logUploader uploads the whole log when it has grown, so B can copy the new
// part into the run's live log.
type logUploader struct {
	w    *worker
	key  string
	path string

	mu   sync.Mutex
	sent int64
}

func (u *logUploader) upload(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	body, err := os.ReadFile(u.path)
	if err != nil {
		return err
	}
	if int64(len(body)) == u.sent && u.sent > 0 {
		return nil
	}
	if err := u.w.put(ctx, u.key, body); err != nil {
		return err
	}
	u.sent = int64(len(body))
	return nil
}

// every uploads the log every d until the returned function is called.
func (u *logUploader) every(ctx context.Context, d time.Duration) (stop func()) {
	if d <= 0 {
		d = 5 * time.Second
	}
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		t := time.NewTicker(d)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				if err := u.upload(ctx); err != nil && ctx.Err() == nil {
					fmt.Fprintf(u.w.Status, "upload log: %v\n", err)
				}
			}
		}
	}()
	return func() { close(done); <-finished }
}
