// Package generalopenstack is General's OpenStack connector, id
// "general-openstack". It renders building blocks as Terraform for
// terraform-provider-openstack ~> 3.4, hands the files to B's shared Terraform
// runner (internal/terraform), and verifies an application credential with one
// Keystone token request. It lists the cloud's flavors, images and external
// networks for the deployment settings (connectors.OptionLister).
//
// Credentials reach Terraform only as OS_* variables in the run's
// environment, never in a file. Every value that reaches HCL is allow-listed,
// pattern-checked and escaped with hclsafe; Terraform addresses are keyed by
// node id and cloud-side names start with the deployment's prefix.
package generalopenstack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
)

// ID is the connector's id.
const ID = "general-openstack"

// The provider this connector renders for. 1.54 cannot find block storage on
// current DevStack; 3.4 works (docs/research/openstack).
const (
	providerSource  = "terraform-provider-openstack/openstack"
	providerVersion = "~> 3.4"
	// timeSource only waits (time_sleep): before a public VM's floating IP is
	// attached, its port must be live (render.go).
	timeSource  = "hashicorp/time"
	timeVersion = "~> 0.14"
)

// Credential field names.
const (
	credAuthURL = "auth_url"
	credID      = "application_credential_id"
	credSecret  = "application_credential_secret"
)

// defaultVerifyTimeout bounds one Keystone request.
const defaultVerifyTimeout = 20 * time.Second

// Connector is the OpenStack connector. Its only state between calls is the
// option-list cache (names from each connection's cloud, never a credential
// or a token), and it is safe for concurrent use.
type Connector struct {
	tf            *terraform.Runner
	client        *http.Client
	verifyTimeout time.Duration
	allowPrivate  bool
	options       optionsCache
}

// Option changes how New builds the connector.
type Option func(*Connector)

// AllowPrivateTargets lets an auth URL be plain http and verification reach
// loopback, private and shared addresses, where a laptop's DevStack lives.
// Only A2 sets it (ADAPTER_ALLOW_PRIVATE_TARGETS, refused unless credentials are
// stored locally): in the cloud it would let a connection probe the network
// B runs in.
func AllowPrivateTargets() Option {
	return func(c *Connector) { c.allowPrivate = true }
}

// New returns the OpenStack connector, running Terraform through tf.
func New(tf *terraform.Runner, opts ...Option) connectors.Connector { return newConnector(tf, opts...) }

func newConnector(tf *terraform.Runner, opts ...Option) *Connector {
	c := &Connector{tf: tf, verifyTimeout: defaultVerifyTimeout}
	for _, o := range opts {
		o(c)
	}
	c.client = newHTTPClient(c.allowPrivate)
	return c
}

// providers are the Terraform providers this connector renders for. The
// shared runner refuses any other.
var providers = []string{providerSource, timeSource}

// ID implements connectors.Connector.
func (*Connector) ID() string { return ID }

// Credentials implements connectors.Connector. Only an application credential
// is accepted; a password clouds.yaml entry is later work.
func (*Connector) Credentials() connectors.CredentialSpec {
	return connectors.CredentialSpec{
		Kind: connectors.CredentialStored,
		Fields: []connectors.CredentialField{
			{Name: credAuthURL, Label: "Auth URL", Required: true,
				Help: "Your cloud's Identity (Keystone) v3 address, ending in /v3, for example https://cloud.example.com:5000/v3."},
			{Name: credID, Label: "Application credential ID", Required: true,
				Help: "Create an application credential in your cloud's dashboard (Identity, Application Credentials) or with the openstack CLI, and paste its ID."},
			{Name: credSecret, Label: "Application credential secret", Secret: true, Required: true,
				Help: "The secret shown once when the application credential was created."},
		},
	}
}

// Validate implements connectors.Connector. After the connector's own checks
// it renders the files and runs the shared runner's check on them, so
// Terraform the runner would refuse is reported before a run.
func (c *Connector) Validate(g connectors.Graph, d connectors.Deployment) []connectors.Problem {
	m, problems := build(g, d)
	if len(problems) > 0 {
		return problems
	}
	if len(m.crossProblems) > 0 {
		// CrossCheck reports them; nothing renders until they are fixed.
		return nil
	}
	if err := terraform.CheckFiles(renderFiles(m), providers); err != nil {
		return []connectors.Problem{{Code: connectors.CodeRenderRefused, Message: connectors.MsgRenderRefused}}
	}
	return nil
}

// Advice implements connectors.Advisor: each public VM nothing can reach.
func (c *Connector) Advice(g connectors.Graph, d connectors.Deployment) []connectors.Problem {
	return connectors.ClosedPublicVMs(g, d)
}

var _ connectors.Advisor = (*Connector)(nil)

// Render returns the Terraform files for deployment d of graph g. It refuses
// a graph Validate refuses, so it never renders a value that was not checked;
// B's core runs the capability check before it calls Render through a run or
// a preview.
func (c *Connector) Render(g connectors.Graph, d connectors.Deployment) (map[string][]byte, error) {
	m, err := c.model(g, d)
	if err != nil {
		return nil, err
	}
	return renderFiles(m), nil
}

