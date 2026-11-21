package aws

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

// instanceOverride is the aws_instance block override.tf renders for addr,
// which must be there: a test of what it lacks passes on a missing block.
func instanceOverride(t *testing.T, override, addr string) string {
	t.Helper()
	got := resourceBlock(override, "aws_instance", addr)
	if got == "" {
		t.Fatalf("override.tf has no block for %s", addr)
	}
	return got
}

// TestTableKeyChangeIsANewName: DynamoDB fixes a table's key, so a key change
// replaces the table, and the replacement is made before the old table is
// deleted when an API reaches the table. A table with the default key keeps
// its plain name (so a live table is not replaced); any other key and key
// type carry their hash, so each key change, the key type's included, is a
// new name. Every name keeps the deployment prefix the users' boundary allows.
func TestTableKeyChangeIsANewName(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	nameOf := func(fields map[string]any) string {
		g := fullGraph()
		b := g.Blocks["table"]
		b.Fields = fields
		g.Blocks["table"] = b
		main := string(renderFor(t, c, g, d)["main.tf"])
		m := regexp.MustCompile(`(?s)resource "aws_dynamodb_table" "[^"]+" \{\n  name         = "([^"]+)"`).FindStringSubmatch(main)
		if m == nil {
			t.Fatal("no table name in main.tf")
		}
		return m[1]
	}
	plain := nameOf(map[string]any{})
	if plain != d.Prefix+"-orders" {
		t.Errorf("a table with the default key is named %q, want %q", plain, d.Prefix+"-orders")
	}
	if got := nameOf(map[string]any{"key": "id", "key_type": "string"}); got != plain {
		t.Errorf("the default key written out renames the table: %q", got)
	}
	seen := map[string]string{"default": plain}
	for label, fields := range map[string]map[string]any{
		"key name":  {"key": "order_id"},
		"key type":  {"key_type": "number"},
		"both":      {"key": "order_id", "key_type": "number"},
		"binary id": {"key_type": "binary"},
	} {
		name := nameOf(fields)
		if !regexp.MustCompile(`^`+d.Prefix+`-orders-[0-9a-f]{8}$`).MatchString(name) || len(name) > maxName {
			t.Errorf("%s: table name %q is not the prefix, the block name and a hash", label, name)
		}
		for other, prev := range seen {
			if prev == name {
				t.Errorf("%s and %s give the same table name %q", label, other, name)
			}
		}
		seen[label] = name
	}
	if a, b := nameOf(map[string]any{"key_type": "number"}), nameOf(map[string]any{"key_type": "number"}); a != b {
		t.Errorf("the same key renders two names: %q and %q", a, b)
	}

	// What reads the name reads the resource, so it follows a new name.
	files := renderFor(t, c, fullGraph(), d)
	all := joined(files)
	for _, want := range []string{
		`CLUTTER_ORDERS_TABLE = "${aws_dynamodb_table.` + d.Prefix + `_table.name}"`,
		`aws_dynamodb_table.` + d.Prefix + `_table.arn,`,
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the rendered files lack %q", want)
		}
	}
}

// TestVolumeZoneNeverReadsTheInstance: a volume's zone forces a new volume,
// and a replaced instance's zone is unknown until it exists, so a volume that
// read it would be replaced, with its data, whenever its VM is. The zone comes
// from the VM's subnet, or from the zone a VM in the default VPC is pinned to.
func TestVolumeZoneNeverReadsTheInstance(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	p := d.Prefix

	// In a network: the public VM "web" carries both volumes.
	files := renderFor(t, c, fullGraph(), d)
	all := joined(files)
	if want := `availability_zone = aws_subnet.` + p + `_net1_public_subnet.availability_zone`; strings.Count(all, want) != 2 {
		t.Errorf("the two volumes do not take the zone of web's subnet (%q)", want)
	}
	if regexp.MustCompile(`aws_instance\.[^.\s]+\.availability_zone`).MatchString(all) {
		t.Error("a rendered file reads an instance's zone")
	}
	if strings.Contains(instanceOverride(t, string(files["override.tf"]), p+"_web"), "availability_zone") {
		t.Error("a VM in a network has its zone pinned; its subnet sets it")
	}
	if strings.Contains(string(files["defaults.tf"]), "aws_availability_zones") {
		t.Error("defaults.tf lists zones for a diagram with no VM in the default VPC")
	}

	// In the default VPC: the VM with a volume is pinned, the one without is not.
	g := connectors.Graph{
		Blocks: map[string]connectors.Block{
			"app":   {ID: "app", Kind: connectors.KindVM, Name: "app", Fields: map[string]any{}},
			"other": {ID: "other", Kind: connectors.KindVM, Name: "other", Fields: map[string]any{}},
			"disk":  {ID: "disk", Kind: connectors.KindVolume, Name: "disk", Fields: map[string]any{}},
		},
		Edges: []connectors.Edge{{Source: "disk", Target: "app"}},
	}
	files = renderFor(t, c, g, d)
	zone := `data.aws_availability_zones.` + p + `_app_azs.names[0]`
	for file, want := range map[string]string{
		"main.tf":     `availability_zone = ` + zone,
		"override.tf": `availability_zone           = ` + zone,
		"defaults.tf": `data "aws_availability_zones" "` + p + `_app_azs" {`,
	} {
		if !strings.Contains(string(files[file]), want) {
			t.Errorf("%s lacks %q:\n%s", file, want, files[file])
		}
	}
	if strings.Contains(instanceOverride(t, string(files["override.tf"]), p+"_other"), "availability_zone") {
		t.Error("a VM with no volume has its zone pinned")
	}
	if strings.Contains(string(files["defaults.tf"]), p+"_other_azs") {
		t.Error("defaults.tf lists zones for a VM with no volume")
	}
}

