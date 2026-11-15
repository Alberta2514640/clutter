// Package aws is B's AWS connector (id "aws"). It reuses A's customer
// templates (ec2, s3, lambda, dynamodb, apigateway) unchanged, adds network,
// firewall and volume templates beside them, and owns everything around them:
// the variable maps from building blocks, validation and escaping, the
// edge-to-IAM step, API routes, and override files where A's templates lack
// an input (docs/architecture/system-b-adapter-layer.md § AWS).
//
// A connection is a role in the user's account that trusts B's runner role,
// and its external id. Each verify and each run asks STS for that role's
// short-lived keys, which reach Terraform only through the run's environment.
//
// The connector reads Clutter's database only through LinkedRoles, a read-only
// adapter over A's linked roles that filters on the organization, and only
// when the composition root hands it one (WithLinkedRoles).
package aws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
)

// ID is the AWS connector's id.
const ID = "aws"

// RoleTemplateVersion is the version of the CloudFormation role template
// (backend/infra/modules/s3/uploads/client_side_terraform_deployer_role.yaml,
// output RoleTemplateVersion) this connector expects. A test keeps the two
// equal. A connection records it, so a later change to the role can tell
// which users must update their stack.
const RoleTemplateVersion = "9"

// Session lengths. A run's keys last one hour, the most STS gives when one
// role assumes another (B's runner role assuming the user's role).
const (
	verifyDuration = 15 * time.Minute
	runDuration    = time.Hour
)

// Connector is the AWS connector. It keeps no state between calls.
type Connector struct {
	tf        *terraform.Runner
	templates TemplateSource
	sts       Assumer
	linked    *LinkedRoles
	// endpoint, when set (WithEndpoint), receives every AWS call a run makes.
	endpoint string
}

// New returns the AWS connector. tf runs Terraform; templates serves the
// templates under templates/aws (S3 in the cloud, a folder on disk locally);
// sts gets a role's short-lived keys.
func New(tf *terraform.Runner, templates TemplateSource, sts Assumer) *Connector {
	return &Connector{tf: tf, templates: templates, sts: sts}
}

// WithLinkedRoles lets the connector offer the roles an organization linked
// through A's CloudFormation flow as ready connections (connectors.Linker).
// Without it, Linked lists none.
func (c *Connector) WithLinkedRoles(l LinkedRoles) *Connector {
	c.linked = &l
	return c
}

