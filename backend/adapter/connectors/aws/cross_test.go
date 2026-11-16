package aws

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/google/uuid"
)

// exportGraph is this AWS part of a split deployment: a bucket, a table and a
// public VM here, and a VM on another cloud ("remote", left out here) that
// reads both and gets the VM's address.
func exportGraph() connectors.Graph {
	return connectors.Graph{
		Blocks: map[string]connectors.Block{
			"files":  {ID: "files", Kind: connectors.KindBucket, Name: "files", Fields: map[string]any{}},
			"orders": {ID: "orders", Kind: connectors.KindTable, Name: "orders", Fields: map[string]any{}},
			"web":    {ID: "web", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{"public": true}},
			"remote": {ID: "remote", Kind: connectors.KindVM, Name: "worker", Fields: map[string]any{}},
		},
		Edges: []connectors.Edge{{Source: "remote", Target: "files"}, {Source: "remote", Target: "orders"}, {Source: "remote", Target: "web"}},
	}
}

func exportDeployment() connectors.Deployment {
	d := testDeployment("ca-central-1")
	d.Exclusions = []string{"remote"}
	d.Cross = &connectors.Cross{Exports: []connectors.CrossEdge{
		connectors.NewCrossEdge("remote", "files", connectors.EdgePermission),
		connectors.NewCrossEdge("remote", "orders", connectors.EdgePermission),
		connectors.NewCrossEdge("remote", "web", connectors.EdgeAddress),
	}}
	return d
}

// importGraph is this AWS part: a VM that reads a bucket placed on another
// cloud ("files", left out here).
func importGraph() connectors.Graph {
	return connectors.Graph{
		Blocks: map[string]connectors.Block{
			"vm":    {ID: "vm", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{connectors.FieldStartupScript: "echo start"}},
			"files": {ID: "files", Kind: connectors.KindBucket, Name: "files", Fields: map[string]any{}},
		},
		Edges: []connectors.Edge{{Source: "vm", Target: "files"}},
	}
}

func importDeployment(withValues bool) connectors.Deployment {
	d := testDeployment("ca-central-1")
	d.Exclusions = []string{"files"}
	imp := connectors.CrossImport{CrossEdge: connectors.NewCrossEdge("vm", "files", connectors.EdgePermission)}
	if withValues {
		imp.Values = map[string]connectors.Value{
			connectors.ValueBucket:          {Text: "c9-files-0a1b"},
			connectors.ValueRegion:          {Text: "us-west-2"},
			connectors.ValueAccessKeyID:     {Text: "AKIAOTHERCLOUD000001"},
			connectors.ValueSecretAccessKey: {Text: fakeScopedSecret, Sensitive: true},
		}
	}
	d.Cross = &connectors.Cross{Imports: []connectors.CrossImport{imp}}
	return d
}

// vmEdgesGraph is a VM that reads a bucket and a table and gets another VM's
// address, all on AWS.
func vmEdgesGraph() connectors.Graph {
	return connectors.Graph{
		Blocks: map[string]connectors.Block{
			"app":    {ID: "app", Kind: connectors.KindVM, Name: "app", Fields: map[string]any{connectors.FieldStartupScript: "#!/usr/bin/env python3\nprint('hi')"}},
			"db":     {ID: "db", Kind: connectors.KindVM, Name: "db", Fields: map[string]any{}},
			"files":  {ID: "files", Kind: connectors.KindBucket, Name: "files", Fields: map[string]any{}},
			"orders": {ID: "orders", Kind: connectors.KindTable, Name: "orders", Fields: map[string]any{}},
		},
		Edges: []connectors.Edge{{Source: "app", Target: "db"}, {Source: "app", Target: "files"}, {Source: "app", Target: "orders"}},
	}
}