// model builds and checks the model, refusing any problem.
func (c *Connector) model(g connectors.Graph, d connectors.Deployment) (model, error) {
	m, problems := build(g, d)
	if len(problems) > 0 || len(m.crossProblems) > 0 {
		return model{}, connectors.ErrCannotBuild
	}
	return m, nil
}

// Preview implements connectors.Previewer. Rendering reads no credentials. On
// a part of a split deployment every value another cloud hands a VM is a
// placeholder, so no other cloud's key is ever in a preview.
func (c *Connector) Preview(_ context.Context, g connectors.Graph, d connectors.Deployment) (map[string][]byte, error) {
	return c.Render(g, connectors.ForPreview(d))
}

var _ connectors.Previewer = (*Connector)(nil)

// CrossCheck implements connectors.CrossCloud: OpenStack hands another cloud
// a public VM's address, and its VMs take any value another cloud hands them.
func (c *Connector) CrossCheck(g connectors.Graph, d connectors.Deployment) []connectors.Problem {
	m, _ := build(g, d)
	return m.crossProblems
}

// PlanCross implements connectors.CrossCloud. Its outputs are the exports'
// values the part's state holds from its last apply.
func (c *Connector) PlanCross(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.PlanResult, connectors.Outputs, error) {
	m, err := c.model(g, d)
	if err != nil {
		return connectors.PlanResult{}, nil, err
	}
	var outputs bytes.Buffer
	out, err := c.terraform(ctx, run, connectors.ActionPlan, d, renderFiles(m), connectors.ExportsWriter(d, &outputs))
	if err != nil {
		return connectors.PlanResult{}, nil, err
	}
	res, err := out.planResult()
	if err != nil {
		return connectors.PlanResult{}, nil, err
	}
	exports, err := connectors.ReadExports(outputs.Bytes(), g, d, false, crossRules.Cloud)
	return res, exports, err
}

// ApplyCross implements connectors.CrossCloud. Every import must carry its
// values: an apply never renders a placeholder.
func (c *Connector) ApplyCross(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.RunResult, connectors.Outputs, error) {
	m, err := c.model(g, d)
	if err != nil {
		return connectors.RunResult{}, nil, fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
	}
	if m.missingImport {
		return connectors.RunResult{}, nil, fmt.Errorf("%w: %w", connectors.ErrNothingBuilt,
			&connectors.UserError{Message: "A value another cloud should hand this deployment is missing. Apply the deployment again."})
	}
	var outputs bytes.Buffer
	out, err := c.terraform(ctx, run, connectors.ActionApply, d, renderFiles(m), &outputs)
	if err != nil {
		return connectors.RunResult{}, nil, err
	}
	exports, err := connectors.ReadExports(outputs.Bytes(), g, d, true, crossRules.Cloud)
	if err != nil {
		return connectors.RunResult{}, nil, err
	}
	return connectors.RunResult{Resources: out.resources(), Results: connectors.ReadResults(outputs.Bytes(), g, d)}, exports, nil
}

var _ connectors.CrossCloud = (*Connector)(nil)

// Plan implements connectors.Connector.
func (c *Connector) Plan(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.PlanResult, error) {
	files, err := c.Render(g, d)
	if err != nil {
		return connectors.PlanResult{}, err
	}
	out, err := c.terraform(ctx, run, connectors.ActionPlan, d, files, nil)
	if err != nil {
		return connectors.PlanResult{}, err
	}
	return out.planResult()
}

// Apply implements connectors.Connector. Its result maps each block's node id
// to the id the cloud gave it, read from Terraform's outputs in the run's log,
// and holds each block's results, read from `terraform output -json`.
func (c *Connector) Apply(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.RunResult, error) {
	files, err := c.Render(g, d)
	if err != nil {
		return connectors.RunResult{}, err
	}
	var outputs bytes.Buffer
	out, err := c.terraform(ctx, run, connectors.ActionApply, d, files, &outputs)
	if err != nil {
		return connectors.RunResult{}, err
	}
	return connectors.RunResult{Resources: out.resources(), Results: connectors.ReadResults(outputs.Bytes(), g, d)}, nil
}

// Destroy implements connectors.Connector. It needs only the provider
// configuration: Terraform destroys everything the deployment's state holds.
func (c *Connector) Destroy(ctx context.Context, run connectors.Run, d connectors.Deployment) (connectors.RunResult, error) {
	if !connectors.ValidPrefix(d.Prefix) {
		return connectors.RunResult{}, errors.New("the deployment's name prefix is not valid")
	}
	if _, err := c.terraform(ctx, run, connectors.ActionDestroy, d, map[string][]byte{"versions.tf": versionsFile()}, nil); err != nil {
		return connectors.RunResult{}, err
	}
	return connectors.RunResult{Resources: map[string]string{}}, nil
}

