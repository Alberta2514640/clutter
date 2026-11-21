package generalopenstack

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/google/uuid"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// joined lays the files out in name order, for a golden file.
func joined(files map[string][]byte) []byte {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var b bytes.Buffer
	for _, n := range names {
		b.WriteString("#### " + n + "\n")
		b.Write(files[n])
		b.WriteString("\n")
	}
	return b.Bytes()
}

func TestRenderSampleMatchesGolden(t *testing.T) {
	c := newConnector(nil)
	files, err := c.Render(sampleGraph(), testDeployment(fixedDeployment))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if err := terraform.CheckFiles(files, providers); err != nil {
		t.Fatalf("the runner refuses the rendered files: %v", err)
	}
	got := joined(files)
	golden := filepath.Join("testdata", "sample.golden")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run with -update to create it): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("rendered files differ from %s (run with -update after checking the change):\n%s", golden, got)
	}
}

func TestRenderNamesAddressesAndPin(t *testing.T) {
	d := testDeployment(fixedDeployment)
	files, err := newConnector(nil).Render(sampleGraph(), d)
	if err != nil {
		t.Fatal(err)
	}
	main := string(files["main.tf"])
	for _, want := range []string{
		`resource "openstack_compute_instance_v2" "b_n1"`,
		`name            = "` + d.Prefix + `-web"`,
		`flavor_name     = "m1.tiny"`,
		`security_groups = [openstack_networking_secgroup_v2.b_n3.name]`,
		`uuid = openstack_networking_network_v2.b_n2.id`,
		`external_network_id = data.openstack_networking_network_v2.external.id`,
		`resource "openstack_networking_floatingip_associate_v2" "b_n1"`,
		`network_id = openstack_compute_instance_v2.b_n1.network[0].uuid`,
		`resource "openstack_networking_secgroup_rule_v2" "b_n3_tcp_22_22"`,
		`resource "openstack_networking_secgroup_rule_v2" "b_n3_udp_1000_2000"`,
		`resource "openstack_networking_secgroup_rule_v2" "b_n3_icmp"`,
		`instance_id = openstack_compute_instance_v2.b_n1.id`,
		`name_regex  = "^cirros"`,
	} {
		if !strings.Contains(main, want) {
			t.Errorf("main.tf lacks %q", want)
		}
	}
	versions := string(files["versions.tf"])
	if !strings.Contains(versions, `version = "~> 3.4"`) || !strings.Contains(versions, `provider "openstack" {}`) {
		t.Errorf("versions.tf does not pin the provider with an empty block:\n%s", versions)
	}
	for name, body := range files {
		if strings.Contains(string(body), "OS_") || strings.Contains(string(body), "application_credential") {
			t.Errorf("%s mentions credentials", name)
		}
	}
	if !strings.Contains(string(files["outputs.tf"]), `output "id_n4"`) {
		t.Error("outputs.tf has no id output for the volume")
	}
}

// TestRenderSurvivesChangesToALiveDeployment pins what a DevStack run found
// when it changed a deployed diagram: an attached volume can grow, a renamed
// firewall or VM leaves the server's security groups as they are, and a new
// address range replaces the servers on that network with its subnet.
func TestRenderSurvivesChangesToALiveDeployment(t *testing.T) {
	c := newConnector(nil)
	d := testDeployment(fixedDeployment)
	render := func(g connectors.Graph) string {
		t.Helper()
		files, err := c.Render(g, d)
		if err != nil {
			t.Fatal(err)
		}
		return string(files["main.tf"])
	}
	main := render(sampleGraph())
	for _, want := range []string{
		"enable_online_resize = true",
		`name        = "` + d.Prefix + `.n3.firewall"`,
		`description = "` + d.Prefix + `-fw"`,
		"ignore_changes = [name]",
		"replace_triggered_by = [openstack_networking_subnet_v2.b_n2.id]",
	} {
		if !strings.Contains(main, want) {
			t.Errorf("main.tf lacks %q:\n%s", want, main)
		}
	}

	// Renaming blocks changes only names Neutron and Nova update in place.
	groupLines := func(main string) []string {
		var out []string
		for _, l := range strings.Split(main, "\n") {
			if strings.Contains(l, `.firewall"`) || strings.Contains(l, `.peers"`) || strings.Contains(l, "security_groups =") || strings.Contains(l, "replace_triggered_by =") {
				out = append(out, l)
			}
		}
		return out
	}
	rename := func(g connectors.Graph, ids ...string) connectors.Graph {
		for _, id := range ids {
			b := g.Blocks[id]
			b.Name = "renamed-" + b.Name
			g.Blocks[id] = b
		}
		return g
	}
	for name, pair := range map[string][2]connectors.Graph{
		"firewall, VM and network": {sampleGraph(), rename(sampleGraph(), "n1", "n2", "n3")},
		"VM with a peer group":     {twoTierGraph(), rename(twoTierGraph(), "db", "web", "db-fw")},
	} {
		before, after := groupLines(render(pair[0])), groupLines(render(pair[1]))
		if len(before) == 0 || !reflect.DeepEqual(before, after) {
			t.Errorf("%s: renaming changed a security group's name or a server's list:\n%q\n%q", name, before, after)
		}
	}
	if peers := render(twoTierGraph()); !strings.Contains(peers, `name        = "`+d.Prefix+`.db.peers"`) {
		t.Errorf("the peer group is not named by its VM's node id:\n%s", peers)
	}

	// A server on two networks is replaced when either subnet is.
	g := sampleGraph()
	g.Blocks["n5"] = connectors.Block{ID: "n5", Kind: connectors.KindNetwork, Name: "back", Fields: map[string]any{"cidr": "10.30.0.0/24"}}
	g.Edges = append(g.Edges, connectors.Edge{Source: "n1", Target: "n5"})
	if main := render(g); !strings.Contains(main, "replace_triggered_by = [openstack_networking_subnet_v2.b_n2.id, openstack_networking_subnet_v2.b_n5.id]") {
		t.Errorf("the server is not replaced with each of its subnets:\n%s", main)
	}
}

