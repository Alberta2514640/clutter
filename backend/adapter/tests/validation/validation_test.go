// Package validation submits the approved set of invalid inputs to System
// B's capability check and proves each is refused before anything runs: no
// run starts, and no preview renders Terraform from it.
//
// The set (ids V01 to V32) covers missing images and flavors, invalid
// dependencies, AWS-only resources on clouds without them, unknown
// providers, malformed network data, and names that try to break out of
// generated code. Each id is one subtest, so the report can list them.
package validation

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/translate"
	"github.com/Alberta2514640/clutter/backend/adapter/tests/testkit"
	"github.com/google/uuid"
)

func TestMain(m *testing.M) { testkit.Main(m) }

type target string

const (
	openstack target = "general-openstack"
	awsCloud  target = "aws"
	mockCloud target = "mock"
)

// invalid is one approved invalid input.
type invalid struct {
	id, group, what string
	on              target
	// doc is a diagram document; when set, graph is ignored and the
	// translator may refuse it first (translatorRefuses).
	doc               string
	translatorRefuses bool
	graph             func() connectors.Graph
	settings          map[string]any
	// node and code, when set, name the problem the check must report.
	node, code string
}

func b(id string, kind connectors.BlockKind, name string, fields map[string]any) connectors.Block {
	if fields == nil {
		fields = map[string]any{}
	}
	return connectors.Block{ID: id, Kind: kind, Name: name, Fields: fields}
}

func g(edges []connectors.Edge, bs ...connectors.Block) func() connectors.Graph {
	return func() connectors.Graph {
		out := connectors.Graph{Blocks: map[string]connectors.Block{}, Edges: edges}
		for _, x := range bs {
			out.Blocks[x.ID] = x
		}
		return out
	}
}

func e(s, t string) connectors.Edge { return connectors.Edge{Source: s, Target: t} }

// vmOnNet is a VM on a network whose fields are net.
func vmOnNet(net map[string]any) func() connectors.Graph {
	return g([]connectors.Edge{e("vm", "net")}, b("vm", connectors.KindVM, "web", nil), b("net", connectors.KindNetwork, "net", net))
}

func withFirewall(fw map[string]any) func() connectors.Graph {
	return g([]connectors.Edge{e("vm", "net"), e("fw", "vm")},
		b("vm", connectors.KindVM, "web", nil), b("net", connectors.KindNetwork, "net", nil), b("fw", connectors.KindFirewall, "fw", fw))
}

func withBlock(kind connectors.BlockKind) func() connectors.Graph {
	return g([]connectors.Edge{e("vm", "net")}, b("vm", connectors.KindVM, "web", nil), b("net", connectors.KindNetwork, "net", nil), b("x", kind, "extra", nil))
}

