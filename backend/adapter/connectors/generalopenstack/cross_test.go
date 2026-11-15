package generalopenstack

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/google/uuid"
)

const otherCloudSecret = "otherCloudSecret/0123456789abcdefABCDEF"

// importGraph is the sample with the VM reading a bucket and getting a VM's
// address, both on other clouds.
func importGraph() connectors.Graph {
	g := sampleGraph()
	g.Blocks["files"] = connectors.Block{ID: "files", Kind: connectors.KindBucket, Name: "files", Fields: map[string]any{}}
	g.Blocks["api"] = connectors.Block{ID: "api", Kind: connectors.KindVM, Name: "api-box", Fields: map[string]any{"public": true}}
	g.Edges = append(g.Edges, connectors.Edge{Source: "n1", Target: "files"}, connectors.Edge{Source: "n1", Target: "api"})
	return g
}

func importDeployment(withValues bool) connectors.Deployment {
	d := testDeployment(fixedDeployment)
	d.Exclusions = []string{"files", "api"}
	key := connectors.CrossImport{CrossEdge: connectors.NewCrossEdge("n1", "files", connectors.EdgePermission)}
	addr := connectors.CrossImport{CrossEdge: connectors.NewCrossEdge("n1", "api", connectors.EdgeAddress)}
	if withValues {
		key.Values = map[string]connectors.Value{
			connectors.ValueBucket:          {Text: "c9-files-0a1b"},
			connectors.ValueRegion:          {Text: "ca-central-1"},
			connectors.ValueAccessKeyID:     {Text: "AKIAOTHERCLOUD000001"},
			connectors.ValueSecretAccessKey: {Text: otherCloudSecret, Sensitive: true},
		}
		addr.Values = map[string]connectors.Value{connectors.ValueAddress: {Text: "198.51.100.7"}}
	}
	d.Cross = &connectors.Cross{Imports: []connectors.CrossImport{key, addr}}
	return d
}

// exportDeployment hands the sample's public VM's address to a VM on another
// cloud ("remote", left out here).
func exportGraph() connectors.Graph {
	g := sampleGraph()
	g.Blocks["remote"] = connectors.Block{ID: "remote", Kind: connectors.KindVM, Name: "worker", Fields: map[string]any{}}
	g.Edges = append(g.Edges, connectors.Edge{Source: "remote", Target: "n1"})
	return g
}

func exportDeployment() connectors.Deployment {
	d := testDeployment(fixedDeployment)
	d.Exclusions = []string{"remote"}
	d.Cross = &connectors.Cross{Exports: []connectors.CrossEdge{connectors.NewCrossEdge("remote", "n1", connectors.EdgeAddress)}}
	return d
}

func TestValuesFromAnotherCloudReachTheServerOnlyAsSensitiveUserData(t *testing.T) {
	c := newConnector(nil)
	d := importDeployment(true)
	if ps := c.CrossCheck(importGraph(), d); len(ps) > 0 {
		t.Fatalf("cross problems: %+v", ps)
	}
	if ps := c.Validate(importGraph(), d); len(ps) > 0 {
		t.Fatalf("problems: %+v", ps)
	}
	files, err := c.Render(importGraph(), d)
	if err != nil {
		t.Fatal(err)
	}
	main := string(files["main.tf"])
	for _, want := range []string{
		`user_data       = sensitive("#!/bin/sh\n`,
		`CLUTTER_FILES_SECRET_ACCESS_KEY='` + otherCloudSecret + `'`,
		`CLUTTER_API_BOX_ADDRESS='198.51.100.7'`,
	} {
		if !strings.Contains(main, want) {
			t.Errorf("main.tf lacks %q\n%s", want, main)
		}
	}
	preview, err := c.Preview(context.Background(), importGraph(), d)
	if err != nil {
		t.Fatal(err)
	}
	if p := string(preview["main.tf"]); strings.Contains(p, otherCloudSecret) || strings.Contains(p, "AKIAOTHERCLOUD") || !strings.Contains(p, connectors.Placeholder()) {
		t.Fatalf("preview:\n%s", p)
	}
}

