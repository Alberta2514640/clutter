package runtasktest

import (
	"context"
	"errors"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
)

// TaskARN is the task the fake ECS starts.
const TaskARN = "arn:aws:ecs:ca-central-1:111122223333:task/clutter-adapter-runs/abc"

// ECS is a fake ECS client that starts one task. Each DescribeTasks call runs
// the next function of Script (the last repeats) to give the task's state.
// Script runs with the fake's lock held, so it may read Stopped directly.
type ECS struct {
	mu sync.Mutex
	// Run is the last RunTask input.
	Run *ecs.RunTaskInput
	// RunErr fails RunTask; Failures are returned in place of a task.
	RunErr   error
	Failures []ecstypes.Failure
	Script   []func() ecstypes.Task
	// Stopped is true once StopTask was called.
	Stopped bool
	calls   int
}

// RunTask implements runtask.ECSAPI.
func (f *ECS) RunTask(_ context.Context, in *ecs.RunTaskInput, _ ...func(*ecs.Options)) (*ecs.RunTaskOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Run = in
	if f.RunErr != nil {
		return nil, f.RunErr
	}
	if f.Failures != nil {
		return &ecs.RunTaskOutput{Failures: f.Failures}, nil
	}
	return &ecs.RunTaskOutput{Tasks: []ecstypes.Task{{TaskArn: aws.String(TaskARN)}}}, nil
}

// DescribeTasks implements runtask.ECSAPI.
func (f *ECS) DescribeTasks(_ context.Context, in *ecs.DescribeTasksInput, _ ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(in.Tasks) != 1 || in.Tasks[0] != TaskARN || len(f.Script) == 0 {
		return nil, errors.New("unknown task")
	}
	i := f.calls
	if i >= len(f.Script) {
		i = len(f.Script) - 1
	}
	f.calls++
	return &ecs.DescribeTasksOutput{Tasks: []ecstypes.Task{f.Script[i]()}}, nil
}

// StopTask implements runtask.ECSAPI.
func (f *ECS) StopTask(_ context.Context, _ *ecs.StopTaskInput, _ ...func(*ecs.Options)) (*ecs.StopTaskOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Stopped = true
	return &ecs.StopTaskOutput{}, nil
}

// Running is a task that has not stopped.
func Running() ecstypes.Task { return ecstypes.Task{LastStatus: aws.String("RUNNING")} }

// StoppedTask is a stopped task whose container ended with code (nil when it
// never ran).
func StoppedTask(container string, code *int32, reason string) ecstypes.Task {
	return ecstypes.Task{
		LastStatus:    aws.String("STOPPED"),
		StoppedReason: aws.String(reason),
		Containers:    []ecstypes.Container{{Name: aws.String(container), ExitCode: code}},
	}
}

// STS is a fake STS client that hands out the keys in its fields.
type STS struct {
	// In is the last AssumeRole input.
	In                                         *sts.AssumeRoleInput
	AccessKeyID, SecretAccessKey, SessionToken string
}

// AssumeRole implements runtask.STSAPI.
func (f *STS) AssumeRole(_ context.Context, in *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	f.In = in
	return &sts.AssumeRoleOutput{Credentials: &ststypes.Credentials{
		AccessKeyId: aws.String(f.AccessKeyID), SecretAccessKey: aws.String(f.SecretAccessKey), SessionToken: aws.String(f.SessionToken),
	}}, nil
}