// approvedSet is the approved set of invalid inputs.
var approvedSet = []invalid{
	// Missing images or flavors.
	{id: "V01", group: "missing image or flavor", what: "empty image", on: openstack, graph: vmOnNet(nil), settings: map[string]any{"image": ""}, code: connectors.CodeMissingValue},
	{id: "V02", group: "missing image or flavor", what: "image of blanks", on: openstack, graph: vmOnNet(nil), settings: map[string]any{"image": "   "}, code: connectors.CodeMissingValue},
	{id: "V03", group: "missing image or flavor", what: "empty small flavor", on: openstack, graph: vmOnNet(nil), settings: map[string]any{"flavor_small": ""}, code: connectors.CodeMissingValue},
	{id: "V04", group: "missing image or flavor", what: "flavor of the wrong type", on: openstack, graph: vmOnNet(nil), settings: map[string]any{"flavor_medium": 5.0}, code: connectors.CodeInvalidValue},
	{id: "V05", group: "missing image or flavor", what: "image name that breaks out of HCL", on: openstack, graph: vmOnNet(nil), settings: map[string]any{"image": `cirros" } resource "x" "y" {`}, code: connectors.CodeInvalidValue},
	{id: "V06", group: "missing image or flavor", what: "AWS instance type outside the list", on: awsCloud, graph: vmOnNet(nil), settings: map[string]any{"instance_type": "x99.mega"}, code: connectors.CodeInvalidValue},

	// Invalid dependencies.
	{id: "V07", group: "invalid dependency", what: "edge to a block that does not exist", on: openstack, translatorRefuses: true,
		doc: `{"engine":"blocks","nodes":[{"id":"vm","type":"vm","data":{"name":"web"}}],"edges":[{"source":"vm","target":"ghost"}]}`},
	{id: "V08", group: "invalid dependency", what: "block joined to itself", on: openstack, translatorRefuses: true,
		doc: `{"engine":"blocks","nodes":[{"id":"vm","type":"vm","data":{"name":"web"}}],"edges":[{"source":"vm","target":"vm"}]}`},
	{id: "V09", group: "invalid dependency", what: "network pointing at a VM (reversed edge)", on: openstack,
		graph: g([]connectors.Edge{e("net", "vm")}, b("vm", connectors.KindVM, "web", nil), b("net", connectors.KindNetwork, "net", nil)), code: capability.CodeUnsupportedEdge},
	{id: "V10", group: "invalid dependency", what: "one volume on two VMs", on: openstack,
		graph: g([]connectors.Edge{e("a", "net"), e("c", "net"), e("v", "a"), e("v", "c")},
			b("a", connectors.KindVM, "web", nil), b("c", connectors.KindVM, "api", nil), b("net", connectors.KindNetwork, "net", nil), b("v", connectors.KindVolume, "data", nil)),
		node: "v", code: connectors.CodeVolumeTwoVMs},
	{id: "V11", group: "invalid dependency", what: "VM with no network on OpenStack", on: openstack, graph: g(nil, b("vm", connectors.KindVM, "web", nil)), node: "vm", code: "vm_without_network"},
	{id: "V12", group: "invalid dependency", what: "two VMs that need each other's address first", on: openstack,
		graph: g([]connectors.Edge{e("a", "net"), e("c", "net"), e("a", "c"), e("c", "a")},
			b("a", connectors.KindVM, "web", nil), b("c", connectors.KindVM, "db", nil), b("net", connectors.KindNetwork, "net", nil)),
		code: connectors.CodeAddressCycle},
	{id: "V13", group: "invalid dependency", what: "AWS volume with no VM", on: awsCloud, graph: g(nil, b("v", connectors.KindVolume, "data", nil)), node: "v", code: "volume_needs_vm"},
	{id: "V14", group: "invalid dependency", what: "AWS HTTP API with no function behind it", on: awsCloud, graph: g(nil, b("api", connectors.KindHTTPAPI, "api", nil)), node: "api", code: "http_api_without_function"},
	{id: "V15", group: "invalid dependency", what: "two blocks with the same name", on: openstack,
		graph: g([]connectors.Edge{e("a", "net"), e("c", "net")}, b("a", connectors.KindVM, "web", nil), b("c", connectors.KindVM, "web", nil), b("net", connectors.KindNetwork, "net", nil)),
		code:  connectors.CodeDuplicateName},

	// Unsupported AWS resources on a cloud without them.
	{id: "V16", group: "unsupported AWS resource", what: "bucket (S3) on OpenStack", on: openstack, graph: withBlock(connectors.KindBucket), node: "x", code: capability.CodeUnsupportedBlock},
	{id: "V17", group: "unsupported AWS resource", what: "function (Lambda) on OpenStack", on: openstack, graph: withBlock(connectors.KindFunction), node: "x", code: capability.CodeUnsupportedBlock},
	{id: "V18", group: "unsupported AWS resource", what: "table (DynamoDB) on OpenStack", on: openstack, graph: withBlock(connectors.KindTable), node: "x", code: capability.CodeUnsupportedBlock},
	{id: "V19", group: "unsupported AWS resource", what: "HTTP API (API Gateway) on OpenStack", on: openstack, graph: withBlock(connectors.KindHTTPAPI), node: "x", code: capability.CodeUnsupportedBlock},
	{id: "V20", group: "unsupported AWS resource", what: "bucket on the mock cloud", on: mockCloud, graph: withBlock(connectors.KindBucket), node: "x", code: capability.CodeUnsupportedBlock},
	{id: "V21", group: "unsupported AWS resource", what: "A's raw AWS node type (ec2) as a block kind", on: awsCloud, graph: withBlock("ec2"), node: "x", code: capability.CodeUnsupportedBlock},
	{id: "V22", group: "unsupported AWS resource", what: "A's provider-prefixed type (aws.s3) in a document", on: openstack, translatorRefuses: true,
		doc: `{"engine":"blocks","nodes":[{"id":"x","type":"aws.s3","data":{"name":"files"}}]}`},

	// Malformed network data.
	{id: "V23", group: "malformed network data", what: "prefix /33", on: openstack, graph: vmOnNet(map[string]any{"cidr": "10.0.0.0/33"}), node: "net", code: connectors.CodeInvalidValue},
	{id: "V24", group: "malformed network data", what: "not an address range", on: openstack, graph: vmOnNet(map[string]any{"cidr": "not-a-cidr"}), node: "net", code: connectors.CodeInvalidValue},
	{id: "V25", group: "malformed network data", what: "octet over 255", on: openstack, graph: vmOnNet(map[string]any{"cidr": "300.1.1.1/24"}), node: "net", code: connectors.CodeInvalidValue},
	{id: "V26", group: "malformed network data", what: "IPv6 range on OpenStack", on: openstack, graph: vmOnNet(map[string]any{"cidr": "fd00::/64"}), node: "net", code: connectors.CodeInvalidValue},
	{id: "V27", group: "malformed network data", what: "range too small (/30)", on: openstack, graph: vmOnNet(map[string]any{"cidr": "10.0.0.0/30"}), node: "net", code: connectors.CodeInvalidValue},
	{id: "V28", group: "malformed network data", what: "range of the wrong type (a number)", on: openstack, graph: vmOnNet(map[string]any{"cidr": 42.0}), node: "net", code: connectors.CodeInvalidValue},
	{id: "V29", group: "malformed network data", what: "range that breaks out of HCL", on: awsCloud, graph: vmOnNet(map[string]any{"cidr": `10.0.0.0/24" } resource "aws_iam_user" "x" {`}), node: "net", code: connectors.CodeInvalidValue},
	{id: "V30", group: "malformed network data", what: "firewall port over 65535", on: openstack, graph: withFirewall(map[string]any{"ingress": "tcp/99999"}), node: "fw", code: connectors.CodeInvalidValue},
	{id: "V31", group: "malformed network data", what: "firewall protocol that does not exist", on: awsCloud, graph: withFirewall(map[string]any{"ingress": "http/80"}), node: "fw", code: connectors.CodeInvalidValue},
	{id: "V32", group: "malformed network data", what: "firewall source that is not a range", on: openstack, graph: withFirewall(map[string]any{"ingress": "tcp/22", "source_cidr": "everywhere"}), node: "fw", code: connectors.CodeInvalidValue},
}

