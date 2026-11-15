package runtask

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// The object layout one cloud run uses in the run bucket, shared by the ECS
// launcher (in B) and the run task program (backend/adapter/cmd/runtask), so the
// two cannot drift. Everything under a run's prefix is deleted when the run
// ends; the bucket's lifecycle rule removes what a crash leaves behind.
//
//	runs/<run>/in/files/<name>   the run folder, one object per file
//	runs/<run>/in/run.json       RunSpec: the steps and the environment
//	runs/<run>/in/task.env       the run's short-lived S3 keys, read by ECS
//	runs/<run>/out/log.txt       Terraform's output so far
//	runs/<run>/out/result.json   Result
//	runs/<run>/out/outputs.json  the output step's stdout: Terraform's outputs,
//	                             sensitive values included; never copied into the log
//
// State lives in the state bucket at state/<deployment>/terraform.tfstate,
// locked by the S3 lock file beside it.
type RunKeys struct {
	Prefix      string
	InPrefix    string
	FilesPrefix string
	Spec        string
	TaskEnv     string
	OutPrefix   string
	Log         string
	Result      string
	Outputs     string
}

// KeysFor returns the object keys of one run.
func KeysFor(runID string) RunKeys {
	p := "runs/" + runID + "/"
	return RunKeys{
		Prefix:      p,
		InPrefix:    p + "in/",
		FilesPrefix: p + "in/files/",
		Spec:        p + "in/run.json",
		TaskEnv:     p + "in/task.env",
		OutPrefix:   p + "out/",
		Log:         p + "out/log.txt",
		Result:      p + "out/result.json",
		Outputs:     p + "out/outputs.json",
	}
}

// StatePrefix is the folder of one deployment's state in the state bucket.
func StatePrefix(deploymentID string) string { return "state/" + deploymentID + "/" }

// StateKey is one deployment's state object; its lock file is StateKey + ".tflock".
func StateKey(deploymentID string) string { return StatePrefix(deploymentID) + "terraform.tfstate" }

// RunSpec is what the run task needs besides the files.
type RunSpec struct {
	Steps [][]string        `json:"steps"`
	Env   map[string]string `json:"env"`
	// Dir is the run folder's path on B's machine. The run task rewrites
	// environment values inside it (HOME, TF_CLI_CONFIG_FILE) to its own copy.
	Dir string `json:"dir"`
	// PublicHosts are Run.PublicHosts: the run task refuses the run when one
	// resolves to a refused address.
	PublicHosts []string `json:"public_hosts,omitempty"`
}

// ErrNotStarted is wrapped by every LaunchRun error that comes before
// Terraform could reach a cloud: a refused run, the upload, the task's start,
// a task that stopped before Terraform ran, or a run the task could not
// prepare.
var ErrNotStarted = errors.New("the run did not start")

// StepError is a Terraform command that ended with a non-zero exit code.
type StepError struct {
	Step     string
	ExitCode int
}

func (e *StepError) Error() string {
	return fmt.Sprintf("terraform %s: exit status %d", e.Step, e.ExitCode)
}

// PrepareStep is Result.Step when the run task could not prepare the run.
const PrepareStep = "prepare"

// Result is what the run task reports when it ends. Step names the command
// that failed, or is empty when every command succeeded.
type Result struct {
	Step     string `json:"step"`
	ExitCode int    `json:"exit_code"`
}

// The run task's own environment. The S3 keys come from task.env, which ECS
// reads with the task's execution role, so they never appear in the task's
// description. The run task never passes any of these to Terraform.
const (
	EnvRunID           = "CLUTTER_RUN_ID"
	EnvRunBucket       = "CLUTTER_RUN_BUCKET"
	EnvRegion          = "CLUTTER_RUN_REGION"
	EnvAccessKeyID     = "CLUTTER_RUN_ACCESS_KEY_ID"
	EnvSecretAccessKey = "CLUTTER_RUN_SECRET_ACCESS_KEY"
	EnvSessionToken    = "CLUTTER_RUN_SESSION_TOKEN"
)

var (
	lockTimeoutPattern = regexp.MustCompile(`^-lock-timeout=[0-9]{1,4}[sm]$`)
	// FileNamePattern is a file of a run folder: a plain name, no folder.
	FileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
)

// OutputStep is the step that prints Terraform's outputs as JSON. Its stdout
// holds sensitive values, so it goes to Run.Outputs, never to the log.
const OutputStep = "output"

// MaxOutputs caps the output step's stdout in bytes.
const MaxOutputs = 1 << 20

// CheckSteps refuses a step that is not init, plan, apply, destroy or output
// with the flags the Terraform runner uses. A flag that names a file or a
// folder (-chdir, -var-file, -state, -backend-config, -plugin-dir) is
// refused, so a run spec cannot point Terraform anywhere but its own folder.
func CheckSteps(steps [][]string) error {
	if len(steps) == 0 {
		return errors.New("a run needs at least one step")
	}
	for _, s := range steps {
		if len(s) == 0 {
			return errors.New("a step is empty")
		}
		switch s[0] {
		case "init", "plan", "apply", "destroy", OutputStep:
		default:
			return fmt.Errorf("step %q is not allowed", s[0])
		}
		for _, a := range s[1:] {
			switch {
			case a == "-input=false", a == "-no-color", a == "-auto-approve":
			case a == "-json" && s[0] == OutputStep:
			case lockTimeoutPattern.MatchString(a):
			default:
				return fmt.Errorf("flag %q is not allowed", a)
			}
		}
	}
	return nil
}

// S3API is the part of the S3 client the launcher and the run task use.
type S3API interface {
	PutObject(ctx context.Context, in *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, in *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(ctx context.Context, in *s3.ListObjectsV2Input, opts ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObject(ctx context.Context, in *s3.DeleteObjectInput, opts ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// ListKeys lists every key under prefix.
func ListKeys(ctx context.Context, c S3API, bucket, prefix string) ([]string, error) {
	var keys []string
	var token *string
	for {
		out, err := c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &bucket, Prefix: &prefix, ContinuationToken: token})
		if err != nil {
			return nil, err
		}
		for _, o := range out.Contents {
			if o.Key != nil {
				keys = append(keys, *o.Key)
			}
		}
		if out.IsTruncated == nil || !*out.IsTruncated || out.NextContinuationToken == nil {
			return keys, nil
		}
		token = out.NextContinuationToken
	}
}

// credentialPattern is the alphabet of AWS access keys, secret keys and
// session tokens. A value outside it is refused before it is written into a
// file the run task or ECS reads.
var credentialPattern = regexp.MustCompile(`^[A-Za-z0-9/+=]{16,}$`)

// ValidCredential reports whether v looks like an AWS key or session token.
func ValidCredential(v string) bool { return len(v) <= 4096 && credentialPattern.MatchString(v) }

// clampDuration keeps a session between STS's lower bound and the one-hour
// limit on a role assumed from another role's session.
func clampDuration(d time.Duration) time.Duration {
	switch {
	case d < 15*time.Minute:
		return 15 * time.Minute
	case d > time.Hour:
		return time.Hour
	}
	return d.Round(time.Second)
}