func TestRenderHonoursExclusions(t *testing.T) {
	g := sampleGraph()
	g.Blocks["n5"] = connectors.Block{ID: "n5", Kind: connectors.KindBucket, Name: "assets", Fields: map[string]any{}}
	g.Blocks["n6"] = connectors.Block{ID: "n6", Kind: connectors.KindFunction, Name: "thumbs", Fields: map[string]any{}}
	g.Edges = append(g.Edges, connectors.Edge{Source: "n6", Target: "n5"})
	c := newConnector(nil)
	d := testDeployment(fixedDeployment)

	// B's core refuses the removed blocks (the capability check), and the
	// connector does not report them again.
	if ps := capability.CheckConnector(c, g, d).Problems; !contract.HasProblem(ps, "n5", capability.CodeUnsupportedBlock) || !contract.HasProblem(ps, "n6", capability.CodeUnsupportedBlock) {
		t.Fatalf("removed blocks are not refused: %+v", ps)
	}
	if ps := c.Validate(g, d); contract.HasProblem(ps, "n5", capability.CodeUnsupportedBlock) {
		t.Fatalf("the connector repeats the capability check: %+v", ps)
	}

	d.Exclusions = []string{"n5", "n6", "n4"}
	if ps := capability.CheckConnector(c, g, d).Problems; len(ps) > 0 {
		t.Fatalf("excluded blocks still refused: %+v", ps)
	}
	files, err := c.Render(g, d)
	if err != nil {
		t.Fatal(err)
	}
	all := string(joined(files))
	for _, gone := range []string{"b_n4", "b_n5", "b_n6", "id_n4", "openstack_compute_volume_attach_v2"} {
		if strings.Contains(all, gone) {
			t.Errorf("excluded block left %q in the files", gone)
		}
	}
	if !strings.Contains(all, `"b_n1"`) {
		t.Error("an included block is missing")
	}

	// Leaving out the VM's only network leaves the VM with none.
	d.Exclusions = []string{"n5", "n6", "n2"}
	if ps := c.Validate(g, d); !contract.HasProblem(ps, "n1", codeNoNetwork) {
		t.Fatalf("a VM without a network is not refused: %+v", ps)
	}
}