type cloud struct {
	c        connectors.Connector
	region   string
	settings map[string]any
}

func clouds(t *testing.T) map[target]cloud {
	fake := &testkit.FakeTerraform{}
	return map[target]cloud{
		openstack: {testkit.NewOpenStack(t, fake), testkit.OpenStackRegion, map[string]any{"image": "cirros", "flavor_small": "m1.tiny"}},
		awsCloud:  {testkit.NewAWS(t, fake), testkit.AWSRegion, map[string]any{}},
		mockCloud: {testkit.NewMock(), "mock-1", map[string]any{}},
	}
}

func deployment(cl cloud, settings map[string]any) connectors.Deployment {
	id := uuid.NewString()
	s := map[string]any{}
	for k, v := range cl.settings {
		s[k] = v
	}
	for k, v := range settings {
		s[k] = v
	}
	return connectors.Deployment{ID: id, Prefix: connectors.PrefixFor(id),
		Connection: connectors.Connection{ID: uuid.NewString(), ConnectorID: cl.c.ID(), Region: cl.region}, Settings: s}
}

// TestValidBaselinePasses is the control: the same shapes, valid, pass on
// every cloud, so a refusal below is about the bad input and nothing else.
func TestValidBaselinePasses(t *testing.T) {
	for name, cl := range clouds(t) {
		t.Run(string(name), func(t *testing.T) {
			r := capability.CheckConnector(cl.c, withFirewall(map[string]any{"ingress": "tcp/22", "source_cidr": "203.0.113.0/24"})(), deployment(cl, nil))
			if !r.Ready {
				t.Fatalf("a valid VM, network and firewall was refused: %+v", r.Problems)
			}
		})
	}
}

