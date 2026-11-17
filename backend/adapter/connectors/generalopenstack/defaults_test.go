package generalopenstack

import (
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/google/uuid"
)

// testSSHKey is a throwaway public key; its private half was never kept.
const testSSHKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAaLzQkB8bkh7I4/uWuVxHsQDSjL4RIq6y+JnLJP5Nn"

// twoTierGraph is a public web VM and a private db VM, each on its own
// network, the web VM getting the db VM's address.
func twoTierGraph() connectors.Graph {
	return connectors.Graph{
		Blocks: map[string]connectors.Block{
			"web":    {ID: "web", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"size": "small", "public": true}},
			"db":     {ID: "db", Kind: connectors.KindVM, Name: "db", Fields: map[string]any{"size": "small"}},
			"front":  {ID: "front", Kind: connectors.KindNetwork, Name: "front", Fields: map[string]any{"cidr": "10.1.0.0/24"}},
			"back":   {ID: "back", Kind: connectors.KindNetwork, Name: "back", Fields: map[string]any{"cidr": "10.2.0.0/24"}},
			"db-fw":  {ID: "db-fw", Kind: connectors.KindFirewall, Name: "db-fw", Fields: map[string]any{"ingress": "tcp/5432", "source_cidr": "10.0.0.0/8"}},
			"web-fw": {ID: "web-fw", Kind: connectors.KindFirewall, Name: "web-fw", Fields: map[string]any{"ingress": "tcp/443"}},
		},
		Edges: []connectors.Edge{
			{Source: "web", Target: "front"},
			{Source: "db", Target: "back"},
			{Source: "db-fw", Target: "db"},
			{Source: "web-fw", Target: "web"},
			{Source: "web", Target: "db"},
		},
	}
}

func renderMainOf(t *testing.T, g connectors.Graph) string {
	t.Helper()
	c := newConnector(nil)
	d := testDeployment(uuid.NewString())
	if ps := c.Validate(g, d); len(ps) > 0 {
		t.Fatalf("problems: %+v", ps)
	}
	files, err := c.Render(g, d)
	if err != nil {
		t.Fatal(err)
	}
	return string(files["main.tf"])
}

// TestOneRouterPerDeployment: every network with internet access joins one
// router; a network with it off joins none.
func TestOneRouterPerDeployment(t *testing.T) {
	main := renderMainOf(t, twoTierGraph())
	if n := strings.Count(main, `resource "openstack_networking_router_v2"`); n != 1 {
		t.Fatalf("%d routers, want 1", n)
	}
	if n := strings.Count(main, "router_id = openstack_networking_router_v2.deployment.id"); n != 2 {
		t.Fatalf("%d networks on the router, want 2", n)
	}

	g := twoTierGraph()
	g.Blocks["back"].Fields["private_internet"] = false
	g.Edges = g.Edges[:4] // the web VM no longer gets the db VM's address
	main = renderMainOf(t, g)
	if strings.Contains(main, `resource "openstack_networking_router_interface_v2" "b_back"`) {
		t.Error("a network with internet access off joined the router")
	}
}

// TestWhatOpenStackCannotReach: a public VM needs its first network on the
// router, and two VMs on different networks reach each other only when both
// are on it.
func TestWhatOpenStackCannotReach(t *testing.T) {
	c := newConnector(nil)
	d := testDeployment(uuid.NewString())

	g := twoTierGraph()
	g.Blocks["back"].Fields["private_internet"] = false
	ps := c.Validate(g, d)
	found := false
	for _, p := range ps {
		found = found || (p.Code == connectors.CodePeerUnreachable && p.EdgeSource == "web" && p.EdgeTarget == "db")
	}
	if !found {
		t.Errorf("VMs on networks that cannot reach each other were accepted: %+v", ps)
	}

	g = twoTierGraph()
	g.Blocks["front"].Fields["private_internet"] = false
	if !contract.HasProblem(c.Validate(g, d), "web", codePublicUnrouted) {
		t.Error("a public VM on a network with no internet access was accepted")
	}
}

// TestPeersAndTags: the VM that gets another's address may reach it through
// a peer group on the target; every resource that takes tags or metadata
// carries ManagedBy and the deployment; a VM's SSH key becomes a key pair.
func TestPeersAndTags(t *testing.T) {
	g := twoTierGraph()
	g.Blocks["web"].Fields[connectors.FieldSSHKey] = testSSHKey + " me@laptop"
	main := renderMainOf(t, g)
	for _, want := range []string{
		`resource "openstack_networking_secgroup_v2" "p_db"`,
		`remote_ip_prefix  = "${openstack_compute_instance_v2.b_web.network[0].fixed_ip_v4}/32"`,
		"security_groups = [openstack_networking_secgroup_v2.b_db-fw.name, openstack_networking_secgroup_v2.p_db.name]",
		`resource "openstack_compute_keypair_v2" "b_web"`,
		`public_key = sensitive("` + testSSHKey + `")`,
		"key_pair        = openstack_compute_keypair_v2.b_web.name",
	} {
		if !strings.Contains(main, want) {
			t.Errorf("main.tf lacks %q", want)
		}
	}
	if strings.Contains(main, "laptop") {
		t.Error("the key's comment reached the Terraform")
	}
	for _, typ := range []string{tNetwork, tSubnet, tRouter, tSecgroup, tFloatingIP} {
		for _, block := range strings.Split(main, `resource "`+typ+`"`)[1:] {
			if !strings.Contains(strings.SplitN(block, "\n}", 2)[0], `"ManagedBy=Clutter", "ClutterDeployment=`) {
				t.Errorf("a %s has no tags", typ)
			}
		}
	}
	if strings.Count(main, `metadata        = { ManagedBy = "Clutter", ClutterDeployment = "`) != 2 {
		t.Error("a server has no metadata")
	}
	g.Blocks["web"].Fields[connectors.FieldSSHKey] = "ssh-dss AAAA"
	if !contract.HasProblem(newConnector(nil).Validate(g, testDeployment(uuid.NewString())), "web", connectors.CodeInvalidValue) {
		t.Error("a bad key was accepted")
	}
}
