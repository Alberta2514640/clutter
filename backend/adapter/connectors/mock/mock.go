// Package mock is a connector that builds nothing. It exists to prove B's core
// and the connector contract without a cloud: it accepts the four OpenStack
// blocks, refuses the other four, and records what it "built" in memory. Its
// log names each block as the diagram does ("Created web (VM)."), and an
// apply says which blocks it created, updated, left unchanged or removed.
package mock

import (
	"context"
	"crypto/subtle"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

// ID is the mock connector's id.
const ID = "mock"

// ValidToken is the only token VerifyConnection accepts.
const ValidToken = "mock-valid-token"

// FailName is the block name that makes the mock fail on purpose: a plan or an
// apply that reaches a block with exactly this name stops with an error, so
// the deploy panel's failure path can be tried without a real cloud. An apply
// keeps what it built before that block.
const FailName = "fail-me"

// held is one block the mock holds for a deployment: its fake cloud id, how
// the log names it, and what it was built from, so a later apply can tell an
// unchanged block from a changed one.
type held struct {
	cloudID string
	label   string
	shape   string
}

// Connector is the mock connector. It is safe for concurrent use.
type Connector struct {
	mu    sync.Mutex
	built map[string]map[string]held // deployment id -> node id -> block
	// imported holds what each deployment's last apply was handed by other
	// clouds (connectors.CrossCloud), by edge key and value name.
	imported map[string]map[string]map[string]connectors.Value
	// titles are the manifest's display names by block kind ("VM").
	titles map[connectors.BlockKind]string
}

// New returns a mock connector with nothing built.
func New() *Connector {
	c := &Connector{built: map[string]map[string]held{}, imported: map[string]map[string]map[string]connectors.Value{}, titles: map[connectors.BlockKind]string{}}
	for _, b := range c.Manifest().Blocks {
		if b.Display != nil {
			c.titles[b.Kind] = b.Display.Name
		}
	}
	return c
}

// label is how the log names a block: its name and its kind's display name.
func (c *Connector) label(b connectors.Block) string {
	title := c.titles[b.Kind]
	if title == "" {
		title = string(b.Kind)
	}
	return fmt.Sprintf("%s (%s)", b.Name, title)
}

// shape is what a block was built from: its kind, name and fields, and the
// lines that touch it. fmt prints map keys in sorted order, so equal blocks
// give equal shapes.
func shape(g connectors.Graph, id string) string {
	b := g.Blocks[id]
	var lines []string
	for _, e := range g.Edges {
		if e.Source == id || e.Target == id {
			lines = append(lines, e.Source+">"+e.Target)
		}
	}
	sort.Strings(lines)
	return fmt.Sprintf("%s|%s|%v|%s", b.Kind, b.Name, b.Fields, strings.Join(lines, ","))
}

// failure is the error a block named FailName stops a run with.
func (c *Connector) failure(b connectors.Block) error {
	return &connectors.UserError{Message: fmt.Sprintf("The mock cloud failed on purpose at %s. It always fails a block named %s; rename the block to build it.", c.label(b), FailName)}
}

// ID implements connectors.Connector.
func (*Connector) ID() string { return ID }

// Manifest implements connectors.Connector.
func (*Connector) Manifest() connectors.Manifest {
	return connectors.Manifest{
		Connector: ID,
		Title:     "Mock cloud",
		Version:   "1",
		Regions:   []string{"mock-1", "mock-2"},
		Display:   connectors.Display{ShortName: "Mock", Logo: "logo", Note: "for testing; builds nothing"},
		Blocks: []connectors.BlockSpec{
			connectors.StandardBlock(connectors.KindVM).WithDisplay("VM", ""),
			connectors.StandardBlock(connectors.KindNetwork).WithDisplay("Network", ""),
			connectors.StandardBlock(connectors.KindFirewall).WithDisplay("Firewall", ""),
			connectors.StandardBlock(connectors.KindVolume).WithDisplay("Volume", ""),
			connectors.RemovedBlock(connectors.KindBucket),
			connectors.RemovedBlock(connectors.KindFunction),
			connectors.RemovedBlock(connectors.KindTable),
			connectors.RemovedBlock(connectors.KindHTTPAPI),
		},
		Edges: []connectors.EdgeSpec{
			{Source: connectors.KindVM, Target: connectors.KindNetwork, Meaning: "the VM is attached to the network", Why: "Servers in one network can talk to each other privately."},
			{Source: connectors.KindVolume, Target: connectors.KindVM, Meaning: "the volume is attached to the VM", Why: "Gives the server more room for its data."},
			{Source: connectors.KindFirewall, Target: connectors.KindVM, Meaning: "the firewall rules apply to the VM", Why: "Only the traffic you allow can reach the server."},
			{Source: connectors.KindVM, Target: connectors.KindVM, Meaning: "the VM gets the other VM's address when it first starts", Why: "Your app on the server can find the other server, such as its database."},
		},
		Settings: []connectors.FieldSpec{
			{Name: "image", Title: "Image", Type: connectors.FieldEnum, Default: "ubuntu-24.04", Allowed: []string{"ubuntu-24.04", "debian-12"},
				AppliesTo: []connectors.BlockKind{connectors.KindVM}},
		},
	}
}

// Credentials implements connectors.Connector.
func (*Connector) Credentials() connectors.CredentialSpec {
	return connectors.CredentialSpec{
		Kind: connectors.CredentialStored,
		Fields: []connectors.CredentialField{
			{Name: "token", Label: "Token", Help: "Any mock token. Only " + ValidToken + " works.", Secret: true, Required: true},
		},
	}
}

// VerifyConnection implements connectors.Connector.
func (c *Connector) VerifyConnection(ctx context.Context, conn connectors.Connection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !c.Manifest().AllowsRegion(conn.Region) {
		return &connectors.UserError{Message: "This region does not exist in the mock cloud."}
	}
	if subtle.ConstantTimeCompare([]byte(conn.Credentials["token"]), []byte(ValidToken)) != 1 {
		return &connectors.UserError{Message: "The mock cloud refused this token. Check it and try again."}
	}
	return nil
}

// Validate implements connectors.Connector. The mock cloud has no rules of
// its own beyond its manifest, which B's capability check holds a graph to.
func (c *Connector) Validate(connectors.Graph, connectors.Deployment) []connectors.Problem {
	return nil
}

// Plan implements connectors.Connector.
func (c *Connector) Plan(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.PlanResult, error) {
	if err := ctx.Err(); err != nil {
		return connectors.PlanResult{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	have := c.built[d.ID]
	var res connectors.PlanResult
	// Like Terraform's plan, it lists what would change and leaves out what
	// would not.
	for _, id := range included(g, d) {
		b := g.Blocks[id]
		if b.Name == FailName {
			return res, c.failure(b)
		}
		h, ok := have[id]
		switch {
		case !ok:
			res.Add++
			fmt.Fprintf(run.Log, "  + %s\n", c.label(b))
		case h.shape != shape(g, id):
			res.Change++
			fmt.Fprintf(run.Log, "  ~ %s\n", c.label(b))
		}
	}
	for _, id := range sortedKeys(have) {
		if _, ok := g.Blocks[id]; !ok || d.Excluded(id) {
			res.Destroy++
			fmt.Fprintf(run.Log, "  - %s\n", have[id].label)
		}
	}
	return res, nil
}

// Apply implements connectors.Connector. It reports as built or changed only
// the blocks it created or updated, as RunResult.Resources asks.
func (c *Connector) Apply(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.RunResult, error) {
	if err := ctx.Err(); err != nil {
		return connectors.RunResult{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	have := c.built[d.ID]
	next := map[string]held{}
	changed := map[string]string{}
	results := map[string][]connectors.Result{}
	for _, id := range included(g, d) {
		b := g.Blocks[id]
		if b.Name == FailName {
			// What it built before this block stays, with what it held.
			for k, v := range have {
				if _, ok := next[k]; !ok {
					next[k] = v
				}
			}
			c.built[d.ID] = next
			return connectors.RunResult{Resources: changed, Results: results}, c.failure(b)
		}
		h := held{cloudID: fmt.Sprintf("mock-%s-%s", d.Prefix, id), label: c.label(b), shape: shape(g, id)}
		old, ok := have[id]
		switch {
		case !ok:
			fmt.Fprintf(run.Log, "Created %s.\n", h.label)
			changed[id] = h.cloudID
		case old.shape != h.shape:
			fmt.Fprintf(run.Log, "Updated %s.\n", h.label)
			changed[id] = h.cloudID
		default:
			fmt.Fprintf(run.Log, "Unchanged %s.\n", h.label)
		}
		next[id] = h
		// A public VM gets the mock's fake address, as its exports do.
		if b.Kind == connectors.KindVM && b.Fields["public"] == true {
			results[id] = []connectors.Result{{Name: connectors.ResultPublicAddress, Value: address(d.ID, id)}}
		}
	}
	for _, id := range sortedKeys(have) {
		if _, ok := next[id]; !ok {
			fmt.Fprintf(run.Log, "Removed %s.\n", have[id].label)
		}
	}
	c.built[d.ID] = next
	return connectors.RunResult{Resources: changed, Results: results}, nil
}

// Destroy implements connectors.Connector.
func (c *Connector) Destroy(ctx context.Context, run connectors.Run, d connectors.Deployment) (connectors.RunResult, error) {
	if err := ctx.Err(); err != nil {
		return connectors.RunResult{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	gone := c.built[d.ID]
	ids := make(map[string]string, len(gone))
	for _, id := range sortedKeys(gone) {
		fmt.Fprintf(run.Log, "Destroyed %s.\n", gone[id].label)
		ids[id] = gone[id].cloudID
	}
	delete(c.built, d.ID)
	return connectors.RunResult{Resources: ids}, nil
}

// Built returns what the mock holds for a deployment, node id to fake cloud
// id, for tests.
func (c *Connector) Built(deploymentID string) map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]string{}
	for id, h := range c.built[deploymentID] {
		out[id] = h.cloudID
	}
	return out
}

// Imported returns what a deployment's last apply was handed by other clouds,
// for tests.
func (c *Connector) Imported(deploymentID string) map[string]map[string]connectors.Value {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]map[string]connectors.Value{}
	for k, vs := range c.imported[deploymentID] {
		out[k] = map[string]connectors.Value{}
		for n, v := range vs {
			out[k][n] = v
		}
	}
	return out
}

// CrossCheck implements connectors.CrossCloud. The mock hands out a VM's
// address, which needs the VM to be public, and creates no keys.
func (c *Connector) CrossCheck(g connectors.Graph, d connectors.Deployment) []connectors.Problem {
	return connectors.CheckCross(g, d, crossRules)
}

var crossRules = connectors.CrossRules{
	Cloud: "Mock",
	CanExport: func(class connectors.EdgeClass, target connectors.BlockKind) bool {
		return class == connectors.EdgeAddress && target == connectors.KindVM
	},
	ExportRefusal: "The mock cloud cannot create a key for another cloud.",
}

// address is the fake public address the mock gives a VM.
func address(deploymentID, nodeID string) string {
	n := 0
	for _, r := range deploymentID + nodeID {
		n = (n*31 + int(r)) % 250
	}
	return fmt.Sprintf("203.0.113.%d", n+1)
}

// outputs are the mock's exports for what it holds of d.
func (c *Connector) outputs(d connectors.Deployment) connectors.Outputs {
	out := connectors.Outputs{}
	if d.Cross == nil {
		return out
	}
	for _, e := range d.Cross.Exports {
		if _, ok := c.built[d.ID][e.Target]; ok && e.Class == connectors.EdgeAddress {
			out[e.Key] = map[string]connectors.Value{connectors.ValueAddress: {Text: address(d.ID, e.Target)}}
		}
	}
	return out
}

// PlanCross implements connectors.CrossCloud.
func (c *Connector) PlanCross(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.PlanResult, connectors.Outputs, error) {
	res, err := c.Plan(ctx, run, g, d)
	if err != nil {
		return res, nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return res, c.outputs(d), nil
}

// ApplyCross implements connectors.CrossCloud. It records what it was handed.
func (c *Connector) ApplyCross(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.RunResult, connectors.Outputs, error) {
	res, err := c.Apply(ctx, run, g, d)
	if err != nil {
		return res, nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	in := map[string]map[string]connectors.Value{}
	if d.Cross != nil {
		for _, imp := range d.Cross.Imports {
			in[imp.Key] = imp.Values
			from := imp.Target
			if b, ok := g.Blocks[imp.Target]; ok && b.Name != "" {
				from = b.Name
			}
			for _, name := range sortedKeys(imp.Values) {
				fmt.Fprintf(run.Log, "Received the %s of %s: %s.\n", strings.ReplaceAll(name, "_", " "), from, imp.Values[name])
			}
		}
	}
	c.imported[d.ID] = in
	return res, c.outputs(d), nil
}

var _ connectors.CrossCloud = (*Connector)(nil)

func included(g connectors.Graph, d connectors.Deployment) []string {
	ids := make([]string, 0, len(g.Blocks))
	for id := range g.Blocks {
		if !d.Excluded(id) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func sortedKeys[V any](m map[string]V) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

var _ connectors.Connector = (*Connector)(nil)
