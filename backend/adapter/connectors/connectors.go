// Package connectors is the contract between B's core and each target cloud.
//
// A connector implements Connector and is added to a Registry by the program's
// composition root. B's core asks the registry for a deployment's connector and
// calls the interface; it never branches on a connector's id.
//
// A connector may not write Clutter's database, keep credentials after a run,
// or put a secret in a log, a Problem or a result. It may not read Clutter's
// database either, with one exception: the AWS connector reads A's linked
// roles (aws_account_access_roles), read-only and filtered on the
// organization, through the adapter the composition root hands it. An AWS
// connection is made only from a role the organization linked through A's
// CloudFormation flow, never from pasted credentials (ADR-019, ADR-020), so
// the connector must read where A records those links.
package connectors

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
)

// BlockKind names a building block. The standard kinds are below; a connector
// may add its own kinds, marked TreatmentNew in its manifest.
type BlockKind string

// The standard building blocks every connector declares a treatment for.
const (
	KindVM       BlockKind = "vm"
	KindNetwork  BlockKind = "network"
	KindFirewall BlockKind = "firewall"
	KindVolume   BlockKind = "volume"
	KindBucket   BlockKind = "bucket"
	KindFunction BlockKind = "function"
	KindTable    BlockKind = "table"
	KindHTTPAPI  BlockKind = "http_api"
)

// StandardBlocks returns the standard building blocks in a fixed order.
func StandardBlocks() []BlockKind {
	return []BlockKind{KindVM, KindNetwork, KindFirewall, KindVolume, KindBucket, KindFunction, KindTable, KindHTTPAPI}
}

// IsStandard reports whether k is a standard building block.
func IsStandard(k BlockKind) bool {
	for _, s := range StandardBlocks() {
		if s == k {
			return true
		}
	}
	return false
}

// Graph is a diagram as building blocks, not tied to any cloud.
type Graph struct {
	// Blocks is keyed by the diagram's node id. Generated code addresses a
	// block by this id, so renaming a block does not re-create it.
	Blocks map[string]Block
	Edges  []Edge
}

// Block is one building block on the canvas.
type Block struct {
	ID     string
	Kind   BlockKind
	Name   string
	Fields map[string]any
	// Code is the zip a user uploaded for a function block, or nil. B's core
	// fills it from its own records, never from the diagram document.
	Code *FunctionCode
}

// Edge joins two blocks by node id. What it means depends on the two kinds
// (a VM on a network, a volume on a VM); the manifest lists the pairs a
// connector can wire.
type Edge struct {
	Source string
	Target string
}

// Secrets holds one connection's credentials for one verify or one run. It is
// never stored by a connector, and it prints and marshals as Redacted so a
// stray log line or response cannot carry it.
type Secrets map[string]string

// Redacted is what B shows in place of a secret: in a log line, a printed
// value or a response.
const Redacted = "[redacted]"

// String hides the values.
func (Secrets) String() string { return Redacted }

// GoString hides the values from %#v.
func (Secrets) GoString() string { return Redacted }

// MarshalJSON hides the values.
func (Secrets) MarshalJSON() ([]byte, error) { return []byte(`"` + Redacted + `"`), nil }

