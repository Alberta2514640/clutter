package mock_test

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
)

// Fixture is the mock's contract fixture: a VM on a network with a volume and
// a firewall.
func Fixture() contract.Fixture {
	return contract.Fixture{
		New:    func() connectors.Connector { return mock.New() },
		Region: "mock-1",
		Good:   connectors.Secrets{"token": mock.ValidToken},
		Bad:    connectors.Secrets{"token": "not-the-token-0123"},
		Graph: connectors.Graph{
			Blocks: map[string]connectors.Block{
				"n1": {ID: "n1", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"size": "small"}},
				"n2": {ID: "n2", Kind: connectors.KindNetwork, Name: "net", Fields: map[string]any{}},
				"n3": {ID: "n3", Kind: connectors.KindVolume, Name: "data", Fields: map[string]any{"size_gb": float64(20)}},
				"n4": {ID: "n4", Kind: connectors.KindFirewall, Name: "fw", Fields: map[string]any{}},
			},
			Edges: []connectors.Edge{{Source: "n1", Target: "n2"}, {Source: "n3", Target: "n1"}, {Source: "n4", Target: "n1"}},
		},
		Settings: map[string]any{"image": "debian-12"},
		Cross: &contract.CrossFixture{
			Graph: connectors.Graph{
				Blocks: map[string]connectors.Block{
					"src": {ID: "src", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{}},
					"dst": {ID: "dst", Kind: connectors.KindVM, Name: "db", Fields: map[string]any{"public": true}},
				},
				Edges: []connectors.Edge{{Source: "src", Target: "dst"}},
			},
			Placed:   []string{"dst"},
			SourceVM: "src",
			TargetVM: "dst",
		},
	}
}

func TestContract(t *testing.T) {
	contract.Run(t, Fixture())
}
