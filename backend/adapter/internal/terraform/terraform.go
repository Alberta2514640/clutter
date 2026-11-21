// Package terraform is the shared Terraform runner connectors hand rendered HCL to.
// It keeps state per deployment behind a lock, refuses dangerous Terraform
// before plan, and runs Terraform through an isolated run task (ADR-017).
package terraform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/codestore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// Job is one Terraform run of one deployment.
type Job struct {
	DeploymentID string
	RunID        string
	Action       connectors.Action
	// Files are the rendered .tf files, by file name. For a destroy they need
	// only the provider configuration.
	Files map[string][]byte
	// Archives are uploaded function code zips the runner stages into the
	// run's folder, each as ArchiveName(ID). They are the only files a run
	// holds besides Files and the runner's own, and the runner names them:
	// rendered text never does.
	Archives []Archive
	// Providers are the provider source addresses the connector renders for,
	// such as "hashicorp/aws". CheckFiles refuses any other source, and the
	// run's Terraform CLI configuration installs only these.
	Providers []string
	// Env holds the run's credentials as the provider reads them. A name that
	// starts with TF_ is refused unless it starts with TF_VAR_.
	Env map[string]string
	// PublicHosts are host names or addresses the run reaches that must be
	// public: an isolated run task (a RunLauncher) resolves each before
	// Terraform starts and refuses the run when one resolves to a private,
	// loopback, link-local or metadata address (netguard.CheckHost). A run on
	// B's own machine does not check them; the connector checks before it
	// renders.
	PublicHosts []string
	Log         io.Writer
	// Outputs, when set on a plan or an apply, receives `terraform output
	// -json` after the plan or apply: the outputs as the state holds them,
	// sensitive values included. It never reaches Log.
	Outputs io.Writer
}

// Runner runs Terraform jobs.
type Runner struct {
	// Bin is the absolute path of the terraform binary.
	Bin string
	// WorkDir holds one folder per run, removed when the run ends.
	WorkDir string
	// StateDir holds one folder per deployment with its state file.
	StateDir string
	// PluginCacheDir is shared by local runs so providers download once.
	// Terraform does not promise the cache is safe for two installs at once,
	// so local runs take initLock for init, and use a cached provider without
	// downloading it again (TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE), so
	// a run's plan or apply never reads a provider another run is rewriting.
	// The run task in the cloud keeps a cache of its own per run.
	PluginCacheDir string
	Launcher       runtask.Launcher
	// Code is where uploaded function code is kept. A job with archives needs it.
	Code codestore.Objects
	// PassEnv names variables of B's own environment a run may see, such as a
	// proxy. They must hold no secret.
	PassEnv []string

	mu       sync.Mutex
	locks    map[string]chan struct{}
	initLock chan struct{}
}

// registryHost is the only registry a run installs providers from.
const registryHost = "registry.terraform.io"