// Linked implements connectors.Linker: the organization's completed role
// links, by account name. A role link has no region; the user picks one.
func (c *Connector) Linked(ctx context.Context, orgID string) ([]connectors.LinkedAccount, error) {
	out := []connectors.LinkedAccount{}
	if c.linked == nil {
		return out, nil
	}
	roles, err := c.linked.List(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, r := range roles {
		out = append(out, connectors.LinkedAccount{ID: r.ID, Name: r.AccountName})
	}
	return out, nil
}

// LinkedSecrets implements connectors.Linker: one of the organization's
// completed role links, as this connector's credentials. Its template version
// is empty: A's link flow does not record which RoleTemplateVersion the
// user's stack was created from, so B cannot know it.
func (c *Connector) LinkedSecrets(ctx context.Context, orgID, linkedID string) (connectors.LinkedCredentials, error) {
	if c.linked == nil {
		return connectors.LinkedCredentials{}, connectors.ErrLinkedNotFound
	}
	s, err := c.linked.Credentials(ctx, orgID, linkedID)
	if errors.Is(err, ErrLinkedRoleNotFound) {
		return connectors.LinkedCredentials{}, connectors.ErrLinkedNotFound
	}
	if err != nil {
		return connectors.LinkedCredentials{}, err
	}
	return connectors.LinkedCredentials{Secrets: s}, nil
}

var _ connectors.Linker = (*Connector)(nil)

// WithEndpoint sends every AWS call of every run to one endpoint on this
// machine, such as LocalStack (backend/devserver/localstack.sh), instead of
// AWS: the rendered files gain an override file that names it for each
// service, and the run's environment gains AWS_ENDPOINT_URL. The endpoint
// must be http or https on a loopback host, with no path, so the option can
// never point a run at another machine. main.go allows it only with the
// local credential store.
func (c *Connector) WithEndpoint(raw string) (*Connector, error) {
	ep, err := localEndpoint(raw)
	if err != nil {
		return nil, err
	}
	c.endpoint = ep
	return c, nil
}

// withEndpoint adds the endpoint override to a run's files, when the connector
// has an endpoint.
func (c *Connector) withEndpoint(files map[string][]byte) (map[string][]byte, error) {
	if c.endpoint == "" {
		return files, nil
	}
	s, err := executeOwn("endpoint_override.tf.tmpl", map[string]any{"Endpoint": hclsafe.Escape(c.endpoint)})
	if err != nil {
		return nil, err
	}
	files["endpoint_override.tf"] = []byte(s)
	return files, nil
}

// render is the files a run of the model applies.
func (c *Connector) render(ctx context.Context, m *model) (map[string][]byte, error) {
	files, err := m.render(ctx, c.templates)
	if err != nil {
		return nil, err
	}
	results, err := executeOwn("results.tf.tmpl", map[string]any{"Outputs": m.resultOutputs(c.endpoint)})
	if err != nil {
		return nil, err
	}
	files["results.tf"] = []byte(results)
	return c.withEndpoint(files)
}

// ID implements connectors.Connector.
func (*Connector) ID() string { return ID }

// Manifest implements connectors.Connector.
func (*Connector) Manifest() connectors.Manifest { return manifest() }

// Credentials implements connectors.Connector. A connection is made only from
// a role the organization linked (LinkedOnly): the external id is in the
// stack link every member of the organization sees, so a pasted pair could be
// another organization's role, linked by someone who has since left it
// (ADR-019).
func (*Connector) Credentials() connectors.CredentialSpec {
	return connectors.CredentialSpec{
		Kind:       connectors.CredentialBorrowed,
		LinkedOnly: true,
		Fields: []connectors.CredentialField{
			{Name: CredRoleARN, Label: "Role ARN", Help: "The RoleArn output of the Clutter CloudFormation stack in your account.", Required: true},
			{Name: CredExternalID, Label: "External id", Help: "The external id Clutter gave the stack when it created the stack link.", Secret: true, Required: true},
		},
	}
}

// providers are the Terraform providers this connector renders for. The
// shared runner refuses any other.
var providers = []string{"hashicorp/aws", "hashicorp/archive", "hashicorp/random"}

var (
	errBadRole   = &connectors.UserError{Message: "The role ARN must look like arn:aws:iam::123456789012:role/name, and the external id must be filled in."}
	errRefused   = &connectors.UserError{Message: "AWS did not let Clutter use this role. Check that the Clutter CloudFormation stack finished in your account and that the role ARN is its RoleArn output."}
	errBadRegion = &connectors.UserError{Message: "AWS does not offer this region through Clutter. Choose another region."}
	errTerraform = &connectors.UserError{Message: "Terraform stopped with an error. The run's log says why."}
)

// Preview implements connectors.Previewer: the Terraform a run would apply. It
// renders without assuming any role, so it needs no credential and holds none.
// On a part of a split deployment, every value another cloud hands a VM is a
// placeholder, so no other cloud's key is ever in a preview.
func (c *Connector) Preview(ctx context.Context, g connectors.Graph, d connectors.Deployment) (map[string][]byte, error) {
	m := newModel(g, connectors.ForPreview(d))
	if len(m.problems) > 0 || len(m.crossProblems) > 0 {
		return nil, connectors.ErrCannotBuild
	}
	return c.render(ctx, m)
}

// CrossCheck implements connectors.CrossCloud: AWS hands another cloud a
// public VM's address or a key that can only read a bucket or table, and its
// VMs take any such values at first boot.
func (c *Connector) CrossCheck(g connectors.Graph, d connectors.Deployment) []connectors.Problem {
	return newModel(g, d).crossProblems
}

// PlanCross implements connectors.CrossCloud. Its outputs are the exports'
// values the part's state holds from its last apply.
func (c *Connector) PlanCross(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.PlanResult, connectors.Outputs, error) {
	var outputs bytes.Buffer
	log, err := c.run(ctx, run, &g, d, connectors.ExportsWriter(d, &outputs))
	if err != nil {
		return connectors.PlanResult{}, nil, err
	}
	if !log.planSeen {
		return connectors.PlanResult{}, nil, errTerraform
	}
	out, err := connectors.ReadExports(outputs.Bytes(), g, d, false, crossRules.Cloud)
	return log.plan, out, err
}

// ApplyCross implements connectors.CrossCloud. Every import must carry its
// values: an apply never renders a placeholder.
func (c *Connector) ApplyCross(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.RunResult, connectors.Outputs, error) {
	if newModel(g, d).missingImports() {
		return connectors.RunResult{}, nil, fmt.Errorf("%w: %w", connectors.ErrNothingBuilt,
			&connectors.UserError{Message: "A value another cloud should hand this deployment is missing. Apply the deployment again."})
	}
	var outputs bytes.Buffer
	res, err := c.apply(ctx, run, g, d, &outputs)
	if err != nil {
		return connectors.RunResult{}, nil, err
	}
	out, err := connectors.ReadExports(outputs.Bytes(), g, d, true, crossRules.Cloud)
	if err != nil {
		return connectors.RunResult{}, nil, err
	}
	return res, out, nil
}

var _ connectors.CrossCloud = (*Connector)(nil)

// assume checks the connection's credentials and asks STS for keys. Its
// errors are user errors that hold no credential and no AWS text.
func (c *Connector) assume(ctx context.Context, conn connectors.Connection, session string, d time.Duration) (SessionKeys, error) {
	if !manifest().AllowsRegion(conn.Region) {
		return SessionKeys{}, errBadRegion
	}
	arn, ext := conn.Credentials[CredRoleARN], conn.Credentials[CredExternalID]
	if !roleARNPattern.MatchString(arn) || !validExternalID(ext) {
		return SessionKeys{}, errBadRole
	}
	if c.sts == nil || !sessionPattern.MatchString(session) {
		return SessionKeys{}, errors.New("the AWS connector has no STS client")
	}
	keys, err := c.sts.AssumeRole(ctx, AssumeInput{RoleARN: arn, ExternalID: ext, SessionName: session, Duration: d})
	if err != nil {
		var userErr *connectors.UserError
		if errors.As(err, &userErr) {
			return SessionKeys{}, userErr
		}
		if ctx.Err() != nil {
			return SessionKeys{}, ctx.Err()
		}
		return SessionKeys{}, errRefused
	}
	if keys.AccessKeyID == "" || keys.SecretAccessKey == "" || keys.SessionToken == "" {
		return SessionKeys{}, errRefused
	}
	return keys, nil
}

// VerifyConnection implements connectors.Connector: the role must be one the
// Clutter stack created (so its permissions boundary exists), and it must let
// B's runner role in with this external id.
func (c *Connector) VerifyConnection(ctx context.Context, conn connectors.Connection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := boundaryARN(conn.Credentials[CredRoleARN]); err != nil {
		return err
	}
	_, err := c.assume(ctx, conn, verifySession, verifyDuration)
	return err
}

// Validate implements connectors.Connector. It checks AWS's own rules, and
// then renders the Terraform and runs the runner's check on it, so text the
// runner would refuse is reported before a run.
func (c *Connector) Validate(g connectors.Graph, d connectors.Deployment) []connectors.Problem {
	m := newModel(g, d)
	if len(m.problems) > 0 {
		return m.problems
	}
	if len(m.crossProblems) > 0 {
		// CrossCheck reports them; nothing can be rendered until they are fixed.
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	files, err := c.render(ctx, m)
	if err == nil {
		err = terraform.CheckFiles(files, providers)
	}
	if errors.Is(err, terraform.ErrRefused) {
		return []connectors.Problem{{Code: connectors.CodeRenderRefused, Message: connectors.MsgRenderRefused}}
	}
	if err != nil {
		return []connectors.Problem{{Code: connectors.CodeRenderRefused,
			Message: "Clutter could not prepare the Terraform for this deployment. Try again later."}}
	}
	return nil
}

// Plan implements connectors.Connector.
func (c *Connector) Plan(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.PlanResult, error) {
	log, err := c.run(ctx, run, &g, d, nil)
	if err != nil {
		return connectors.PlanResult{}, err
	}
	if !log.planSeen {
		return connectors.PlanResult{}, errTerraform
	}
	return log.plan, nil
}

// Apply implements connectors.Connector.
func (c *Connector) Apply(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.RunResult, error) {
	var outputs bytes.Buffer
	return c.apply(ctx, run, g, d, &outputs)
}

// apply applies g for d. outputs receives the run's outputs, and the result's
// Results are read from them.
func (c *Connector) apply(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment, outputs *bytes.Buffer) (connectors.RunResult, error) {
	m := newModel(g, d)
	log, err := c.run(ctx, run, &g, d, outputs)
	if err != nil {
		return connectors.RunResult{}, err
	}
	res := connectors.RunResult{Resources: map[string]string{}, Results: connectors.ReadResults(outputs.Bytes(), g, d)}
	for addr, nodeID := range m.primaryAddresses() {
		if id, ok := log.created[addr]; ok {
			res.Resources[nodeID] = id
		}
	}
	return res, nil
}

// Destroy implements connectors.Connector. It needs only the provider
// configuration: Terraform destroys what the deployment's state holds.
func (c *Connector) Destroy(ctx context.Context, run connectors.Run, d connectors.Deployment) (connectors.RunResult, error) {
	if _, err := c.run(ctx, run, nil, d, nil); err != nil {
		return connectors.RunResult{}, err
	}
	return connectors.RunResult{Resources: map[string]string{}}, nil
}

// run renders the files (only the provider for a destroy, when g is nil),
// gets the run's keys and hands both to the Terraform runner. outputs, when
// set, receives Terraform's outputs after a plan or apply; they never reach
// the log.
func (c *Connector) run(ctx context.Context, run connectors.Run, g *connectors.Graph, d connectors.Deployment, outputs io.Writer) (_ *runLog, err error) {
	reachedTerraform := false
	defer func() {
		if err != nil && !reachedTerraform {
			err = fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
		}
	}()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.tf == nil {
		return nil, errors.New("the AWS connector has no Terraform runner")
	}
	var files map[string][]byte
	var archives []terraform.Archive
	if g != nil {
		m := newModel(*g, d)
		if len(m.problems) > 0 || len(m.crossProblems) > 0 {
			return nil, connectors.ErrCannotBuild
		}
		var err error
		if files, err = c.render(ctx, m); err != nil {
			return nil, fmt.Errorf("render: %w", err)
		}
		archives = m.archives()
	} else {
		p, err := providerFile(d)
		if err != nil {
			return nil, err
		}
		if files, err = c.withEndpoint(map[string][]byte{"main.tf": p}); err != nil {
			return nil, err
		}
	}

	// Every role and every user a run creates must name the stack's
	// permissions boundary for its kind.
	boundary, err := boundaryARN(d.Connection.Credentials[CredRoleARN])
	if err != nil {
		return nil, err
	}
	userBoundary, err := userBoundaryARN(d.Connection.Credentials[CredRoleARN])
	if err != nil {
		return nil, err
	}
	keys, err := c.assume(ctx, d.Connection, sessionName(run.ID), runDuration)
	if err != nil {
		return nil, err
	}
	log := newRunLog(run.Log, keys)
	env := map[string]string{
		"AWS_ACCESS_KEY_ID":     keys.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY": keys.SecretAccessKey,
		"AWS_SESSION_TOKEN":     keys.SessionToken,
		"AWS_REGION":            d.Connection.Region,
		boundaryEnv:             boundary,
		userBoundaryEnv:         userBoundary,
		// Never fall back to the credentials of the machine B runs on.
		"AWS_EC2_METADATA_DISABLED": "true",
	}
	if c.endpoint != "" {
		// Any service the override file does not name goes there too.
		env["AWS_ENDPOINT_URL"] = c.endpoint
	}
	reachedTerraform = true
	err = c.tf.Run(ctx, terraform.Job{
		DeploymentID: d.ID,
		RunID:        run.ID,
		Action:       run.Action,
		Files:        files,
		Archives:     archives,
		Providers:    providers,
		Env:          env,
		Log:          log,
		Outputs:      outputs,
	})
	if ferr := log.flush(); ferr != nil && err == nil {
		err = ferr
	}
	if err != nil {
		// The runner marks a failure before plan, apply or destroy started
		// (a refusal, the lock wait, terraform init); keep that mark.
		nothingBuilt := errors.Is(err, connectors.ErrNothingBuilt)
		var userErr *connectors.UserError
		switch {
		case errors.As(err, &userErr):
			// The runner's own message, such as code that changed after the
			// run was queued.
			err = userErr
		case errors.Is(err, terraform.ErrRefused):
			err = &connectors.UserError{Message: "Clutter refused to run this Terraform. Check the deployment's compatibility."}
		case ctx.Err() != nil:
			err = ctx.Err()
		default:
			err = fmt.Errorf("%w (%s)", errTerraform, err.Error())
		}
		if nothingBuilt {
			err = fmt.Errorf("%w: %w", connectors.ErrNothingBuilt, err)
		}
		return nil, err
	}
	return log, nil
}

// sessionName names a run's STS session after the run, so the user's
// CloudTrail shows which Clutter run did what.
func sessionName(runID string) string {
	id := strings.ReplaceAll(runID, "-", "")
	if len(id) > 32 {
		id = id[:32]
	}
	return "clutter-adapter-" + id
}

var _ connectors.Connector = (*Connector)(nil)
