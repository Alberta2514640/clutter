package split

import (
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/google/uuid"
)

// single is a connector without CrossCloud: it can never be one of several
// clouds.
type single struct{ *mock.Connector }

func (s single) ID() string { return "single" }

// The embedded mock's cross-cloud methods are hidden by these names, so
// single does not implement connectors.CrossCloud.
func (single) CrossCheck() {}

func graph(blocks map[string]connectors.BlockKind, edges ...[2]string) connectors.Graph {
	g := connectors.Graph{Blocks: map[string]connectors.Block{}}
	for id, k := range blocks {
		g.Blocks[id] = connectors.Block{ID: id, Kind: k, Name: "b-" + id, Fields: map[string]any{}}
	}
	for _, e := range edges {
		g.Edges = append(g.Edges, connectors.Edge{Source: e[0], Target: e[1]})
	}
	return g
}

func clouds(c connectors.Connector) (primary, other Cloud) {
	return Cloud{ConnectionID: uuid.NewString(), Name: "aws-main", Connector: c, Region: "mock-1", Primary: true, SecretRef: "ref-1"},
		Cloud{ConnectionID: uuid.NewString(), Name: "lab", Connector: c, Region: "mock-2", SecretRef: "ref-2"}
}

func TestPartsAreOrderedTargetsFirst(t *testing.T) {
	dep := uuid.NewString()
	// web (primary) gets db's address; db sits on the other cloud, so that
	// cloud is applied first.
	g := graph(map[string]connectors.BlockKind{"web": connectors.KindVM, "net": connectors.KindNetwork, "db": connectors.KindVM, "net2": connectors.KindNetwork},
		[2]string{"web", "net"}, [2]string{"db", "net2"}, [2]string{"web", "db"})
	p, o := clouds(mock.New())
	o.Nodes = []string{"db", "net2"}
	plan := Build(g, dep, nil, []Cloud{p, o})
	if len(plan.Problems) > 0 {
		t.Fatalf("problems: %+v", plan.Problems)
	}
	if len(plan.Parts) != 2 || plan.Parts[0].ConnectionID != o.ConnectionID || plan.Parts[1].ConnectionID != p.ConnectionID {
		t.Fatalf("order: %+v", plan.Parts)
	}
	target, source := plan.Parts[0], plan.Parts[1]
	if source.Deployment.ID != dep {
		t.Fatal("the primary's state moved")
	}
	if target.Deployment.ID == dep || target.Deployment.ID != PartID(dep, o.ConnectionID, false) {
		t.Fatalf("the placed cloud's state id is %q", target.Deployment.ID)
	}
	if target.Deployment.Prefix != connectors.PrefixFor(dep) || source.Deployment.Prefix != connectors.PrefixFor(dep) {
		t.Fatal("the parts do not share the deployment's name prefix")
	}
	// Each part leaves out every block the other holds.
	if !source.Deployment.Excluded("db") || source.Deployment.Excluded("web") || !target.Deployment.Excluded("web") || target.Deployment.Excluded("db") {
		t.Fatalf("exclusions: %v / %v", source.Deployment.Exclusions, target.Deployment.Exclusions)
	}
	if len(plan.Edges) != 1 || plan.Edges[0].Class != connectors.EdgeAddress {
		t.Fatalf("edges %+v", plan.Edges)
	}
	if len(target.Deployment.Cross.Exports) != 1 || len(source.Deployment.Cross.Imports) != 1 ||
		len(target.Deployment.Cross.Imports) != 0 || len(source.Deployment.Cross.Exports) != 0 {
		t.Fatalf("cross: %+v / %+v", target.Deployment.Cross, source.Deployment.Cross)
	}
	if len(plan.Notes) != 1 || plan.Notes[0].Code != CodeCrossAddress {
		t.Fatalf("notes %+v", plan.Notes)
	}
	if plan.Owner["db"] != 0 || plan.Owner["web"] != 1 {
		t.Fatalf("owner %v", plan.Owner)
	}
}