var (
	fileNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}\.tf$`)
	// providerPart is one part of a provider source: its namespace or type.
	providerPart = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
)

// ErrRefused is wrapped by every error CheckFiles and CheckEnv return.
var ErrRefused = errors.New("refused")

// NormalizeProvider returns a provider source address in its full form,
// "registry.terraform.io/namespace/type", lower-cased. It refuses any other
// registry host and any part that is not a plain name.
func NormalizeProvider(source string) (string, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(source)), "/")
	switch len(parts) {
	case 2:
		parts = append([]string{registryHost}, parts...)
	case 3:
		if parts[0] != registryHost {
			return "", fmt.Errorf("%w: provider host %q", ErrRefused, parts[0])
		}
	default:
		return "", fmt.Errorf("%w: provider source %q", ErrRefused, source)
	}
	if !providerPart.MatchString(parts[1]) || !providerPart.MatchString(parts[2]) {
		return "", fmt.Errorf("%w: provider source %q", ErrRefused, source)
	}
	return strings.Join(parts, "/"), nil
}

// allowedSet normalizes the connector's provider list.
func allowedSet(providers []string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, p := range providers {
		n, err := NormalizeProvider(p)
		if err != nil {
			return nil, err
		}
		out[n] = true
	}
	return out, nil
}

// CheckFiles parses every rendered file as Terraform's native syntax and
// refuses, by block type and labels: a provisioner anywhere, the "external"
// and "http" data sources, a module (which could bring either), a backend or
// cloud block (the runner sets the state), and a required_providers entry
// whose source is not in providers. It also refuses any file that is not a
// plainly named .tf file (JSON syntax, for one) and any file that does not
// parse. Text inside a string is never read as a block, so a description that
// says "module" is not refused.
func CheckFiles(files map[string][]byte, providers []string) error {
	if len(files) == 0 {
		return fmt.Errorf("%w: no Terraform files", ErrRefused)
	}
	allowed, err := allowedSet(providers)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !fileNamePattern.MatchString(name) {
			return fmt.Errorf("%w: file name %q", ErrRefused, name)
		}
		f, diags := hclsyntax.ParseConfig(files[name], name, hcl.InitialPos)
		if diags.HasErrors() {
			return fmt.Errorf("%w: %s does not parse", ErrRefused, name)
		}
		body, ok := f.Body.(*hclsyntax.Body)
		if !ok {
			return fmt.Errorf("%w: %s does not parse", ErrRefused, name)
		}
		if err := checkBody(name, body, true, allowed); err != nil {
			return err
		}
	}
	return nil
}

// checkBody walks one body's blocks. top is true for a file's own body.
func checkBody(file string, body *hclsyntax.Body, top bool, allowed map[string]bool) error {
	for _, b := range body.Blocks {
		label := func(i int) string {
			if i < len(b.Labels) {
				return b.Labels[i]
			}
			return ""
		}
		switch {
		case b.Type == "provisioner", b.Type == "dynamic" && label(0) == "provisioner":
			return fmt.Errorf("%w: %s holds a provisioner", ErrRefused, file)
		case top && b.Type == "data" && (label(0) == "external" || label(0) == "http"):
			return fmt.Errorf("%w: %s holds the %q data source", ErrRefused, file, label(0))
		case top && b.Type == "module":
			return fmt.Errorf("%w: %s holds a module", ErrRefused, file)
		case top && b.Type == "terraform":
			if err := checkTerraformBlock(file, b.Body, allowed); err != nil {
				return err
			}
		}
		if err := checkBody(file, b.Body, false, allowed); err != nil {
			return err
		}
	}
	return nil
}

// checkTerraformBlock refuses a backend or cloud block and any provider
// source outside the allow-list.
func checkTerraformBlock(file string, body *hclsyntax.Body, allowed map[string]bool) error {
	for _, b := range body.Blocks {
		switch b.Type {
		case "backend", "cloud":
			return fmt.Errorf("%w: %s holds a %s block", ErrRefused, file, b.Type)
		case "required_providers":
			if len(b.Body.Blocks) > 0 {
				return fmt.Errorf("%w: %s has a block inside required_providers", ErrRefused, file)
			}
			for name, attr := range b.Body.Attributes {
				source, err := providerSource(name, attr.Expr)
				if err != nil {
					return fmt.Errorf("%w: %s: provider %q: %w", ErrRefused, file, name, err)
				}
				n, err := NormalizeProvider(source)
				if err != nil {
					return err
				}
				if !allowed[n] {
					return fmt.Errorf("%w: %s requires provider %q, which this cloud does not use", ErrRefused, file, n)
				}
			}
		}
	}
	return nil
}

// providerSource reads one required_providers entry's source. An entry with
// no source, or the old form that gives only a version string, means
// hashicorp/<name>, as Terraform reads it.
func providerSource(name string, expr hclsyntax.Expression) (string, error) {
	obj, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		if _, ok := plainString(expr); !ok {
			return "", errors.New("is not an object or a version string")
		}
		return "hashicorp/" + name, nil
	}
	for _, item := range obj.Items {
		k, ok := plainString(item.KeyExpr)
		if !ok {
			return "", errors.New("has a key that is not a plain name")
		}
		if k != "source" {
			continue
		}
		v, ok := plainString(item.ValueExpr)
		if !ok {
			return "", errors.New("has a source that is not plain text")
		}
		return v, nil
	}
	return "hashicorp/" + name, nil
}

// plainString evaluates expr with no variables and returns it when it is a
// known string. A reference or a function call does not evaluate.
func plainString(expr hcl.Expression) (string, bool) {
	v, diags := expr.Value(nil)
	if diags.HasErrors() || !v.IsWhollyKnown() || v.IsNull() || !v.Type().Equals(cty.String) {
		return "", false
	}
	return v.AsString(), true
}

// CheckEnv refuses a run variable that would change how Terraform runs (any
// TF_ name but TF_VAR_), such as TF_LOG, which writes provider debug output,
// credentials included, to the log.
func CheckEnv(env map[string]string) error {
	for k := range env {
		if strings.HasPrefix(k, "TF_") && !strings.HasPrefix(k, "TF_VAR_") {
			return fmt.Errorf("%w: variable %q", ErrRefused, k)
		}
	}
	return nil
}

// cliConfigName is the run's Terraform CLI configuration, in the run's folder.
const cliConfigName = "clutter.tfrc"

// cliConfig is a run's Terraform CLI configuration. Terraform installs only
// the allowed providers, and only from the public registry. With none
// allowed, it may install only from an empty folder, so it installs nothing:
// an empty include list would mean every provider.
func cliConfig(dir string, allowed map[string]bool) []byte {
	var b strings.Builder
	b.WriteString("provider_installation {\n")
	if len(allowed) == 0 {
		fmt.Fprintf(&b, "  filesystem_mirror {\n    path = \"%s\"\n  }\n", hclsafe.Escape(filepath.Join(dir, "no-providers")))
	} else {
		names := make([]string, 0, len(allowed))
		for n := range allowed {
			names = append(names, n)
		}
		sort.Strings(names)
		b.WriteString("  direct {\n    include = [\n")
		for _, n := range names {
			// NormalizeProvider allows only [a-z0-9-] and slashes, so no escape is needed.
			fmt.Fprintf(&b, "      \"%s\",\n", n)
		}
		b.WriteString("    ]\n  }\n")
	}
	b.WriteString("}\n")
	return []byte(b.String())
}

// backendFileName is the run folder's state backend file, which the runner
// writes after CheckFiles, so no rendered file can set where state goes.
const backendFileName = "zz_clutter_backend.tf"

// checkBackendFile accepts only what a launcher's StateBackend may return:
// one terraform block holding one backend block and nothing else.
func checkBackendFile(body []byte) error {
	f, diags := hclsyntax.ParseConfig(body, backendFileName, hcl.InitialPos)
	if diags.HasErrors() {
		return errors.New("the launcher's state backend does not parse")
	}
	b, ok := f.Body.(*hclsyntax.Body)
	if !ok || len(b.Attributes) != 0 || len(b.Blocks) != 1 || b.Blocks[0].Type != "terraform" {
		return errors.New("the launcher's state backend is not one terraform block")
	}
	tb := b.Blocks[0].Body
	if len(tb.Attributes) != 0 || len(tb.Blocks) != 1 || tb.Blocks[0].Type != "backend" {
		return errors.New("the launcher's state backend holds more than a backend block")
	}
	return nil
}

// Run checks, locks, prepares and runs one job. It holds the deployment's
// lock until Terraform ends, so two runs of one deployment never share state.
// Every failure before terraform plan, apply or destroy starts (a refused
// job, the lock wait, preparing the folder, terraform init) wraps
// connectors.ErrNothingBuilt: none of them reaches the cloud.
//
// With a runtask.RunLauncher (the cloud), the launcher names the state
// backend and runs every step in one task; B needs no terraform binary.
func (r *Runner) Run(ctx context.Context, j Job) (err error) {
	reachedCloud := false
	defer func() {
		if err != nil && !reachedCloud && !errors.Is(err, connectors.ErrNothingBuilt) {
			err = fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
		}
	}()
	if _, err := uuid.Parse(j.DeploymentID); err != nil {
		return fmt.Errorf("deployment id is not valid")
	}
	if _, err := uuid.Parse(j.RunID); err != nil {
		return fmt.Errorf("run id is not valid")
	}
	if !j.Action.Valid() {
		return fmt.Errorf("action %q is not valid", j.Action)
	}
	if err := CheckFiles(j.Files, j.Providers); err != nil {
		return err
	}
	if _, ok := j.Files[backendFileName]; ok {
		return fmt.Errorf("%w: file name %q is the runner's own", ErrRefused, backendFileName)
	}
	if err := CheckEnv(j.Env); err != nil {
		return err
	}
	if err := CheckArchives(j.Archives); err != nil {
		return err
	}
	if len(j.Archives) > 0 && r.Code == nil {
		return errors.New("the Terraform runner has no code store")
	}
	allowed, err := allowedSet(j.Providers)
	if err != nil {
		return err
	}
	rl, remote := r.Launcher.(runtask.RunLauncher)
	if !remote && !filepath.IsAbs(r.Bin) {
		return errors.New("the terraform binary path must be absolute")
	}

	unlock, err := r.lock(ctx, j.DeploymentID)
	if err != nil {
		return err
	}
	defer unlock()

	var backend []byte
	if remote {
		if backend, err = rl.StateBackend(j.DeploymentID); err != nil {
			return err
		}
		if err := checkBackendFile(backend); err != nil {
			return err
		}
	} else {
		stateDir := filepath.Join(r.StateDir, j.DeploymentID)
		if err := os.MkdirAll(stateDir, 0o700); err != nil {
			return err
		}
		if err := os.MkdirAll(r.PluginCacheDir, 0o700); err != nil {
			return err
		}
		backend = fmt.Appendf(nil, "terraform {\n  backend \"local\" {\n    path = \"%s\"\n  }\n}\n",
			hclsafe.Escape(filepath.Join(stateDir, "terraform.tfstate")))
	}
	dir := filepath.Join(r.WorkDir, j.RunID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	for name, body := range j.Files {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			return err
		}
	}
	for _, a := range j.Archives {
		if err := r.stage(ctx, dir, a); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, backendFileName), backend, 0o600); err != nil {
		return err
	}
	cliPath := filepath.Join(dir, cliConfigName)
	if err := os.WriteFile(cliPath, cliConfig(dir, allowed), 0o600); err != nil {
		return err
	}

	env := r.env(dir, cliPath, j.Env)
	steps := [][]string{{"init", "-input=false", "-no-color"}}
	switch j.Action {
	case connectors.ActionPlan:
		steps = append(steps, []string{"plan", "-input=false", "-no-color", "-lock-timeout=60s"})
	case connectors.ActionApply:
		steps = append(steps, []string{"apply", "-input=false", "-no-color", "-auto-approve", "-lock-timeout=60s"})
	case connectors.ActionDestroy:
		steps = append(steps, []string{"destroy", "-input=false", "-no-color", "-auto-approve", "-lock-timeout=60s"})
	}
	outputs := j.Outputs != nil && j.Action != connectors.ActionDestroy
	if outputs {
		steps = append(steps, []string{runtask.OutputStep, "-json", "-no-color"})
	}
	if remote {
		run := runtask.Run{RunID: j.RunID, DeploymentID: j.DeploymentID, Dir: dir, Steps: steps, Env: env, PublicHosts: j.PublicHosts, Output: j.Log}
		if outputs {
			run.Outputs = j.Outputs
		}
		err := rl.LaunchRun(ctx, run)
		reachedCloud = err != nil && !notStarted(err)
		return err
	}
	for _, args := range steps {
		// init only installs providers and sets up the local state; the
		// steps after it are the first to reach the cloud.
		reachedCloud = args[0] != "init"
		t := runtask.Task{Dir: dir, Path: r.Bin, Args: args, Env: env, Output: j.Log}
		if args[0] == runtask.OutputStep {
			// Sensitive values: stdout to Outputs only, errors to the log.
			t.Output, t.Stderr = j.Outputs, j.Log
		}
		unlock := func() {}
		if args[0] == "init" {
			if unlock, err = r.lockInit(ctx); err != nil {
				return err
			}
		}
		err := r.Launcher.Launch(ctx, t)
		unlock()
		if err != nil {
			return fmt.Errorf("terraform %s: %w", args[0], err)
		}
	}
	return nil
}

// Archive is one uploaded function zip a run stages.
type Archive struct {
	// ID names the staged file (ArchiveName); the connector's rendered
	// Terraform refers to it by that name.
	ID string
	// Key is the zip's key in the code store.
	Key string
	// SHA256 is the zip's digest in lowercase hex, as recorded at upload.
	SHA256 string
}

// archiveIDPattern is an archive id: a Terraform address part, so it is safe
// as a file name.
var archiveIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// ArchiveName is the file an archive is staged as in the run's folder. A
// connector renders this name; the runner, not the rendered text, decides
// which archives exist.
func ArchiveName(id string) string { return "code-" + id + ".zip" }

// maxArchives caps one run's archives.
const maxArchives = 100

// errCodeChanged is the user's message when an archive cannot be staged.
var errCodeChanged = &connectors.UserError{Message: "The code of a function changed or was removed after this run was queued. Start the run again."}

// CheckArchives refuses an archive list with a bad or repeated id, a key the
// code store would not take, or a digest that is not lowercase hex.
func CheckArchives(as []Archive) error {
	if len(as) > maxArchives {
		return fmt.Errorf("%w: more than %d function code archives", ErrRefused, maxArchives)
	}
	seen := map[string]bool{}
	for _, a := range as {
		if !archiveIDPattern.MatchString(a.ID) {
			return fmt.Errorf("%w: archive id %q", ErrRefused, a.ID)
		}
		if seen[a.ID] {
			return fmt.Errorf("%w: archive id %q is given twice", ErrRefused, a.ID)
		}
		seen[a.ID] = true
		if !codestore.ValidKey(a.Key) || !connectors.SHA256Pattern.MatchString(a.SHA256) {
			return fmt.Errorf("%w: archive %q has no valid key or digest", ErrRefused, a.ID)
		}
	}
	return nil
}

// stage copies one archive from the code store into the run's folder. The
// copy must have the digest recorded at upload, and passes the upload's
// path check again, so the run holds exactly the zip the user uploaded.
func (r *Runner) stage(ctx context.Context, dir string, a Archive) error {
	rc, err := r.Code.Open(ctx, a.Key)
	if errors.Is(err, codestore.ErrNotFound) {
		return errCodeChanged
	}
	if err != nil {
		return fmt.Errorf("open function code: %w", err)
	}
	defer rc.Close()
	f, err := os.OpenFile(filepath.Join(dir, ArchiveName(a.ID)), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(rc, codestore.MaxSize+1))
	if err != nil {
		return fmt.Errorf("copy function code: %w", err)
	}
	if n > codestore.MaxSize || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
		return errCodeChanged
	}
	if err := codestore.CheckZip(f, n); err != nil {
		return fmt.Errorf("%w: function code %q: %w", ErrRefused, a.ID, err)
	}
	return f.Close()
}

// notStarted reports whether a launched run failed before Terraform could
// reach the cloud: it never started, or terraform init failed. Any other
// failure, one the run task could not explain included, may have built
// something.
func notStarted(err error) bool {
	if errors.Is(err, runtask.ErrNotStarted) {
		return true
	}
	var se *runtask.StepError
	return errors.As(err, &se) && se.Step == "init"
}

// env builds a run's whole environment: Terraform's own switches, the passed
// variables, and the run's credentials. TF_LOG is never set.
// The runner's own values are set last, so a credential cannot replace them.
func (r *Runner) env(dir, cliConfigPath string, creds map[string]string) map[string]string {
	env := map[string]string{}
	for k, v := range creds {
		env[k] = v
	}
	for _, name := range r.PassEnv {
		if v, ok := os.LookupEnv(name); ok {
			env[name] = v
		}
	}
	env["PATH"] = "/usr/bin:/bin"
	env["HOME"] = dir
	env["TF_IN_AUTOMATION"] = "1"
	env["TF_INPUT"] = "0"
	env["CHECKPOINT_DISABLE"] = "1"
	env["TF_PLUGIN_CACHE_DIR"] = r.PluginCacheDir
	// A run's folder never has a lock file, so without this every init
	// downloads each provider again and rewrites the shared cache.
	env["TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE"] = "1"
	env["TF_CLI_CONFIG_FILE"] = cliConfigPath
	return env
}

// lockInit lets one local run at a time run init, which installs providers
// into the shared plugin cache, waiting until ctx ends.
func (r *Runner) lockInit(ctx context.Context) (func(), error) {
	r.mu.Lock()
	if r.initLock == nil {
		r.initLock = make(chan struct{}, 1)
	}
	ch := r.initLock
	r.mu.Unlock()
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// lock takes the deployment's lock, waiting until ctx ends.
func (r *Runner) lock(ctx context.Context, deploymentID string) (func(), error) {
	r.mu.Lock()
	if r.locks == nil {
		r.locks = map[string]chan struct{}{}
	}
	ch, ok := r.locks[deploymentID]
	if !ok {
		ch = make(chan struct{}, 1)
		r.locks[deploymentID] = ch
	}
	r.mu.Unlock()
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
