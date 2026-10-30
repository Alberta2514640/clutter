package gateway

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Alberta2514640/clutter/backend/devserver/internal/routes"
	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/google/uuid"
)

// Invoker runs one Lambda invocation and returns its raw payload, or the
// function error the Lambda runtime would report.
type Invoker interface {
	Invoke(ctx context.Context, fn string, payload []byte) ([]byte, *messages.InvokeResponse_Error, error)
}

// ErrTimeout means the Lambda ran past its configured timeout.
var ErrTimeout = errors.New("lambda timed out")

// BuildLambdas compiles every Lambda in routes.Lambdas from apiDir into binDir,
// one binary per function name, for the host OS (not the Lambda's linux/arm64).
func BuildLambdas(ctx context.Context, apiDir, binDir string, logger *log.Logger) error {
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	for _, l := range routes.Lambdas {
		wg.Add(1)
		go func(l routes.Lambda) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out := filepath.Join(binDir, l.Name)
			cmd := exec.CommandContext(ctx, "go", "-C", apiDir, "build", "-o", out, "./"+l.Dir)
			cmd.Env = os.Environ()
			if b, err := cmd.CombinedOutput(); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("build %s (%s): %v\n%s", l.Name, l.Dir, err, b))
				mu.Unlock()
				return
			}
		}(l)
	}
	wg.Wait()
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	logger.Printf("built %d Lambdas into %s", len(routes.Lambdas), binDir)
	return nil
}

// proc is one running Lambda binary in RPC mode.
type proc struct {
	lambda routes.Lambda
	bin    string

	// invokeMu admits one invocation at a time, as a Lambda instance does.
	// aws-lambda-go's RPC mode is not built for overlapping calls: under
	// parallel load they fail, and the browser sends requests in parallel.
	invokeMu sync.Mutex

	mu     sync.Mutex
	cmd    *exec.Cmd
	client *rpc.Client
	exited chan struct{}
}

// Pool runs one process per Lambda, like one warm Lambda instance each.
type Pool struct {
	cfg    Config
	binDir string
	logger *log.Logger

	procs map[string]*proc
}

// NewPool prepares a pool for the Lambdas built into binDir.
func NewPool(cfg Config, binDir string, logger *log.Logger) *Pool {
	p := &Pool{cfg: cfg, binDir: binDir, logger: logger, procs: map[string]*proc{}}
	for _, l := range routes.Lambdas {
		p.procs[l.Name] = &proc{lambda: l, bin: filepath.Join(binDir, l.Name)}
	}
	return p
}

// Start launches every Lambda and waits until each answers an RPC ping.
func (p *Pool) Start(ctx context.Context) error {
	for _, pr := range p.procs {
		if err := p.start(ctx, pr); err != nil {
			return err
		}
	}
	p.logger.Printf("started %d Lambda processes", len(p.procs))
	return nil
}

// Pids returns the process ids of the running Lambdas.
func (p *Pool) Pids() []int {
	var pids []int
	for _, pr := range p.procs {
		pr.mu.Lock()
		if pr.cmd != nil && pr.cmd.Process != nil {
			pids = append(pids, pr.cmd.Process.Pid)
		}
		pr.mu.Unlock()
	}
	return pids
}

// Stop kills every Lambda process.
func (p *Pool) Stop() {
	for _, pr := range p.procs {
		pr.mu.Lock()
		p.killLocked(pr)
		pr.mu.Unlock()
	}
}

func (p *Pool) killLocked(pr *proc) {
	if pr.client != nil {
		_ = pr.client.Close()
		pr.client = nil
	}
	if pr.cmd != nil && pr.cmd.Process != nil {
		_ = pr.cmd.Process.Kill()
		<-pr.exited
	}
	pr.cmd = nil
}

func freePort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func (p *Pool) start(ctx context.Context, pr *proc) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return p.startLocked(ctx, pr)
}