func TestApprovedInvalidInputsAreRefused(t *testing.T) {
	cs := clouds(t)
	for _, in := range approvedSet {
		t.Run(in.id+" "+in.group+": "+in.what, func(t *testing.T) {
			cl := cs[in.on]
			var gr connectors.Graph
			if in.doc != "" {
				var err error
				gr, err = translate.Translate([]byte(in.doc))
				if in.translatorRefuses {
					var te *translate.Error
					if !errors.As(err, &te) {
						t.Fatalf("the translator accepted it (err %v)", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
			} else {
				gr = in.graph()
			}
			d := deployment(cl, in.settings)
			r := capability.CheckConnector(cl.c, gr, d)
			if r.Ready {
				t.Fatalf("the capability check said ready: %+v", r)
			}
			if in.code != "" && !contract.HasProblem(r.Problems, in.node, in.code) {
				t.Fatalf("want %s on %q, got %+v", in.code, in.node, r.Problems)
			}
			for _, p := range r.Problems {
				if p.Message == "" {
					t.Fatalf("a problem with no message: %+v", p)
				}
			}
			// Nothing reaches generated code. A connector's Preview promises to
			// refuse what its own Validate refuses; what only the core's check
			// refuses is stopped by B API before Preview is called
			// (TestPreviewRouteRefusesWhatTheCheckRefuses).
			if pv, ok := cl.c.(connectors.Previewer); ok && len(cl.c.Validate(gr, d)) > 0 {
				files, err := pv.Preview(context.Background(), gr, d)
				if err == nil {
					t.Fatalf("preview rendered %d files from a graph its own Validate refuses", len(files))
				}
				var ue *connectors.UserError
				if !errors.As(err, &ue) {
					t.Fatalf("preview refused with a raw error: %v", err)
				}
			}
		})
	}
}

// TestUnknownProvidersAreRefused covers the unknown-provider part of the set
// at the registry and, with Docker, at B API.
func TestUnknownProvidersAreRefused(t *testing.T) {
	unknown := []string{"patrii", "azure", "AWS", "../aws", "general-openstack ", ""}
	reg := connectors.NewRegistry()
	for _, c := range clouds(t) {
		if err := reg.Register(c.c); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range unknown {
		if _, ok := reg.Get(id); ok {
			t.Fatalf("the registry found %q", id)
		}
	}

	env := testkit.NewEnv(t, testkit.Options{}, testkit.NewMock())
	tn := env.NewTenant(t, testkit.MockDoc)
	for _, id := range unknown {
		code, body := env.Call(t, &tn, "POST", "/v1/organizations/"+tn.Org+"/connections",
			map[string]any{"connector": id, "name": "x", "region": "r", "credentials": map[string]string{"token": "t"}})
		if code != http.StatusUnprocessableEntity && code != http.StatusBadRequest {
			t.Fatalf("connector %q: %d %s", id, code, body)
		}
	}
	code, body := env.Call(t, &tn, "GET", "/v1/organizations/"+tn.Org+"/connectors/patrii/linked-accounts", nil)
	if code != http.StatusNotFound {
		t.Fatalf("linked accounts of an unknown connector: %d %s", code, body)
	}
}

// TestCompatibilityRouteRefusesBeforeARun proves the same check guards B
// API: a deployment whose diagram holds an unsupported block is not ready,
// and a run of it fails before the cloud with the reason named.
func TestCompatibilityRouteRefusesBeforeARun(t *testing.T) {
	env := testkit.NewEnv(t, testkit.Options{}, testkit.NewMock())
	doc := strings.Replace(testkit.MockDoc, `{"id":"n4","type":"firewall"`, `{"id":"n5","type":"bucket","data":{"name":"files"}},{"id":"n4","type":"firewall"`, 1)
	tn := env.NewTenant(t, doc)
	conn := env.Connect(t, &tn, "mock", "mock-1", map[string]string{"token": "mock-valid-token"})
	dep := env.Deploy(t, &tn, conn, map[string]any{"image": "debian-12"}, nil)
	code, body := env.Call(t, &tn, "GET", "/v1/organizations/"+tn.Org+"/deployments/"+dep+"/compatibility", nil)
	if code != http.StatusOK || !strings.Contains(body, `"ready":false`) || !strings.Contains(body, capability.CodeUnsupportedBlock) {
		t.Fatalf("compatibility: %d %s", code, body)
	}
	code, body = env.Call(t, &tn, "POST", "/v1/organizations/"+tn.Org+"/deployments/"+dep+"/runs", map[string]string{"action": "apply"})
	if code == http.StatusAccepted {
		seen := env.WaitRun(t, &tn, testkit.Field(t, body, "id"), 20*time.Second)
		if seen[len(seen)-1] != "failed" {
			t.Fatalf("a run of an unready deployment ended %v", seen)
		}
	} else if code != http.StatusUnprocessableEntity && code != http.StatusConflict {
		t.Fatalf("run of an unready deployment: %d %s", code, body)
	}
}

// TestPreviewRouteRefusesWhatTheCheckRefuses proves B API runs the capability
// check before a connector's Preview: an OpenStack deployment of a diagram
// with a bucket is answered 422 not_ready, with no files.
func TestPreviewRouteRefusesWhatTheCheckRefuses(t *testing.T) {
	k := testkit.NewKeystone(t)
	env := testkit.NewEnv(t, testkit.Options{}, testkit.NewOpenStack(t, &testkit.FakeTerraform{}))
	doc := `{"engine":"blocks","nodes":[
	 {"id":"vm","type":"vm","data":{"name":"web","fields":{"size":"small"}}},
	 {"id":"net","type":"network","data":{"name":"net","fields":{"cidr":"10.20.0.0/24"}}},
	 {"id":"bkt","type":"bucket","data":{"name":"uploads"}}],
	 "edges":[{"source":"vm","target":"net"}]}`
	tn := env.NewTenant(t, doc)
	conn := env.Connect(t, &tn, "general-openstack", testkit.OpenStackRegion, k.Secrets(testkit.OpenStackCredID, testkit.OpenStackSecret))
	dep := env.Deploy(t, &tn, conn, map[string]any{"image": "cirros"}, nil)
	code, body := env.Call(t, &tn, "GET", "/v1/organizations/"+tn.Org+"/deployments/"+dep+"/preview", nil)
	if code != http.StatusUnprocessableEntity || !strings.Contains(body, "not_ready") || strings.Contains(body, `"files"`) {
		t.Fatalf("preview of an unready deployment: %d %s", code, body)
	}
	// Leaving the bucket out makes the same deployment previewable.
	code, body = env.Call(t, &tn, "PATCH", "/v1/organizations/"+tn.Org+"/deployments/"+dep, map[string]any{"exclusions": []string{"bkt"}})
	if code != http.StatusOK {
		t.Fatalf("leave the bucket out: %d %s", code, body)
	}
	code, body = env.Call(t, &tn, "GET", "/v1/organizations/"+tn.Org+"/deployments/"+dep+"/preview", nil)
	if code != http.StatusOK || !strings.Contains(body, `"files"`) {
		t.Fatalf("preview after leaving the bucket out: %d %.300s", code, body)
	}
}