func TestAnotherVMsAddressOnOneCloudIsAReference(t *testing.T) {
	g := sampleGraph()
	g.Blocks["db"] = connectors.Block{ID: "db", Kind: connectors.KindVM, Name: "db", Fields: map[string]any{"size": "small"}}
	g.Edges = append(g.Edges, connectors.Edge{Source: "db", Target: "n2"}, connectors.Edge{Source: "n1", Target: "db"})
	c := newConnector(nil)
	files, err := c.Render(g, testDeployment(fixedDeployment))
	if err != nil {
		t.Fatal(err)
	}
	main := string(files["main.tf"])
	if !strings.Contains(main, `CLUTTER_DB_ADDRESS='${openstack_compute_instance_v2.b_db.access_ip_v4}'`) || strings.Contains(main, "sensitive(") {
		t.Fatalf("main.tf:\n%s", main)
	}
	g.Edges = append(g.Edges, connectors.Edge{Source: "db", Target: "n1"})
	found := false
	for _, p := range c.Validate(g, testDeployment(fixedDeployment)) {
		found = found || p.Code == connectors.CodeAddressCycle
	}
	if !found {
		t.Fatal("a circle of addresses was not refused")
	}
}

func TestOpenStackHandsOverOnlyAPublicServersAddress(t *testing.T) {
	c := newConnector(nil)
	files, err := c.Render(exportGraph(), exportDeployment())
	if err != nil {
		t.Fatal(err)
	}
	key := exportDeployment().Cross.Exports[0].Key
	if !strings.Contains(string(files["outputs.tf"]), `output "cross_`+key+`_address" {`+"\n  value = openstack_networking_floatingip_v2.b_n1.address") {
		t.Fatalf("outputs.tf:\n%s", files["outputs.tf"])
	}

	private := exportGraph()
	private.Blocks["n1"].Fields["public"] = false
	if ps := c.CrossCheck(private, exportDeployment()); len(ps) != 1 || ps[0].Code != connectors.CodeCrossNeedsPublic {
		t.Fatalf("a private server's address: %+v", ps)
	}
	// OpenStack has no key scoped to one resource to hand over.
	d := exportDeployment()
	d.Cross.Exports = []connectors.CrossEdge{connectors.NewCrossEdge("remote", "n1", connectors.EdgePermission)}
	if ps := c.CrossCheck(exportGraph(), d); len(ps) != 1 || ps[0].Code != connectors.CodeCrossInvalid {
		t.Fatalf("a permission export: %+v", ps)
	}
	if _, err := c.Render(exportGraph(), d); err == nil {
		t.Fatal("a refused export rendered")
	}
}

func TestApplyCrossReturnsTheAddress(t *testing.T) {
	k := newKeystone(t)
	c := newTestConnector(t, &fakeTerraform{})
	d := exportDeployment()
	d.ID = uuid.NewString()
	d.Connection.Credentials = k.secrets(goodSecret)
	var log bytes.Buffer
	_, out, err := c.ApplyCross(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &log}, exportGraph(), d)
	if err != nil {
		t.Fatalf("%v\n%s", err, log.String())
	}
	if got := out[d.Cross.Exports[0].Key][connectors.ValueAddress]; got.Text != "172.24.4.77" || got.Sensitive {
		t.Fatalf("address %v", got)
	}
	// An apply never renders a placeholder for a value it was not handed.
	i := importDeployment(false)
	i.ID = uuid.NewString()
	i.Connection.Credentials = k.secrets(goodSecret)
	if _, _, err := c.ApplyCross(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &log}, importGraph(), i); !errors.Is(err, connectors.ErrNothingBuilt) {
		t.Fatalf("apply with missing values: %v", err)
	}
}
