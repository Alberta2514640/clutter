package mock_test

import (
	"context"
	"strings"
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

// A second apply of the same diagram changes nothing, and says so per block;
// it reports no block as built or changed.
func TestASecondApplySaysUnchanged(t *testing.T) {
	m, f := mock.New(), Fixture()
	d := connectors.Deployment{ID: "d1", Prefix: "p1", Connection: connectors.Connection{ConnectorID: mock.ID, Region: f.Region}}
	var first strings.Builder
	if _, err := m.Apply(context.Background(), connectors.Run{Log: &first}, f.Graph, d); err != nil {
		t.Fatal(err)
	}
	var again strings.Builder
	res, err := m.Apply(context.Background(), connectors.Run{Log: &again}, f.Graph, d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.String(), "Created web (VM).") || strings.Contains(first.String(), "p1") {
		t.Fatalf("first apply: %q", first.String())
	}
	if strings.Contains(again.String(), "Created") || !strings.Contains(again.String(), "Unchanged web (VM).") || len(res.Resources) != 0 {
		t.Fatalf("second apply: %q, resources %v", again.String(), res.Resources)
	}
	var plan strings.Builder
	if p, err := m.Plan(context.Background(), connectors.Run{Log: &plan}, f.Graph, d); err != nil || p != (connectors.PlanResult{}) || plan.Len() != 0 {
		t.Fatalf("plan of an unchanged diagram: %+v, %v, %q", p, err, plan.String())
	}
}

// A block named fail-me fails a plan and an apply on purpose, with a message
// for the user.
func TestABlockNamedFailMeFails(t *testing.T) {
	m, f := mock.New(), Fixture()
	f.Graph.Blocks["n3"] = connectors.Block{ID: "n3", Kind: connectors.KindVolume, Name: mock.FailName, Fields: map[string]any{}}
	d := connectors.Deployment{ID: "d1", Connection: connectors.Connection{ConnectorID: mock.ID, Region: f.Region}}
	run := connectors.Run{Log: &strings.Builder{}}
	_, perr := m.Plan(context.Background(), run, f.Graph, d)
	_, aerr := m.Apply(context.Background(), run, f.Graph, d)
	for _, err := range []error{perr, aerr} {
		if msg := connectors.UserMessage(err, ""); !strings.Contains(msg, "failed on purpose at fail-me (Volume)") {
			t.Fatalf("error %v", err)
		}
	}
	// What it built before the failing block stays, so a destroy removes it.
	if got := m.Built("d1"); len(got) != 2 {
		t.Fatalf("built %v, want the two blocks before fail-me", got)
	}
}
