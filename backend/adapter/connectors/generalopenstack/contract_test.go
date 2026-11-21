package generalopenstack

import (
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
)

// TestManifestHelpIsOpenStacks: the help a person reads for this cloud says
// what holds here, with no other cloud's names or prices, and warns where a
// change replaces a deployed server.
func TestManifestHelpIsOpenStacks(t *testing.T) {
	m := (&Connector{}).Manifest()
	if err := m.Check(); err != nil {
		t.Fatal(err)
	}
	help := func(kind connectors.BlockKind) string {
		b, ok := m.Block(kind)
		if !ok {
			t.Fatalf("no %s block", kind)
		}
		lines := b.Help
		for _, f := range b.Fields {
			lines = append(lines, f.Help)
		}
		return strings.Join(lines, "\n")
	}
	vm, network := help(connectors.KindVM), help(connectors.KindNetwork)
	for _, text := range []string{vm, network} {
		for _, other := range []string{"AWS", "$", "NAT", "EC2", "VPC"} {
			if strings.Contains(text, other) {
				t.Errorf("OpenStack help names %q:\n%s", other, text)
			}
		}
	}
	for _, want := range []string{"startup script on a deployed server replaces the server"} {
		if !strings.Contains(vm, want) {
			t.Errorf("VM help lacks %q:\n%s", want, vm)
		}
	}
	for _, want := range []string{"public server's first network needs it on", "address range of a deployed network replaces its servers"} {
		if !strings.Contains(network, want) {
			t.Errorf("network help lacks %q:\n%s", want, network)
		}
	}
}

// TestContract runs B's shared connector contract against this connector,
// with a fake Keystone for verification and a fake terraform behind the real
// shared runner, so it needs no cloud and no network.
func TestContract(t *testing.T) {
	k := newKeystone(t)
	contract.Run(t, contract.Fixture{
		New:      func() connectors.Connector { return newTestConnector(t, &fakeTerraform{}) },
		Region:   testRegion,
		Good:     k.secrets(goodSecret),
		Bad:      k.secrets(badSecret),
		Graph:    sampleGraph(),
		Settings: sampleSettings(),
		Cross: &contract.CrossFixture{
			Graph: connectors.Graph{
				Blocks: map[string]connectors.Block{
					"src":  {ID: "src", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"size": "small"}},
					"net1": {ID: "net1", Kind: connectors.KindNetwork, Name: "net1", Fields: map[string]any{"cidr": "10.20.0.0/24"}},
					"dst":  {ID: "dst", Kind: connectors.KindVM, Name: "db", Fields: map[string]any{"size": "small", "public": true}},
					"net2": {ID: "net2", Kind: connectors.KindNetwork, Name: "net2", Fields: map[string]any{"cidr": "10.30.0.0/24"}},
				},
				Edges: []connectors.Edge{{Source: "src", Target: "net1"}, {Source: "dst", Target: "net2"}, {Source: "src", Target: "dst"}},
			},
			Placed:   []string{"dst", "net2"},
			SourceVM: "src",
			TargetVM: "dst",
		},
	})
}
