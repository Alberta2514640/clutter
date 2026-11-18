// Package contract runs the one shared connector contract suite
// (connectors/contract) against every connector, side by side: General's
// OpenStack connector, the AWS connector and the mock. Each is driven through
// B's real job runner; their clouds are fakes (a fake Keystone, a fake STS and
// a stand-in for the terraform binary), so the suite needs no account.
//
// Patrii Cloud has no connector yet: it has no public API or Terraform
// provider (docs/research/patricloud). Its row skips with that reason, so the
// report shows the gap instead of hiding it.
package contract

import (
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/Alberta2514640/clutter/backend/adapter/tests/testkit"
)

func vmPair(fields map[string]any, extra ...connectors.Block) connectors.Graph {
	src := map[string]any{}
	dst := map[string]any{"public": true}
	for k, v := range fields {
		src[k], dst[k] = v, v
	}
	g := connectors.Graph{Blocks: map[string]connectors.Block{
		"src": {ID: "src", Kind: connectors.KindVM, Name: "web", Fields: src},
		"dst": {ID: "dst", Kind: connectors.KindVM, Name: "db", Fields: dst},
	}, Edges: []connectors.Edge{{Source: "src", Target: "dst"}}}
	for _, b := range extra {
		g.Blocks[b.ID] = b
	}
	return g
}

func TestContractMock(t *testing.T) {
	contract.Run(t, contract.Fixture{
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
		Cross:    &contract.CrossFixture{Graph: vmPair(nil), Placed: []string{"dst"}, SourceVM: "src", TargetVM: "dst"},
	})
}

func TestContractGeneralOpenStack(t *testing.T) {
	k := testkit.NewKeystone(t)
	fake := &testkit.FakeTerraform{}
	cross := connectors.Graph{
		Blocks: map[string]connectors.Block{
			"src":  {ID: "src", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"size": "small"}},
			"net1": {ID: "net1", Kind: connectors.KindNetwork, Name: "net1", Fields: map[string]any{"cidr": "10.20.0.0/24"}},
			"dst":  {ID: "dst", Kind: connectors.KindVM, Name: "db", Fields: map[string]any{"size": "small", "public": true}},
			"net2": {ID: "net2", Kind: connectors.KindNetwork, Name: "net2", Fields: map[string]any{"cidr": "10.30.0.0/24"}},
		},
		Edges: []connectors.Edge{{Source: "src", Target: "net1"}, {Source: "dst", Target: "net2"}, {Source: "src", Target: "dst"}},
	}
	contract.Run(t, contract.Fixture{
		New:    func() connectors.Connector { return testkit.NewOpenStack(t, fake) },
		Region: testkit.OpenStackRegion,
		Good:   k.Secrets(testkit.OpenStackCredID, testkit.OpenStackSecret),
		Bad:    k.Secrets(testkit.OpenStackCredID, testkit.OpenStackBadValue),
		Graph: connectors.Graph{
			Blocks: map[string]connectors.Block{
				"n1": {ID: "n1", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"size": "small", "public": true}},
				"n2": {ID: "n2", Kind: connectors.KindNetwork, Name: "net", Fields: map[string]any{"cidr": "10.20.0.0/24"}},
				"n3": {ID: "n3", Kind: connectors.KindFirewall, Name: "fw", Fields: map[string]any{"ingress": "tcp/22, tcp/80", "source_cidr": "0.0.0.0/0"}},
				"n4": {ID: "n4", Kind: connectors.KindVolume, Name: "data", Fields: map[string]any{"size_gb": float64(1)}},
			},
			Edges: []connectors.Edge{{Source: "n1", Target: "n2"}, {Source: "n3", Target: "n1"}, {Source: "n4", Target: "n1"}},
		},
		Settings: map[string]any{"image": "cirros", "flavor_small": "m1.tiny"},
		Timeout:  2 * time.Minute,
		Cross:    &contract.CrossFixture{Graph: cross, Placed: []string{"dst", "net2"}, SourceVM: "src", TargetVM: "dst"},
	})
}

func TestContractAWS(t *testing.T) {
	fake := &testkit.FakeTerraform{}
	blocks := []connectors.Block{
		{ID: "net1", Kind: connectors.KindNetwork, Name: "main-net", Fields: map[string]any{"cidr": "10.1.0.0/24"}},
		{ID: "web", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"size": "medium", "public": true}},
		{ID: "fw", Kind: connectors.KindFirewall, Name: "web-fw", Fields: map[string]any{"ingress": "tcp/22, tcp/443", "source_cidr": "203.0.113.0/24"}},
		{ID: "vol", Kind: connectors.KindVolume, Name: "data", Fields: map[string]any{"size_gb": float64(20)}},
		{ID: "bucket", Kind: connectors.KindBucket, Name: "uploads", Fields: map[string]any{"versioning": true}},
		{ID: "fn", Kind: connectors.KindFunction, Name: "api_handler", Fields: map[string]any{"memory_mb": float64(256)}},
		{ID: "table", Kind: connectors.KindTable, Name: "orders", Fields: map[string]any{"key": "order_id"}},
		{ID: "api", Kind: connectors.KindHTTPAPI, Name: "public-api", Fields: map[string]any{"cors": true}},
	}
	g := connectors.Graph{Blocks: map[string]connectors.Block{}, Edges: []connectors.Edge{
		{Source: "web", Target: "net1"}, {Source: "fw", Target: "web"}, {Source: "vol", Target: "web"},
		{Source: "fn", Target: "bucket"}, {Source: "fn", Target: "table"}, {Source: "api", Target: "fn"},
	}}
	for _, b := range blocks {
		g.Blocks[b.ID] = b
	}
	contract.Run(t, contract.Fixture{
		New:      func() connectors.Connector { return testkit.NewAWS(t, fake) },
		Region:   testkit.AWSRegion,
		Good:     testkit.AWSSecrets(true),
		Bad:      testkit.AWSSecrets(false),
		Graph:    g,
		Settings: map[string]any{"function_runtime": "python3.12", "instance_type": "t3.large"},
		Timeout:  2 * time.Minute,
		Cross:    &contract.CrossFixture{Graph: vmPair(nil), Placed: []string{"dst"}, SourceVM: "src", TargetVM: "dst"},
	})
}

func TestContractPatriiCloud(t *testing.T) {
	t.Skip("not implementable yet: Patrii Cloud publishes no customer API or Terraform provider, so there is no Patrii connector to run the contract against (docs/research/patricloud/2026-10-26-patricloud-research.md). Through Path A it would be a connection of general-openstack, which this suite already covers.")
}