func TestAPermissionEdgeIsNotedWithItsKey(t *testing.T) {
	g := graph(map[string]connectors.BlockKind{"vm": connectors.KindVM, "files": connectors.KindBucket}, [2]string{"vm", "files"})
	p, o := clouds(mock.New())
	o.Nodes = []string{"files"}
	plan := Build(g, uuid.NewString(), nil, []Cloud{p, o})
	if len(plan.Notes) != 1 || plan.Notes[0].Code != CodeCrossKey ||
		plan.Notes[0].Message != "Clutter creates a key that can only read b-files and puts it on b-vm. The key is deleted when this deployment is destroyed." {
		t.Fatalf("notes %+v", plan.Notes)
	}
}

func TestEdgesThatCannotCrossAreRefused(t *testing.T) {
	for name, e := range map[string][2]string{
		"vm on a network": {"vm", "net"},
		"volume on a vm":  {"vol", "vm"},
		"function reads":  {"fn", "files"},
	} {
		g := graph(map[string]connectors.BlockKind{"vm": connectors.KindVM, "net": connectors.KindNetwork, "vol": connectors.KindVolume,
			"fn": connectors.KindFunction, "files": connectors.KindBucket}, e)
		p, o := clouds(mock.New())
		o.Nodes = []string{e[1]}
		plan := Build(g, uuid.NewString(), nil, []Cloud{p, o})
		if len(plan.Problems) != 1 || plan.Problems[0].Code != CodeCrossEdge || plan.Problems[0].EdgeSource != e[0] {
			t.Errorf("%s: %+v", name, plan.Problems)
		}
		// Left out of the deployment, the edge is no longer refused.
		plan = Build(g, uuid.NewString(), []string{e[0]}, []Cloud{p, o})
		if len(plan.Problems) != 0 {
			t.Errorf("%s excluded: %+v", name, plan.Problems)
		}
	}
}

func TestACycleBetweenCloudsIsRefused(t *testing.T) {
	g := graph(map[string]connectors.BlockKind{"a": connectors.KindVM, "b": connectors.KindVM}, [2]string{"a", "b"}, [2]string{"b", "a"})
	p, o := clouds(mock.New())
	o.Nodes = []string{"b"}
	plan := Build(g, uuid.NewString(), nil, []Cloud{p, o})
	if len(plan.Problems) != 1 || plan.Problems[0].Code != CodeCycle {
		t.Fatalf("problems %+v", plan.Problems)
	}
	if len(plan.Parts) != 2 {
		t.Fatal("a cycle lost a part")
	}
}

func TestACloudWithoutCrossCloudCannotSplit(t *testing.T) {
	g := graph(map[string]connectors.BlockKind{"a": connectors.KindVM, "b": connectors.KindVM})
	p, _ := clouds(mock.New())
	_, o := clouds(single{mock.New()})
	o.Nodes = []string{"b"}
	plan := Build(g, uuid.NewString(), nil, []Cloud{p, o})
	if len(plan.Problems) != 1 || plan.Problems[0].Code != CodeNoSplit || !strings.Contains(plan.Problems[0].Message, "lab") {
		t.Fatalf("problems %+v", plan.Problems)
	}
}

func TestAPlacedBlockNoLongerInTheDiagramIsANote(t *testing.T) {
	g := graph(map[string]connectors.BlockKind{"a": connectors.KindVM})
	p, o := clouds(mock.New())
	o.Nodes = []string{"gone"}
	plan := Build(g, uuid.NewString(), nil, []Cloud{p, o})
	if len(plan.Problems) != 0 || len(plan.Notes) != 1 || plan.Notes[0].NodeID != "gone" {
		t.Fatalf("problems %+v notes %+v", plan.Problems, plan.Notes)
	}
	if plan.Owner["a"] != 0 {
		t.Fatal("an unplaced block is not on the primary")
	}
}

func TestStepRunIDsDiffer(t *testing.T) {
	run, conn := uuid.NewString(), uuid.NewString()
	a, b := StepRunID(run, conn, "apply"), StepRunID(run, conn, "rollback")
	if a == b || a == run || StepRunID(run, conn, "apply") != a {
		t.Fatal("step run ids are not distinct and stable")
	}
	if _, err := uuid.Parse(a); err != nil {
		t.Fatal(err)
	}
}