func TestAScopedKeyCanOnlyReadItsBucketOrTable(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := exportDeployment()
	if ps := c.CrossCheck(exportGraph(), d); len(ps) > 0 {
		t.Fatalf("cross problems: %+v", ps)
	}
	files := renderFor(t, c, exportGraph(), d)
	cross := string(files["cross.tf"])
	m := newModel(exportGraph(), d)
	bucket, table, web := m.blocks["files"].Addr, m.blocks["orders"].Addr, m.blocks["web"].Addr
	kb, kt, ka := d.Cross.Exports[0].Key, d.Cross.Exports[1].Key, d.Cross.Exports[2].Key
	userB := d.Prefix + "_" + kb
	for _, want := range []string{
		`resource "aws_iam_user" "` + userB + `"`,
		`name                 = "` + d.Prefix + "-" + kb + `"`,
		`path                 = "/clutter/"`,
		// The users' boundary, never the roles' wider one.
		`permissions_boundary = var.user_permissions_boundary_arn`,
		`ManagedBy = "Clutter"`,
		`resource "aws_iam_user_policy" "` + userB + `"`,
		`Resource = aws_s3_bucket.` + bucket + `.arn`,
		`Resource = "${aws_s3_bucket.` + bucket + `.arn}/*"`,
		`aws_dynamodb_table.` + table + `.arn,`,
		`resource "aws_iam_access_key" "` + userB + `"`,
		`output "cross_` + kb + `_bucket"`,
		`output "cross_` + kt + `_table"`,
		`output "cross_` + kb + `_region"`,
		`value = "ca-central-1"`,
		"output \"cross_" + kb + "_secret_access_key\" {\n  value     = aws_iam_access_key." + userB + ".secret\n  sensitive = true\n}",
		`output "cross_` + ka + `_address" {` + "\n  value = aws_instance." + web + ".public_ip",
	} {
		if !strings.Contains(cross, want) {
			t.Errorf("cross.tf lacks %q\n%s", want, cross)
		}
	}
	// Only reads, only on the one resource, and nothing in IAM.
	for _, refused := range []string{`"*"`, "s3:PutObject", "s3:DeleteObject", "dynamodb:PutItem", "iam:", "force_destroy"} {
		if strings.Contains(cross, refused) {
			t.Errorf("cross.tf holds %q", refused)
		}
	}
	if strings.Count(cross, `resource "aws_iam_access_key"`) != 2 {
		t.Fatal("not one key per permission edge")
	}
	// The users' boundary in the role stack allows reads only on
	// arn:aws:s3:::c????????-* and table/c????????-*, so a key reads the
	// bucket and table it is for only while B names them that way.
	boundaryName := regexp.MustCompile(`^c[0-9a-f]{8}-`)
	main := string(files["main.tf"])
	for _, name := range []string{m.bucketName(m.blocks["files"]), m.physical(m.blocks["orders"], maxName)} {
		if !boundaryName.MatchString(name) || !strings.Contains(main, `"`+name) {
			t.Errorf("name %q does not match the users' boundary pattern, or main.tf does not use it", name)
		}
	}
}

func TestValuesFromAnotherCloudReachTheVMOnlyAsSensitiveUserData(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := importDeployment(true)
	if ps := c.CrossCheck(importGraph(), d); len(ps) > 0 {
		t.Fatalf("cross problems: %+v", ps)
	}
	files := renderFor(t, c, importGraph(), d)
	override := string(files["override.tf"])
	if !strings.Contains(override, `user_data_base64            = sensitive(base64encode("#!/bin/sh\n`) ||
		!strings.Contains(override, `CLUTTER_FILES_SECRET_ACCESS_KEY='`+fakeScopedSecret+`'`) ||
		!strings.Contains(override, `CLUTTER_FILES_BUCKET='c9-files-0a1b'`) ||
		!strings.Contains(override, "user_data_replace_on_change = true") {
		t.Fatalf("override.tf:\n%s", override)
	}
	if _, ok := files["cross.tf"]; ok {
		t.Fatal("a part that exports nothing rendered cross.tf")
	}

	// A preview never holds the other cloud's key, even when it is handed one.
	preview, err := c.Preview(context.Background(), importGraph(), d)
	if err != nil {
		t.Fatal(err)
	}
	all := joined(preview)
	if strings.Contains(all, fakeScopedSecret) || strings.Contains(all, "AKIAOTHERCLOUD") || !strings.Contains(all, connectors.Placeholder()) {
		t.Fatalf("preview:\n%s", all)
	}
}