// terraform runs one action through the shared runner and returns what it
// captured from Terraform's output. outputs, when set, receives Terraform's
// outputs as JSON after a plan or apply; they never reach the log.
func (c *Connector) terraform(ctx context.Context, run connectors.Run, a connectors.Action, d connectors.Deployment, files map[string][]byte, outputs io.Writer) (_ *capture, err error) {
	reachedTerraform := false
	defer func() {
		if err != nil && !reachedTerraform {
			err = fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
		}
	}()
	if c.tf == nil {
		return nil, errors.New("the Terraform runner is not configured")
	}
	if run.Action != "" && run.Action != a {
		return nil, fmt.Errorf("the run is a %s, not a %s", run.Action, a)
	}
	env, err := runEnv(c.Credentials(), d.Connection, c.allowPrivate)
	if err != nil {
		return nil, err
	}
	publicHosts, err := runPublicHosts(env, c.allowPrivate)
	if err != nil {
		return nil, err
	}
	log := run.Log
	if log == nil {
		log = io.Discard
	}
	out := newCapture(log)
	reachedTerraform = true
	err = c.tf.Run(ctx, terraform.Job{
		DeploymentID: d.ID,
		RunID:        run.ID,
		Action:       a,
		Files:        files,
		Providers:    providers,
		Env:          env,
		PublicHosts:  publicHosts,
		Log:          out,
		Outputs:      outputs,
	})
	if ferr := out.flush(); err == nil {
		err = ferr
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// runEnv is the run's environment: the provider reads these four variables, so
// the rendered provider block is empty and no file holds a credential.
func runEnv(spec connectors.CredentialSpec, conn connectors.Connection, allowHTTP bool) (map[string]string, error) {
	if err := spec.CheckSupplied(conn.Credentials); err != nil {
		return nil, err
	}
	authURL, err := normalizeAuthURL(conn.Credentials[credAuthURL], allowHTTP)
	if err != nil {
		return nil, err
	}
	if !connectors.RegionPattern.MatchString(conn.Region) {
		return nil, &connectors.UserError{Message: "The connection's region name is not valid."}
	}
	return map[string]string{
		"OS_AUTH_URL":                      authURL,
		"OS_REGION_NAME":                   conn.Region,
		"OS_APPLICATION_CREDENTIAL_ID":     conn.Credentials[credID],
		"OS_APPLICATION_CREDENTIAL_SECRET": conn.Credentials[credSecret],
	}, nil
}

// runPublicHosts is the run's terraform.Job.PublicHosts: the auth URL's host,
// which an isolated run task resolves again before Terraform connects
// (VerifyConnection checked it when the connection was linked, and DNS can
// change since). None when private targets are allowed: a laptop's DevStack
// is private by design.
func runPublicHosts(env map[string]string, allowPrivate bool) ([]string, error) {
	if allowPrivate {
		return nil, nil
	}
	u, err := url.Parse(env["OS_AUTH_URL"])
	if err != nil || u.Hostname() == "" {
		return nil, errors.New("the run's auth URL has no host")
	}
	return []string{u.Hostname()}, nil
}

// capture passes Terraform's output through to the run's log and keeps what
// the connector reads back: the plan summary (terraform.PlanSummary), the
// id outputs, and the blocks whose main resource the apply created or
// changed (terraform.ChangedResource).
type capture struct {
	mu    sync.Mutex
	w     io.Writer
	lines *terraform.LineWriter

	plan     connectors.PlanResult
	planSeen bool
	ids      map[string]string // node id -> cloud id, from the id outputs
	changed  map[string]bool   // node ids
}

// primaryTypes are the resource types that stand for a block in an apply's
// count: each block kind's main resource, at the block's address (addr).
var primaryTypes = []string{tInstance, tNetwork, tSecgroup, tVolume}

var idOutput = regexp.MustCompile(`^` + outputIDPrefix + `([A-Za-z0-9_-]{1,64}) = "([^"\\]*)"\s*$`)

func newCapture(w io.Writer) *capture {
	c := &capture{w: w, ids: map[string]string{}, changed: map[string]bool{}}
	c.lines = terraform.NewLineWriter(c.keep)
	return c
}

func (c *capture) Write(p []byte) (int, error) {
	if _, err := c.w.Write(p); err != nil {
		return 0, err
	}
	return c.lines.Write(p)
}

func (c *capture) flush() error { return c.lines.Flush() }

func (c *capture) keep(line string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	line = strings.TrimRight(line, "\r\n")
	if res, ok := terraform.PlanSummary(line); ok && !c.planSeen {
		c.plan, c.planSeen = res, true
	}
	if m := idOutput.FindStringSubmatch(line); m != nil {
		c.ids[m[1]] = m[2]
	}
	if a, _, ok := terraform.ChangedResource(line); ok {
		for _, t := range primaryTypes {
			if id, found := strings.CutPrefix(a, t+"."+addr("")); found {
				c.changed[id] = true
			}
		}
	}
	return nil
}

// resources maps each block the apply created or changed to its cloud id.
func (c *capture) resources() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]string{}
	for k, v := range c.ids {
		if c.changed[k] {
			out[k] = v
		}
	}
	return out
}

func (c *capture) planResult() (connectors.PlanResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.planSeen {
		return connectors.PlanResult{}, errors.New("terraform plan finished without a plan summary")
	}
	return c.plan, nil
}

var _ connectors.Connector = (*Connector)(nil)