// Values returns the non-empty credential values, for redacting output.
func (s Secrets) Values() []string {
	out := make([]string, 0, len(s))
	for _, v := range s {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// Connection is one organization's linked cloud account as a connector sees it.
type Connection struct {
	ID          string
	ConnectorID string
	Region      string
	// Credentials are filled by B's broker for one verify or one run only.
	Credentials Secrets
}

// Deployment carries everything one run needs: its id (which keys the state),
// its cloud-name prefix, its connection, and its own settings and exclusions.
type Deployment struct {
	ID string
	// Prefix starts every cloud-side name, so two deployments, or A and B, can
	// share one cloud account without colliding. See PrefixFor.
	Prefix     string
	Connection Connection
	Settings   map[string]any
	// Exclusions are node ids left out of this deployment only. On a part of
	// a split deployment they also hold every block placed on another part.
	Exclusions []string
	// Cross is set only on a part of a split deployment (CrossCloud): the
	// cross-cloud edges this part exports and imports. ID is then the part's
	// own id, which keys its state, and Prefix is the whole deployment's.
	Cross *Cross
}

// Excluded reports whether the block with this node id is left out of d.
func (d Deployment) Excluded(nodeID string) bool {
	for _, e := range d.Exclusions {
		if e == nodeID {
			return true
		}
	}
	return false
}

// PrefixFor returns the short cloud-name prefix of a deployment: "c" and the
// first eight hex digits of its id, a UUID. It starts with a letter, so it is
// a valid start for every cloud's resource names.
func PrefixFor(deploymentID string) string {
	hex := strings.ReplaceAll(strings.ToLower(deploymentID), "-", "")
	if len(hex) > 8 {
		hex = hex[:8]
	}
	return "c" + hex
}

// prefixPattern is the shape of PrefixFor's result for a UUID.
var prefixPattern = regexp.MustCompile(`^c[0-9a-f]{8}$`)

// ValidPrefix reports whether p has the shape PrefixFor gives a deployment id.
// A connector checks a deployment's prefix with it before naming anything.
func ValidPrefix(p string) bool { return prefixPattern.MatchString(p) }

// Action is what a run does.
type Action string

// The run actions.
const (
	ActionPlan    Action = "plan"
	ActionApply   Action = "apply"
	ActionDestroy Action = "destroy"
)

// Valid reports whether a is one of the run actions.
func (a Action) Valid() bool {
	return a == ActionPlan || a == ActionApply || a == ActionDestroy
}

// Run is one plan, apply or destroy. Log is the run's log, owned by B's job
// runner, which stores it and removes the run's credential values from it.
// A connector writes its progress there.
type Run struct {
	ID     string
	Action Action
	Log    io.Writer
}

// Problem is a reason a graph cannot deploy as it is, or a note about it. Its
// message is shown to the user, so it never holds a secret or a raw cloud error.
type Problem struct {
	NodeID     string `json:"node_id,omitempty"`
	EdgeSource string `json:"edge_source,omitempty"`
	EdgeTarget string `json:"edge_target,omitempty"`
	Field      string `json:"field,omitempty"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

// PlanResult says what an apply would change.
type PlanResult struct {
	Add     int
	Change  int
	Destroy int
}

// RunResult says what an apply or destroy did.
type RunResult struct {
	// Resources maps a node id to the cloud's id, for each block whose main
	// resource the run created or changed. A block left as it was is not in it.
	Resources map[string]string
	// Results maps a node id to what an apply returns about that block, such
	// as a server's address (ResultNames). Every block the deployment holds
	// after the apply may have some, built in this run or before. Results are
	// shown to the user, so they never hold a secret.
	Results map[string][]Result
}

// ErrNothingBuilt marks a run that failed before Terraform could change
// anything in the cloud (bad credentials, a refused role, a diagram the cloud
// cannot build, rendered text the runner refuses, a failed terraform init). A
// connector wraps such an error with it (fmt.Errorf("%w: %w", ErrNothingBuilt,
// err)), and the shared Terraform runner wraps its own; the job runner
// then leaves the deployment's status as it was, so a failed first apply does
// not lock the connection against deletes.
var ErrNothingBuilt = errors.New("nothing was built")

// UserError is an error whose message is safe to show the user: it says what
// to do next and holds no secret and no raw cloud error.
type UserError struct {
	Message string
}

func (e *UserError) Error() string { return e.Message }

// UserMessage returns err's message when err is or wraps a *UserError, and
// fallback otherwise.
func UserMessage(err error, fallback string) string {
	var ue *UserError
	if errors.As(err, &ue) {
		return ue.Message
	}
	return fallback
}

// Connector is the one contract every target implements.
type Connector interface {
	// ID is the connector's stable id, such as "general-openstack".
	ID() string
	// Manifest says which blocks, fields, edges, settings and regions it accepts.
	Manifest() Manifest
	// Credentials says what a connection supplies and how the UI asks for it.
	Credentials() CredentialSpec
	// VerifyConnection proves a connection's credentials work. It runs before
	// a connection is saved and again before each run.
	VerifyConnection(ctx context.Context, c Connection) error
	// Validate returns the connector's own problems with g for d before
	// anything runs: its cloud's ranges and limits, its wiring rules, names
	// that would collide, and rendered text the runner would refuse. It
	// honours d.Exclusions. B's core runs the capability check (blocks,
	// edges, field types and allowed values against the manifest) beside it,
	// so Validate never repeats those, and it must not panic when they are
	// wrong.
	Validate(g Graph, d Deployment) []Problem
	Plan(ctx context.Context, run Run, g Graph, d Deployment) (PlanResult, error)
	Apply(ctx context.Context, run Run, g Graph, d Deployment) (RunResult, error)
	Destroy(ctx context.Context, run Run, d Deployment) (RunResult, error)
}

// Previewer is optional. A connector that renders Terraform implements it, so
// the deploy panel can show the files before a run. The core finds it by type
// assertion; a connector without it has no preview.
//
// The deployment it is handed carries no credentials, and the files it returns
// are shown to the user: they never hold a secret. It refuses a graph Validate
// refuses, with a *UserError.
type Previewer interface {
	Preview(ctx context.Context, g Graph, d Deployment) (map[string][]byte, error)
}

// Advisor is optional. A connector gives notes before a run: what its cloud
// bills for that the diagram does not show as a block (a gateway charged by
// the hour), with code CodeCost, so the deploy confirmation can name the
// cost; and what will not work as drawn, such as a public VM nothing can
// reach (ClosedPublicVMs), which every connector that builds VMs gives. The
// core finds it by type assertion. Notes never stop a run.
type Advisor interface {
	Advice(g Graph, d Deployment) []Problem
}

// LinkedAccount is an account an organization already linked to Clutter in
// the cloud's own way (for AWS, a role from Clutter's CloudFormation stack),
// offered as a ready connection. It holds no secret.
type LinkedAccount struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Region string `json:"region,omitempty"`
}

// ErrLinkedNotFound means the organization has no linked account with that id.
var ErrLinkedNotFound = errors.New("linked account not found")

// LinkedCredentials is what a linked account gives a new connection.
type LinkedCredentials struct {
	// Secrets are in the shape the connector's CredentialSpec asks for.
	Secrets Secrets
	// TemplateVersion is the version of the access template (for AWS, the
	// CloudFormation role template) the account was linked with, when the
	// cloud's link flow recorded it; empty when it is not known. The core
	// stores it on the connection, so a later template change can tell which
	// accounts must update. Nothing reads the stored value back yet, and the
	// aws connector leaves it empty (A's link flow does not record it).
	TemplateVersion string
}

// MaxTemplateVersion caps a LinkedCredentials.TemplateVersion's length.
const MaxTemplateVersion = 32

// Linker is optional. A connector whose cloud is linked outside B implements
// it, so a user can make a connection from a linked account without pasting
// its credentials. The core finds it by type assertion. A connector whose
// CredentialSpec is LinkedOnly must implement it.
//
// Both methods filter on orgID: another organization's linked account is
// ErrLinkedNotFound. The core verifies and stores LinkedSecrets' credentials
// as it does pasted ones, and they never reach the browser.
type Linker interface {
	Linked(ctx context.Context, orgID string) ([]LinkedAccount, error)
	LinkedSecrets(ctx context.Context, orgID, linkedID string) (LinkedCredentials, error)
}

// Choice is one value a deployment setting can take on a connection's cloud,
// such as a flavor name, with a label for the UI.
type Choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Options are the values a connection's cloud offers for the connector's
// deployment settings, keyed by setting name. A setting that is missing has
// no list (the cloud could not list it); an empty list means the cloud has
// none.
type Options struct {
	Settings map[string][]Choice `json:"settings"`
}

// OptionLister is optional. A connector whose settings name things that
// already exist in the user's cloud (an image, a flavor, a network)
// implements it, so the UI can offer them as a list instead of a text box.
// The core finds it by type assertion; a connector without it has no lists.
//
// The connection it is handed carries credentials, filled by B's broker for
// this one call. Values and labels are shown to the user, so they never hold
// a secret, and errors follow the same rules as VerifyConnection's. The core
// keeps only settings the manifest declares and values CheckValue accepts.
type OptionLister interface {
	Options(ctx context.Context, c Connection) (Options, error)
}