// TestPublicAddressArgument: a private VM in a network leaves
// associate_public_ip_address to its subnet, which gives none; a public VM
// asks for one, and a private VM in the default VPC, whose subnets give one,
// refuses it.
func TestPublicAddressArgument(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	p := d.Prefix
	override := string(renderFor(t, c, fullGraph(), d)["override.tf"])
	if got := instanceOverride(t, override, p+"_web"); !strings.Contains(got, "associate_public_ip_address = true") {
		t.Errorf("the public VM does not ask for a public address:\n%s", got)
	}
	if got := instanceOverride(t, override, p+"_worker"); strings.Contains(got, "associate_public_ip_address") {
		t.Errorf("the private VM in a network sets associate_public_ip_address:\n%s", got)
	}
	g := connectors.Graph{Blocks: map[string]connectors.Block{
		"solo": {ID: "solo", Kind: connectors.KindVM, Name: "solo", Fields: map[string]any{}},
	}}
	override = string(renderFor(t, c, g, d)["override.tf"])
	if got := instanceOverride(t, override, p+"_solo"); !strings.Contains(got, "associate_public_ip_address = false") {
		t.Errorf("the private VM in the default VPC does not refuse a public address:\n%s", got)
	}
}

// TestRenameChangesOnlyTags: a security group's or key pair's name forces a
// new one, AWS keeps a group an instance uses, and a new key pair name
// replaces the instance. So their names come from the node id, the block's
// name is their Name tag, and each ignores later changes to its name, so one
// named from the block's name stays.
func TestRenameChangesOnlyTags(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	p := d.Prefix
	render := func(rename bool) map[string][]byte {
		g := fullGraph()
		g.Edges = append(g.Edges, connectors.Edge{Source: "web", Target: "worker"})
		g.Blocks["web"].Fields[connectors.FieldSSHKey] = testSSHKey
		if rename {
			for _, id := range []string{"fw", "web", "worker"} {
				b := g.Blocks[id]
				b.Name += "-renamed"
				g.Blocks[id] = b
			}
		}
		return renderFor(t, c, g, d)
	}
	for _, rename := range []bool{false, true} {
		files := render(rename)
		suffix := ""
		if rename {
			suffix = "-renamed"
		}
		for _, r := range []struct{ file, kind, addr, name, tag, ignored string }{
			{"main.tf", "aws_security_group", p + "_fw", `name        = "` + p + `.fw.firewall"`, `Name = "` + p + `-web-fw` + suffix + `"`, "name"},
			{"iam.tf", "aws_security_group", p + "_worker_peers", `name        = "` + p + `.worker.peers"`, `Name = "` + p + `-worker` + suffix + `.peers"`, "name"},
			{"defaults.tf", "aws_key_pair", p + "_web", `key_name   = "` + p + `.web.key"`, `Name = "` + p + `-web` + suffix + `"`, "key_name"},
		} {
			block := resourceBlock(string(files[r.file]), r.kind, r.addr)
			for _, want := range []string{r.name, r.tag, "ignore_changes = [" + r.ignored + "]"} {
				if !strings.Contains(block, want) {
					t.Errorf("rename=%v: %s %s lacks %q:\n%s", rename, r.kind, r.addr, want, block)
				}
			}
		}
	}
}

// TestVMHelpSaysWhatReplaces: the VM's help names each change that replaces
// a deployed server on AWS, and that attached volumes stay.
func TestVMHelpSaysWhatReplaces(t *testing.T) {
	spec, _ := manifest().Block(connectors.KindVM)
	help := strings.Join(spec.Help, "\n")
	for _, want := range []string{"Reachable from the internet", "the startup script", "the network", "Adding an SSH key",
		"replaces a deployed server: its own disk is lost", "attached volumes move over",
		"Changing the size restarts the server",
		"A changed key reaches new servers only; a running server keeps the key it started with."} {
		if !strings.Contains(help, want) {
			t.Errorf("the VM help lacks %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "Size") || strings.Contains(help, "instance type") {
		t.Error("the VM help says a size change replaces the server; it changes in place")
	}
	network, _ := manifest().Block(connectors.KindNetwork)
	if !strings.Contains(strings.Join(network.Help, "\n"), "adds a NAT gateway: about $40 a month plus data") {
		t.Errorf("the network help does not name the NAT gateway's cost: %q", network.Help)
	}
	if err := manifest().Check(); err != nil {
		t.Fatal(err)
	}
}
