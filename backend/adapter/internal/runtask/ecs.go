package runtask

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/internal/s3name"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

// ECSSettings is where cloud runs go. LoadECSSettings reads it from B's
// environment.
type ECSSettings struct {
	Region string
	// Cluster is B's own run cluster (name or ARN), never A's.
	Cluster string
	// TaskDefinition is the run task's family, family:revision or ARN.
	TaskDefinition string
	// Container is the container's name in the task definition.
	Container      string
	Subnets        []string
	SecurityGroups []string
	// AssignPublicIP gives each task a public address, for subnets without NAT.
	AssignPublicIP bool
	// RunBucket holds each run's files, log and result; StateBucket holds
	// each deployment's state and its lock file.
	RunBucket   string
	StateBucket string
	// RunFilesRoleARN is the role B assumes for each run, narrowed by a
	// session policy to that run's objects and that deployment's state. The
	// run task has no role of its own.
	RunFilesRoleARN string
	// PollInterval is how often B reads the task's status and log. Zero means
	// 3 seconds.
	PollInterval time.Duration
	// StopGrace is how long a cancelled task has to stop. Zero means 150
	// seconds, longer than the task definition's 120-second stop timeout.
	StopGrace time.Duration
}

var (
	regionPattern  = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-[0-9]$`)
	subnetPattern  = regexp.MustCompile(`^subnet-[0-9a-f]{8,17}$`)
	sgPattern      = regexp.MustCompile(`^sg-[0-9a-f]{8,17}$`)
	roleARNPattern = regexp.MustCompile(`^arn:aws[a-z-]*:iam::[0-9]{12}:role/[A-Za-z0-9+=,.@_/-]{1,512}$`)
	namePattern    = regexp.MustCompile(`^[A-Za-z0-9:/_.-]{1,255}$`)
)

// LoadECSSettings reads the run launcher's settings: AWS_REGION, ADAPTER_RUN_CLUSTER,
// ADAPTER_RUN_TASK_DEFINITION, ADAPTER_RUN_CONTAINER, ADAPTER_RUN_SUBNETS and
// ADAPTER_RUN_SECURITY_GROUPS (comma-separated), ADAPTER_RUN_BUCKET, ADAPTER_STATE_BUCKET and
// ADAPTER_RUN_FILES_ROLE_ARN, all required, and ADAPTER_RUN_ASSIGN_PUBLIC_IP ("true" or
// "false", default false). Errors name the variable, never its value.
func LoadECSSettings(getenv func(string) string) (ECSSettings, error) {
	var errs []error
	get := func(name string, re *regexp.Regexp) string {
		v := strings.TrimSpace(getenv(name))
		switch {
		case v == "":
			errs = append(errs, fmt.Errorf("%s is not set", name))
		case !re.MatchString(v):
			errs = append(errs, fmt.Errorf("%s is not valid", name))
		}
		return v
	}
	list := func(name string, re *regexp.Regexp) []string {
		var out []string
		for _, v := range strings.Split(getenv(name), ",") {
			if v = strings.TrimSpace(v); v == "" {
				continue
			}
			if !re.MatchString(v) {
				errs = append(errs, fmt.Errorf("%s is not valid", name))
				return nil
			}
			out = append(out, v)
		}
		if len(out) == 0 {
			errs = append(errs, fmt.Errorf("%s is not set", name))
		}
		return out
	}
	s := ECSSettings{
		Region:          get("AWS_REGION", regionPattern),
		Cluster:         get("ADAPTER_RUN_CLUSTER", namePattern),
		TaskDefinition:  get("ADAPTER_RUN_TASK_DEFINITION", namePattern),
		Container:       get("ADAPTER_RUN_CONTAINER", namePattern),
		Subnets:         list("ADAPTER_RUN_SUBNETS", subnetPattern),
		SecurityGroups:  list("ADAPTER_RUN_SECURITY_GROUPS", sgPattern),
		RunBucket:       get("ADAPTER_RUN_BUCKET", s3name.BucketPattern),
		StateBucket:     get("ADAPTER_STATE_BUCKET", s3name.BucketPattern),
		RunFilesRoleARN: get("ADAPTER_RUN_FILES_ROLE_ARN", roleARNPattern),
	}
	switch strings.TrimSpace(getenv("ADAPTER_RUN_ASSIGN_PUBLIC_IP")) {
	case "", "false":
	case "true":
		s.AssignPublicIP = true
	default:
		errs = append(errs, errors.New(`ADAPTER_RUN_ASSIGN_PUBLIC_IP must be "true" or "false"`))
	}
	if err := errors.Join(errs...); err != nil {
		return ECSSettings{}, err
	}
	return s, nil
}

// ECSAPI is the part of the ECS client the launcher uses.
type ECSAPI interface {
	RunTask(ctx context.Context, in *ecs.RunTaskInput, opts ...func(*ecs.Options)) (*ecs.RunTaskOutput, error)
	DescribeTasks(ctx context.Context, in *ecs.DescribeTasksInput, opts ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
	StopTask(ctx context.Context, in *ecs.StopTaskInput, opts ...func(*ecs.Options)) (*ecs.StopTaskOutput, error)
}

// STSAPI is the part of the STS client the launcher uses.
type STSAPI interface {
	AssumeRole(ctx context.Context, in *sts.AssumeRoleInput, opts ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

// ECS runs each run as its own Fargate task in B's run cluster (ADR-017).
//
// For each run B assumes the run-files role with a session policy that
// allows only that run's objects and that deployment's state, uploads the
// run folder and the run spec, writes those keys into task.env, and starts
// the task with nothing in its overrides but the run id, the bucket, the
// region and task.env's address. The task has no role, no database and no
// secret store. B follows the task until it stops, copying its log into the
// run's output, and deletes the run's objects at the end.
type ECS struct {
	Settings ECSSettings
	ECS      ECSAPI
	S3       S3API
	STS      STSAPI
}

// maxDescribeFailures is how many status reads in a row may fail before B
// gives up on following a task. The task is stopped when that happens.
const maxDescribeFailures = 10

// Launch implements Launcher. The cloud runs a whole run in one task, so a
// single command is refused; the Terraform runner calls LaunchRun.
func (e *ECS) Launch(context.Context, Task) error {
	return errors.New("the ECS launcher runs whole runs only (LaunchRun)")
}

// StateBackend implements RunLauncher: S3 state per deployment, encrypted,
// locked by S3's lock file. Workspaces are kept under the deployment's folder
// too, so Terraform never lists outside it.
func (e *ECS) StateBackend(deploymentID string) ([]byte, error) {
	if _, err := uuid.Parse(deploymentID); err != nil {
		return nil, errors.New("deployment id is not valid")
	}
	if !s3name.BucketPattern.MatchString(e.Settings.StateBucket) || !regionPattern.MatchString(e.Settings.Region) {
		return nil, errors.New("the state bucket or region is not valid")
	}
	return []byte(fmt.Sprintf(`terraform {
  backend "s3" {
    bucket               = %q
    key                  = %q
    workspace_key_prefix = %q
    region               = %q
    encrypt              = true
    use_lockfile         = true
  }
}
`, e.Settings.StateBucket, StateKey(deploymentID), StatePrefix(deploymentID)+"workspaces", e.Settings.Region)), nil
}

// LaunchRun implements RunLauncher.
func (e *ECS) LaunchRun(ctx context.Context, r Run) error {
	if _, err := uuid.Parse(r.RunID); err != nil {
		return fmt.Errorf("%w: run id is not valid", ErrNotStarted)
	}
	if _, err := uuid.Parse(r.DeploymentID); err != nil {
		return fmt.Errorf("%w: deployment id is not valid", ErrNotStarted)
	}
	if err := CheckSteps(r.Steps); err != nil {
		return fmt.Errorf("%w: %w", ErrNotStarted, err)
	}
	if hasOutputStep(r.Steps) && r.Outputs == nil {
		return fmt.Errorf("%w: the run prints its outputs but nothing takes them", ErrNotStarted)
	}
	if _, err := EnvList(r.Env); err != nil {
		return fmt.Errorf("%w: %w", ErrNotStarted, err)
	}
	out := r.Output
	if out == nil {
		out = io.Discard
	}
	keys := KeysFor(r.RunID)
	defer e.cleanup(keys)

	creds, err := e.runCredentials(ctx, r)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotStarted, err)
	}
	if err := e.upload(ctx, r, keys, creds); err != nil {
		return fmt.Errorf("%w: %w", ErrNotStarted, err)
	}
	taskARN, err := e.start(ctx, r, keys)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotStarted, err)
	}
	if err := e.follow(ctx, taskARN, keys, out); err != nil {
		return err
	}
	return e.outputs(ctx, r, keys)
}

// hasOutputStep reports whether a run prints Terraform's outputs.
func hasOutputStep(steps [][]string) bool {
	for _, s := range steps {
		if len(s) > 0 && s[0] == OutputStep {
			return true
		}
	}
	return false
}

// outputs copies a finished run's output step result to r.Outputs. It is
// never copied into the log.
func (e *ECS) outputs(ctx context.Context, r Run, keys RunKeys) error {
	if !hasOutputStep(r.Steps) {
		return nil
	}
	if r.Outputs == nil {
		return errors.New("the run prints its outputs but nothing takes them")
	}
	body, err := e.get(ctx, keys.Outputs, "")
	if err != nil {
		return fmt.Errorf("read the run's outputs: %w", err)
	}
	if len(body) > MaxOutputs {
		return errors.New("the run's outputs are too large")
	}
	_, err = r.Outputs.Write(body)
	return err
}

type sessionKeys struct{ id, secret, token string }

// runCredentials assumes the run-files role for this run only. The session
// lasts until the run's deadline, between 15 minutes and one hour (the limit
// for a role assumed from B's own role session).
func (e *ECS) runCredentials(ctx context.Context, r Run) (sessionKeys, error) {
	policy, err := runPolicy(e.Settings.RunBucket, e.Settings.StateBucket, r.RunID, r.DeploymentID)
	if err != nil {
		return sessionKeys{}, err
	}
	d := time.Hour
	if dl, ok := ctx.Deadline(); ok {
		d = time.Until(dl) + 5*time.Minute
	}
	d = clampDuration(d)
	out, err := e.STS.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(e.Settings.RunFilesRoleARN),
		RoleSessionName: aws.String("b-run-" + r.RunID),
		DurationSeconds: aws.Int32(int32(d / time.Second)),
		Policy:          aws.String(policy),
	})
	if err != nil {
		return sessionKeys{}, fmt.Errorf("run keys: %w", err)
	}
	c := out.Credentials
	if c == nil || c.AccessKeyId == nil || c.SecretAccessKey == nil || c.SessionToken == nil {
		return sessionKeys{}, errors.New("run keys: STS returned no credentials")
	}
	k := sessionKeys{*c.AccessKeyId, *c.SecretAccessKey, *c.SessionToken}
	if !ValidCredential(k.id) || !ValidCredential(k.secret) || !ValidCredential(k.token) {
		return sessionKeys{}, errors.New("run keys: STS returned keys of an unexpected form")
	}
	return k, nil
}

type policyStatement struct {
	Effect    string         `json:"Effect"`
	Action    []string       `json:"Action"`
	Resource  string         `json:"Resource"`
	Condition map[string]any `json:"Condition,omitempty"`
}

// runPolicy is the session policy of one run: read and delete its inputs,
// write its outputs, and read, write and lock one deployment's state.
func runPolicy(runBucket, stateBucket, runID, deploymentID string) (string, error) {
	k := KeysFor(runID)
	sp := StatePrefix(deploymentID)
	st := StateKey(deploymentID)
	doc := struct {
		Version   string            `json:"Version"`
		Statement []policyStatement `json:"Statement"`
	}{"2012-10-17", []policyStatement{
		{"Allow", []string{"s3:GetObject", "s3:DeleteObject"}, "arn:aws:s3:::" + runBucket + "/" + k.InPrefix + "*", nil},
		{"Allow", []string{"s3:PutObject"}, "arn:aws:s3:::" + runBucket + "/" + k.OutPrefix + "*", nil},
		{"Allow", []string{"s3:ListBucket"}, "arn:aws:s3:::" + runBucket,
			map[string]any{"StringLike": map[string]string{"s3:prefix": k.InPrefix + "*"}}},
		{"Allow", []string{"s3:ListBucket"}, "arn:aws:s3:::" + stateBucket,
			map[string]any{"StringLike": map[string]string{"s3:prefix": sp + "*"}}},
		{"Allow", []string{"s3:GetObject", "s3:PutObject"}, "arn:aws:s3:::" + stateBucket + "/" + st, nil},
		{"Allow", []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"}, "arn:aws:s3:::" + stateBucket + "/" + st + ".tflock", nil},
	}}
	b, err := json.Marshal(doc)
	return string(b), err
}

// upload sends the run folder, the run spec and task.env.
func (e *ECS) upload(ctx context.Context, r Run, keys RunKeys, creds sessionKeys) error {
	entries, err := os.ReadDir(r.Dir)
	if err != nil {
		return fmt.Errorf("read run folder: %w", err)
	}
	for _, ent := range entries {
		if !ent.Type().IsRegular() {
			return fmt.Errorf("the run folder holds %q, which is not a plain file", ent.Name())
		}
		if !FileNamePattern.MatchString(ent.Name()) {
			return fmt.Errorf("file name %q is not allowed", ent.Name())
		}
		body, err := os.ReadFile(filepath.Join(r.Dir, ent.Name()))
		if err != nil {
			return err
		}
		if err := e.put(ctx, keys.FilesPrefix+ent.Name(), body); err != nil {
			return fmt.Errorf("upload run files: %w", err)
		}
	}
	spec, err := json.Marshal(RunSpec{Steps: r.Steps, Env: r.Env, Dir: r.Dir, PublicHosts: r.PublicHosts})
	if err != nil {
		return err
	}
	if err := e.put(ctx, keys.Spec, spec); err != nil {
		return fmt.Errorf("upload run spec: %w", err)
	}
	env := EnvAccessKeyID + "=" + creds.id + "\n" +
		EnvSecretAccessKey + "=" + creds.secret + "\n" +
		EnvSessionToken + "=" + creds.token + "\n"
	if err := e.put(ctx, keys.TaskEnv, []byte(env)); err != nil {
		return fmt.Errorf("upload task keys: %w", err)
	}
	return nil
}

func (e *ECS) put(ctx context.Context, key string, body []byte) error {
	_, err := e.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(e.Settings.RunBucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	return err
}

// start starts the task. The run id is the idempotency token, so a retried
// call never starts a second task for one run.
func (e *ECS) start(ctx context.Context, r Run, keys RunKeys) (string, error) {
	assign := ecstypes.AssignPublicIpDisabled
	if e.Settings.AssignPublicIP {
		assign = ecstypes.AssignPublicIpEnabled
	}
	out, err := e.ECS.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:        aws.String(e.Settings.Cluster),
		TaskDefinition: aws.String(e.Settings.TaskDefinition),
		LaunchType:     ecstypes.LaunchTypeFargate,
		Count:          aws.Int32(1),
		ClientToken:    aws.String(r.RunID),
		StartedBy:      aws.String("b-run/" + r.RunID),
		NetworkConfiguration: &ecstypes.NetworkConfiguration{AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
			Subnets:        e.Settings.Subnets,
			SecurityGroups: e.Settings.SecurityGroups,
			AssignPublicIp: assign,
		}},
		Overrides: &ecstypes.TaskOverride{ContainerOverrides: []ecstypes.ContainerOverride{{
			Name: aws.String(e.Settings.Container),
			Environment: []ecstypes.KeyValuePair{
				{Name: aws.String(EnvRunID), Value: aws.String(r.RunID)},
				{Name: aws.String(EnvRunBucket), Value: aws.String(e.Settings.RunBucket)},
				{Name: aws.String(EnvRegion), Value: aws.String(e.Settings.Region)},
			},
			EnvironmentFiles: []ecstypes.EnvironmentFile{{
				Type:  ecstypes.EnvironmentFileTypeS3,
				Value: aws.String("arn:aws:s3:::" + e.Settings.RunBucket + "/" + keys.TaskEnv),
			}},
		}}},
	})
	if err != nil {
		return "", fmt.Errorf("start run task: %w", err)
	}
	if len(out.Failures) > 0 {
		f := out.Failures[0]
		return "", fmt.Errorf("start run task: %s %s", aws.ToString(f.Reason), aws.ToString(f.Detail))
	}
	if len(out.Tasks) != 1 || out.Tasks[0].TaskArn == nil {
		return "", errors.New("start run task: ECS started no task")
	}
	return *out.Tasks[0].TaskArn, nil
}

// follow waits for the task to stop, copying its log as it grows. When ctx
// ends it stops the task and waits for it, so Terraform can release the
// state lock.
func (e *ECS) follow(ctx context.Context, taskARN string, keys RunKeys, out io.Writer) error {
	poll := e.Settings.PollInterval
	if poll <= 0 {
		poll = 3 * time.Second
	}
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	var offset int64
	failures := 0
	for {
		select {
		case <-ctx.Done():
			return e.stop(taskARN, keys, out, &offset, ctx.Err())
		case <-ticker.C:
		}
		offset = e.tail(ctx, keys.Log, offset, out)
		t, err := e.describe(ctx, taskARN)
		if err != nil {
			if ctx.Err() != nil {
				continue
			}
			if failures++; failures >= maxDescribeFailures {
				return e.stop(taskARN, keys, out, &offset, fmt.Errorf("follow run task: %w", err))
			}
			continue
		}
		failures = 0
		if aws.ToString(t.LastStatus) == "STOPPED" {
			offset = e.tail(ctx, keys.Log, offset, out)
			return e.outcome(ctx, t, keys)
		}
	}
}

func (e *ECS) describe(ctx context.Context, taskARN string) (ecstypes.Task, error) {
	out, err := e.ECS.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(e.Settings.Cluster),
		Tasks:   []string{taskARN},
	})
	if err != nil {
		return ecstypes.Task{}, err
	}
	if len(out.Tasks) != 1 {
		return ecstypes.Task{}, errors.New("ECS does not describe the run task")
	}
	return out.Tasks[0], nil
}

// stop stops the task, waits up to StopGrace for it, copies the rest of its
// log, and returns cause.
func (e *ECS) stop(taskARN string, keys RunKeys, out io.Writer, offset *int64, cause error) error {
	grace := e.Settings.StopGrace
	if grace <= 0 {
		grace = 150 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if _, err := e.ECS.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: aws.String(e.Settings.Cluster),
		Task:    aws.String(taskARN),
		Reason:  aws.String("Clutter stopped the run"),
	}); err != nil {
		fmt.Fprintf(out, "\nclutter: the run task could not be stopped: %v\n", err)
		return cause
	}
	poll := e.Settings.PollInterval
	if poll <= 0 {
		poll = 3 * time.Second
	}
	for {
		*offset = e.tail(ctx, keys.Log, *offset, out)
		if t, err := e.describe(ctx, taskARN); err == nil && aws.ToString(t.LastStatus) == "STOPPED" {
			*offset = e.tail(ctx, keys.Log, *offset, out)
			return cause
		}
		select {
		case <-ctx.Done():
			fmt.Fprintln(out, "\nclutter: the run task did not stop in time")
			return cause
		case <-time.After(poll):
		}
	}
}

// outcome turns a stopped task into the run's result: nil, an error wrapping
// ErrNotStarted, or a *StepError naming the command that failed. A failure the
// result file does not explain is a StepError for "run", which the Terraform
// runner treats as possibly having reached the cloud.
func (e *ECS) outcome(ctx context.Context, t ecstypes.Task, keys RunKeys) error {
	var code *int32
	for _, c := range t.Containers {
		if aws.ToString(c.Name) == e.Settings.Container {
			code = c.ExitCode
		}
	}
	if code == nil {
		return fmt.Errorf("%w: the run task stopped before Terraform ran: %s", ErrNotStarted, aws.ToString(t.StoppedReason))
	}
	if *code == 0 {
		return nil
	}
	step := "run"
	if body, err := e.get(ctx, keys.Result, ""); err == nil {
		var res Result
		if json.Unmarshal(body, &res) == nil && CheckSteps([][]string{{res.Step}}) == nil {
			step = res.Step
		} else if res.Step == PrepareStep {
			return fmt.Errorf("%w: the run task could not prepare the run (exit status %d)", ErrNotStarted, *code)
		}
	}
	return &StepError{Step: step, ExitCode: int(*code)}
}

// tail copies the log from offset onward and returns the new offset. A log
// that does not exist yet, or has not grown, copies nothing; any other error
// is retried at the next poll.
func (e *ECS) tail(ctx context.Context, key string, offset int64, out io.Writer) int64 {
	body, err := e.get(ctx, key, fmt.Sprintf("bytes=%d-", offset))
	if err != nil {
		return offset
	}
	n, _ := out.Write(body)
	return offset + int64(n)
}

func (e *ECS) get(ctx context.Context, key, rng string) ([]byte, error) {
	in := &s3.GetObjectInput{Bucket: aws.String(e.Settings.RunBucket), Key: aws.String(key)}
	if rng != "" {
		in.Range = aws.String(rng)
	}
	res, err := e.S3.GetObject(ctx, in)
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) && (ae.ErrorCode() == "NoSuchKey" || ae.ErrorCode() == "InvalidRange") {
			return nil, errNoObject
		}
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

var errNoObject = errors.New("no such object")

// cleanup deletes every object of the run, with its own deadline because the
// run's context may have ended. The bucket's lifecycle rule removes what a
// failed cleanup leaves.
func (e *ECS) cleanup(keys RunKeys) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	objs, err := ListKeys(ctx, e.S3, e.Settings.RunBucket, keys.Prefix)
	if err != nil {
		return
	}
	for _, k := range objs {
		_, _ = e.S3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(e.Settings.RunBucket), Key: aws.String(k)})
	}
}
