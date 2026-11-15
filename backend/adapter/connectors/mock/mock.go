// Package mock is a connector that builds nothing. It exists to prove B's core
// and the connector contract without a cloud: it accepts the four OpenStack
// blocks, refuses the other four, and records what it "built" in memory.
package mock

import (
	"context"
	"crypto/subtle"
	"fmt"
	"sort"
	"sync"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

// ID is the mock connector's id.
const ID = "mock"

// ValidToken is the only token VerifyConnection accepts.
const ValidToken = "mock-valid-token"

// Connector is the mock connector. It is safe for concurrent use.
type Connector struct {
	mu    sync.Mutex
	built map[string]map[string]string // deployment id -> node id -> fake cloud id
	// imported holds what each deployment's last apply was handed by other
	// clouds (connectors.CrossCloud), by edge key and value name.
	imported map[string]map[string]map[string]connectors.Value
}

// New returns a mock connector with nothing built.
func New() *Connector {
	return &Connector{built: map[string]map[string]string{}, imported: map[string]map[string]map[string]connectors.Value{}}
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
		Display:   connectors.Display{ShortName: "Mock", Logo: "logo"},
		Blocks: []connectors.BlockSpec{
			connectors.StandardBlock(connectors.KindVM, "VM").WithDisplay("VM", ""),
			connectors.StandardBlock(connectors.KindNetwork, "Network").WithDisplay("Network", ""),
			connectors.StandardBlock(connectors.KindFirewall, "Firewall").WithDisplay("Firewall", ""),
			connectors.StandardBlock(connectors.KindVolume, "Volume").WithDisplay("Volume", ""),
			{Kind: connectors.KindBucket, Treatment: connectors.TreatmentRemoved, Title: "Bucket"},
			{Kind: connectors.KindFunction, Treatment: connectors.TreatmentRemoved, Title: "Function"},
			{Kind: connectors.KindTable, Treatment: connectors.TreatmentRemoved, Title: "Table"},
			{Kind: connectors.KindHTTPAPI, Treatment: connectors.TreatmentRemoved, Title: "HTTP API"},
		},
		Edges: []connectors.EdgeSpec{
			{Source: connectors.KindVM, Target: connectors.KindNetwork, Meaning: "the VM is attached to the network", Why: "Servers in one network can talk to each other privately."},
			{Source: connectors.KindVolume, Target: connectors.KindVM, Meaning: "the volume is attached to the VM", Why: "Gives the server more room for its data."},
			{Source: connectors.KindFirewall, Target: connectors.KindVM, Meaning: "the firewall rules apply to the VM", Why: "Only the traffic you allow can reach the server."},
			{Source: connectors.KindVM, Target: connectors.KindVM, Meaning: "the VM gets the other VM's address when it first starts", Why: "Your app on the server can find the other server, such as its database."},
		},
		Settings: []connectors.FieldSpec{
			{Name: "image", Title: "Image", Type: connectors.FieldEnum, Default: "ubuntu-24.04", Allowed: []string{"ubuntu-24.04", "debian-12"}},
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
	want := included(g, d)
	for _, id := range want {
		if _, ok := have[id]; ok {
			res.Change++
			fmt.Fprintf(run.Log, "  ~ %s.%s (%s)\n", d.Prefix, id, g.Blocks[id].Kind)
		} else {
			res.Add++
			fmt.Fprintf(run.Log, "  + %s.%s (%s)\n", d.Prefix, id, g.Blocks[id].Kind)
		}
	}
	for id := range have {
		if _, ok := g.Blocks[id]; !ok || d.Excluded(id) {
			res.Destroy++
			fmt.Fprintf(run.Log, "  - %s.%s\n", d.Prefix, id)
		}
	}
	return res, nil
}

// Apply implements connectors.Connector.
func (c *Connector) Apply(ctx context.Context, run connectors.Run, g connectors.Graph, d connectors.Deployment) (connectors.RunResult, error) {
	if err := ctx.Err(); err != nil {
		return connectors.RunResult{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	built := map[string]string{}
	results := map[string][]connectors.Result{}
	for _, id := range included(g, d) {
		built[id] = fmt.Sprintf("mock-%s-%s", d.Prefix, id)
		fmt.Fprintf(run.Log, "Created %s (%s).\n", built[id], g.Blocks[id].Kind)
		// A public VM gets the mock's fake address, as its exports do.
		if b := g.Blocks[id]; b.Kind == connectors.KindVM && b.Fields["public"] == true {
			results[id] = []connectors.Result{{Name: connectors.ResultPublicAddress, Value: address(d.ID, id)}}
		}
	}
	c.built[d.ID] = built
	return connectors.RunResult{Resources: copyMap(built), Results: results}, nil
}

// Destroy implements connectors.Connector.
func (c *Connector) Destroy(ctx context.Context, run connectors.Run, d connectors.Deployment) (connectors.RunResult, error) {
	if err := ctx.Err(); err != nil {
		return connectors.RunResult{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	gone := c.built[d.ID]
	ids := make([]string, 0, len(gone))
	for id := range gone {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Fprintf(run.Log, "Destroyed %s.\n", gone[id])
	}
	delete(c.built, d.ID)
	return connectors.RunResult{Resources: copyMap(gone)}, nil
}

// Built returns what the mock holds for a deployment, for tests.
func (c *Connector) Built(deploymentID string) map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return copyMap(c.built[deploymentID])
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
			for name, v := range imp.Values {
				fmt.Fprintf(run.Log, "Handed %s.%s %s.\n", imp.Source, name, v)
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

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

var _ connectors.Connector = (*Connector)(nil)