func (p *Pool) startLocked(ctx context.Context, pr *proc) error {
	port, err := freePort()
	if err != nil {
		return err
	}
	env, err := processEnv(pr.lambda.Name, pr.lambda.EnvKeys, port, p.cfg)
	if err != nil {
		return err
	}
	cmd := exec.Command(pr.bin)
	cmd.Env = env
	cmd.Dir = p.cfg.HomeDir
	pipeR, pipeW := io.Pipe()
	cmd.Stdout = pipeW
	cmd.Stderr = pipeW
	if err := cmd.Start(); err != nil {
		_ = pipeW.Close()
		return fmt.Errorf("start %s: %w", pr.lambda.Name, err)
	}
	exited := make(chan struct{})
	go p.relay(pr.lambda.Name, pipeR)
	go func() {
		_ = cmd.Wait()
		_ = pipeW.Close()
		close(exited)
	}()
	pr.cmd = cmd
	pr.exited = exited

	// aws-lambda-go listens on "localhost:<port>" in RPC mode.
	addr := net.JoinHostPort("localhost", port)
	deadline := time.Now().Add(15 * time.Second)
	for {
		select {
		case <-exited:
			return fmt.Errorf("%s exited during start-up", pr.lambda.Name)
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		client, err := rpc.Dial("tcp", addr)
		if err == nil {
			var pong messages.PingResponse
			if err = client.Call("Function.Ping", &messages.PingRequest{}, &pong); err == nil {
				pr.client = client
				return nil
			}
			_ = client.Close()
		}
		if time.Now().After(deadline) {
			p.killLocked(pr)
			return fmt.Errorf("%s did not answer on %s: %v", pr.lambda.Name, addr, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// relay copies a Lambda's output into the gateway log, one prefixed line each,
// like CloudWatch Logs would hold it.
func (p *Pool) relay(name string, r io.Reader) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		p.logger.Printf("[%s] %s", name, s.Text())
	}
}

// Invoke implements Invoker. It applies the Lambda's own timeout; a timed-out
// or crashed process is replaced, as Lambda would replace the instance.
func (p *Pool) Invoke(ctx context.Context, fn string, payload []byte) ([]byte, *messages.InvokeResponse_Error, error) {
	pr, ok := p.procs[fn]
	if !ok {
		return nil, nil, fmt.Errorf("no Lambda named %q", fn)
	}
	pr.invokeMu.Lock()
	defer pr.invokeMu.Unlock()
	pr.mu.Lock()
	if pr.client == nil {
		if err := p.startLocked(ctx, pr); err != nil {
			pr.mu.Unlock()
			return nil, nil, err
		}
	}
	client := pr.client
	pr.mu.Unlock()

	timeout := time.Duration(pr.lambda.TimeoutSeconds) * time.Second
	deadline := time.Now().Add(timeout)
	req := &messages.InvokeRequest{
		Payload:            payload,
		RequestId:          uuid.NewString(),
		Deadline:           messages.InvokeRequest_Timestamp{Seconds: deadline.Unix(), Nanos: int64(deadline.Nanosecond())},
		InvokedFunctionArn: "arn:aws:lambda:" + localRegion + ":" + localAccountID + ":function:" + fn,
	}
	var resp messages.InvokeResponse
	call := client.Go("Function.Invoke", req, &resp, make(chan *rpc.Call, 1))

	// One second of grace past the deadline, then the instance is replaced.
	timer := time.NewTimer(timeout + time.Second)
	defer timer.Stop()
	select {
	case <-call.Done:
		if call.Error != nil {
			p.replace(pr, client)
			return nil, nil, fmt.Errorf("invoke %s: %w", fn, call.Error)
		}
		return resp.Payload, resp.Error, nil
	case <-timer.C:
		p.replace(pr, client)
		return nil, nil, fmt.Errorf("%w after %s: %s", ErrTimeout, timeout, fn)
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

// replace kills the process behind a failed client so the next call starts a
// fresh one. A client that was already replaced is left alone.
func (p *Pool) replace(pr *proc, failed *rpc.Client) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	if pr.client != failed {
		return
	}
	p.logger.Printf("replacing Lambda process %s", pr.lambda.Name)
	p.killLocked(pr)
}
