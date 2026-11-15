package generalopenstack

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
)

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