func TestHostileTextIsRefusedBeforeRendering(t *testing.T) {
	type change func(g *connectors.Graph, d *connectors.Deployment)
	setField := func(id, field string, v any) change {
		return func(g *connectors.Graph, _ *connectors.Deployment) { g.Blocks[id].Fields[field] = v }
	}
	setSetting := func(name string, v any) change {
		return func(_ *connectors.Graph, d *connectors.Deployment) { d.Settings[name] = v }
	}
	rename := func(id, name string) change {
		return func(g *connectors.Graph, _ *connectors.Deployment) {
			b := g.Blocks[id]
			b.Name = name
			g.Blocks[id] = b
		}
	}
	cases := map[string]change{
		"name with a quote":             rename("n1", "web\" }\nresource \"x\" \"y\" {"),
		"name with interpolation":       rename("n2", "${file(\"/etc/passwd\")}"),
		"name in capitals":              rename("n3", "Web"),
		"image with a quote":            setSetting("image", "cirros\" } data \"external\" \"x\" {"),
		"image with interpolation":      setSetting("image", "${path.module}"),
		"image with a newline":          setSetting("image", "cirros\nprovisioner"),
		"image with a backslash":        setSetting("image", `cirros\`),
		"image with brackets and a ref": setSetting("image", "Ubuntu [${var.x}]"),
		"empty image":                   setSetting("image", "  "),
		"flavor with a directive":       setSetting("flavor_small", "%{ if true }m1%{ endif }"),
		"external network with braces":  setSetting("external_network", "public}"),
		"ingress with a shell command":  setField("n3", "ingress", "tcp/22; rm -rf /"),
		"ingress port out of range":     setField("n3", "ingress", "tcp/70000"),
		"ingress range backwards":       setField("n3", "ingress", "tcp/90-80"),
		"ingress unknown protocol":      setField("n3", "ingress", "sctp/22"),
		"ingress with a quote":          setField("n3", "ingress", `tcp/22"`),
		"source not a range":            setField("n3", "source_cidr", `0.0.0.0/0" }`),
		"source with host bits":         setField("n3", "source_cidr", "10.0.0.1/24"),
		"cidr not a range":              setField("n2", "cidr", "${var.x}"),
		"cidr is IPv6":                  setField("n2", "cidr", "fd00::/64"),
		"cidr too wide":                 setField("n2", "cidr", "10.0.0.0/4"),
		"volume size zero":              setField("n4", "size_gb", float64(0)),
		"volume size not whole":         setField("n4", "size_gb", 1.5),
		"size not allowed":              setField("n1", "size", "huge"),
		"unknown field":                 setField("n1", "user_data", "#!/bin/sh"),
		"unknown setting":               setSetting("cloud", "x"),
		"node id with a quote": func(g *connectors.Graph, _ *connectors.Deployment) {
			b := g.Blocks["n4"]
			delete(g.Blocks, "n4")
			b.ID = `n4"x`
			g.Blocks[b.ID] = b
			// The volume keeps its edge to the VM.
			g.Edges[2] = connectors.Edge{Source: b.ID, Target: "n1"}
		},
		"prefix with a quote": func(_ *connectors.Graph, d *connectors.Deployment) { d.Prefix = `c1"` },
		"volume on two VMs": func(g *connectors.Graph, _ *connectors.Deployment) {
			g.Blocks["n5"] = connectors.Block{ID: "n5", Kind: connectors.KindVM, Name: "api", Fields: map[string]any{}}
			g.Edges = append(g.Edges, connectors.Edge{Source: "n5", Target: "n2"}, connectors.Edge{Source: "n4", Target: "n5"})
		},
		"two firewalls with one name": func(g *connectors.Graph, _ *connectors.Deployment) {
			g.Blocks["n5"] = connectors.Block{ID: "n5", Kind: connectors.KindFirewall, Name: "fw", Fields: map[string]any{}}
		},
		"edge the wrong way": func(g *connectors.Graph, _ *connectors.Deployment) {
			g.Edges = append(g.Edges, connectors.Edge{Source: "n2", Target: "n1"})
		},
	}
	c := newConnector(nil)
	for name, apply := range cases {
		t.Run(name, func(t *testing.T) {
			g := sampleGraph()
			d := testDeployment(fixedDeployment)
			apply(&g, &d)
			// B's core runs both checks before any render.
			if ps := capability.CheckConnector(c, g, d).Problems; len(ps) == 0 {
				t.Fatal("the checks accepted it")
			}
			// What the connector itself refuses, Render refuses too.
			if len(c.Validate(g, d)) > 0 {
				if files, err := c.Render(g, d); err == nil {
					t.Fatalf("rendered it:\n%s", joined(files))
				}
			}
		})
	}
}

func TestValidAcceptedVariants(t *testing.T) {
	c := newConnector(nil)
	g := sampleGraph()
	g.Blocks["n3"].Fields["ingress"] = ""
	g.Blocks["n3"].Fields["source_cidr"] = "2001:db8::/32"
	g.Blocks["n1"].Fields["public"] = false
	g.Blocks["n5"] = connectors.Block{ID: "n5", Kind: connectors.KindNetwork, Name: "back", Fields: map[string]any{}}
	g.Edges = append(g.Edges, connectors.Edge{Source: "n1", Target: "n5"})
	d := testDeployment(fixedDeployment)
	d.Settings = map[string]any{"image": "Ubuntu 24.04.2 LTS (x86_64) [2025-04-03]"}
	if ps := c.Validate(g, d); len(ps) > 0 {
		t.Fatalf("refused: %+v", ps)
	}
	files, err := c.Render(g, d)
	if err != nil {
		t.Fatal(err)
	}
	main := string(files["main.tf"])
	for _, want := range []string{
		`name_regex  = "^Ubuntu 24\\.04\\.2 LTS \\(x86_64\\) \\[2025-04-03\\]"`,
		`flavor_name     = "m1.tiny"`,
		`cidr       = "10.0.0.0/24"`,
		"depends_on = [openstack_networking_subnet_v2.b_n2, openstack_networking_subnet_v2.b_n5]",
	} {
		if !strings.Contains(main, want) {
			t.Errorf("main.tf lacks %q:\n%s", want, main)
		}
	}
	for _, gone := range []string{"secgroup_rule", "floatingip"} {
		if strings.Contains(main, gone) {
			t.Errorf("main.tf holds %q", gone)
		}
	}
}

func TestStrEscapesForHCL(t *testing.T) {
	if got, want := str("a\"b${c}%{d}\\\n"), `"a\"b$${c}%%{d}\\\n"`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestRunsGoThroughTheSharedRunner(t *testing.T) {
	k := newKeystone(t)
	fake := &fakeTerraform{}
	c := newTestConnector(t, fake)
	d := testDeployment(uuid.NewString())
	d.Connection.Credentials = k.secrets(goodSecret)
	ctx := context.Background()
	var log bytes.Buffer

	plan, err := c.Plan(ctx, connectors.Run{ID: uuid.NewString(), Action: connectors.ActionPlan, Log: &log}, sampleGraph(), d)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan != (connectors.PlanResult{Add: 11}) {
		t.Fatalf("plan result %+v", plan)
	}
	task, files := fake.last()
	if task.Args[0] != "plan" {
		t.Fatalf("last step %v", task.Args)
	}
	want := map[string]string{
		"OS_AUTH_URL":                      k.authURL(),
		"OS_REGION_NAME":                   testRegion,
		"OS_APPLICATION_CREDENTIAL_ID":     goodCredID,
		"OS_APPLICATION_CREDENTIAL_SECRET": goodSecret,
	}
	for name, v := range want {
		if task.Env[name] != v {
			t.Errorf("run env %s is not the connection's value", name)
		}
	}
	for name := range task.Env {
		if strings.HasPrefix(name, "OS_") && want[name] == "" {
			t.Errorf("run env has an extra %s", name)
		}
	}
	for _, name := range []string{"main.tf", "outputs.tf", "versions.tf", "zz_clutter_backend.tf"} {
		if _, ok := files[name]; !ok {
			t.Errorf("the run folder lacks %s", name)
		}
	}

	res, err := c.Apply(ctx, connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &log}, sampleGraph(), d)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	// Only the blocks the apply built or changed count: n3 has an id but
	// was left as it was.
	if len(res.Resources) != 2 || res.Resources["n1"] != "9b1d5a8e-0000-4000-8000-0000000000a1" || res.Resources["n2"] == "" {
		t.Fatalf("apply resources %v", res.Resources)
	}
	// n1 is the sample's public VM: it has both addresses.
	wantResults := []connectors.Result{
		{Name: connectors.ResultPrivateAddress, Value: "10.0.0.12"},
		{Name: connectors.ResultPublicAddress, Value: "172.24.4.10"},
	}
	if !reflect.DeepEqual(res.Results["n1"], wantResults) {
		t.Fatalf("apply results %+v", res.Results)
	}

	if _, err := c.Destroy(ctx, connectors.Run{ID: uuid.NewString(), Action: connectors.ActionDestroy, Log: &log}, d); err != nil {
		t.Fatalf("destroy: %v", err)
	}
	task, files = fake.last()
	if task.Args[0] != "destroy" {
		t.Fatalf("last step %v", task.Args)
	}
	if _, ok := files["main.tf"]; ok {
		t.Error("destroy was given the resources")
	}
	if !strings.Contains(log.String(), "Destroy complete!") {
		t.Errorf("the run log lacks Terraform's output:\n%s", log.String())
	}

	if _, err := c.Plan(ctx, connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &log}, sampleGraph(), d); err == nil {
		t.Error("a plan ran for a run marked apply")
	}
}