func TestVMEdgesOnOneCloudUseTheInstanceRoleAndReferences(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	files := renderFor(t, c, vmEdgesGraph(), d)
	m := newModel(vmEdgesGraph(), d)
	app := m.blocks["app"].Addr
	iam, override := string(files["iam.tf"]), string(files["override.tf"])
	for _, want := range []string{
		`role = aws_iam_role.` + app + `_ssm_role.id`,
		`Resource = aws_s3_bucket.` + m.blocks["files"].Addr + `.arn`,
		`aws_dynamodb_table.` + m.blocks["orders"].Addr + `.arn,`,
	} {
		if !strings.Contains(iam, want) {
			t.Errorf("iam.tf lacks %q\n%s", want, iam)
		}
	}
	for _, want := range []string{
		`CLUTTER_DB_ADDRESS='${aws_instance.` + m.blocks["db"].Addr + `.private_ip}'`,
		`CLUTTER_FILES_BUCKET='${aws_s3_bucket.` + m.blocks["files"].Addr + `.bucket}'`,
		`CLUTTER_ORDERS_TABLE='${aws_dynamodb_table.` + m.blocks["orders"].Addr + `.name}'`,
		`CLUTTER_FILES_REGION='ca-central-1'`,
		`user_data_base64            = base64encode(`,
	} {
		if !strings.Contains(override, want) {
			t.Errorf("override.tf lacks %q\n%s", want, override)
		}
	}
	// Terraform drops replace_triggered_by from an override file, so it must
	// be in the instance's own block (ADR-026).
	if strings.Contains(override, "replace_triggered_by") {
		t.Errorf("override.tf sets replace_triggered_by, which Terraform ignores there\n%s", override)
	}
	trigger := `replace_triggered_by = [aws_instance.` + m.blocks["db"].Addr + `.private_ip, aws_s3_bucket.` + m.blocks["files"].Addr + `.bucket, aws_dynamodb_table.` + m.blocks["orders"].Addr + `.name]`
	if !strings.Contains(joined(files), trigger) {
		t.Errorf("no file holds %q", trigger)
	}
	if strings.Contains(override, "sensitive(") {
		t.Fatal("nothing secret, yet the user data is sensitive")
	}
	// No key is made for an edge inside one cloud.
	if _, ok := files["cross.tf"]; ok {
		t.Fatal("cross.tf on one cloud")
	}
}

func TestVMsThatGetEachOthersAddressAreRefused(t *testing.T) {
	g := vmEdgesGraph()
	g.Edges = append(g.Edges, connectors.Edge{Source: "db", Target: "app"})
	c, _, _ := newTestConnector(t)
	found := false
	for _, p := range c.Validate(g, testDeployment("ca-central-1")) {
		found = found || p.Code == connectors.CodeAddressCycle
	}
	if !found {
		t.Fatal("a circle of addresses was not refused")
	}
}

func TestCrossCheckRefusesWhatAWSCannotHandOver(t *testing.T) {
	c, _, _ := newTestConnector(t)
	g := exportGraph()
	g.Blocks["web"] = connectors.Block{ID: "web", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{}}
	ps := c.CrossCheck(g, exportDeployment())
	if len(ps) != 1 || ps[0].Code != connectors.CodeCrossNeedsPublic || ps[0].NodeID != "web" {
		t.Fatalf("a private VM's address: %+v", ps)
	}
	// Rendering refuses it too.
	if _, err := c.Preview(context.Background(), g, exportDeployment()); err == nil {
		t.Fatal("a refused export rendered")
	}
	d := exportDeployment()
	d.Cross.Exports = append(d.Cross.Exports, connectors.CrossEdge{Source: "remote", Target: "files", Class: connectors.EdgeAddress, Key: "bad"})
	if ps := c.CrossCheck(exportGraph(), d); len(ps) == 0 {
		t.Fatal("a malformed export passed")
	}
}

func TestApplyCrossReturnsTheExportsAndNeverLogsTheKey(t *testing.T) {
	c, _, _ := newTestConnector(t)
	var log bytes.Buffer
	run := connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &log}
	res, out, err := c.ApplyCross(context.Background(), run, exportGraph(), exportDeployment())
	if err != nil {
		t.Fatalf("%v\n%s", err, log.String())
	}
	if len(res.Resources) == 0 {
		t.Fatal("nothing reported built")
	}
	d := exportDeployment()
	kb, ka := d.Cross.Exports[0].Key, d.Cross.Exports[2].Key
	if s := out[kb][connectors.ValueSecretAccessKey]; s.Text != fakeScopedSecret || !s.Sensitive {
		t.Fatalf("secret %v", s)
	}
	if out[ka][connectors.ValueAddress].Text != "198.51.100.7" {
		t.Fatalf("address %v", out[ka])
	}
	if strings.Contains(log.String(), fakeScopedSecret) {
		t.Fatal("the scoped key reached the run's log")
	}

	// A plan reads what the state holds, and a part with no exports prints
	// no outputs at all.
	_, out, err = c.PlanCross(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionPlan, Log: &log}, importGraph(), importDeployment(false))
	if err != nil || len(out) != 0 {
		t.Fatalf("plan: %v, %v", out, err)
	}
	// An apply never renders a placeholder for a value it was not handed.
	_, _, err = c.ApplyCross(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &log}, importGraph(), importDeployment(false))
	if !errors.Is(err, connectors.ErrNothingBuilt) {
		t.Fatalf("apply with missing values: %v", err)
	}
}
