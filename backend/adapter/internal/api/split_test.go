package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/split"
)

// splitDoc is a VM that gets a public VM's address, and a network.
const splitDoc = `{"engine":"blocks","nodes":[
 {"id":"web","type":"vm","data":{"name":"web","fields":{"size":"small"}}},
 {"id":"db","type":"vm","data":{"name":"db","fields":{"size":"small","public":true}}},
 {"id":"net","type":"network","data":{"name":"net"}}],
 "edges":[{"id":"e1","source":"web","target":"db"},{"id":"e2","source":"web","target":"net"}]}`

func (e *env) createSplit(t *testing.T, tn *tenant, primary string, placements []map[string]any) (int, string) {
	t.Helper()
	return e.call(t, tn, "POST", "/v1/organizations/"+tn.org+"/diagrams/"+tn.diagram+"/deployments", map[string]any{
		"connection_id": primary, "settings": map[string]any{"image": "debian-12"}, "placements": placements})
}

func placement(conn string, nodes ...string) map[string]any {
	return map[string]any{"connection_id": conn, "node_ids": nodes, "settings": map[string]any{"image": "ubuntu-24.04"}}
}

func TestAPlacementIsScopedToTheOrganization(t *testing.T) {
	e := newEnv(t)
	a, b := e.tenant(t, splitDoc), e.tenant(t, splitDoc)
	primary, mine, theirs := e.connect(t, &a, "main"), e.connect(t, &a, "lab"), e.connect(t, &b, "theirs")

	// Another organization's connection is not found, and nothing is made.
	if code, body := e.createSplit(t, &a, primary, []map[string]any{placement(theirs, "db")}); code != http.StatusNotFound {
		t.Fatalf("another organization's connection: %d %s", code, body)
	}
	_, list := e.call(t, &a, "GET", "/v1/organizations/"+a.org+"/diagrams/"+a.diagram+"/deployments", nil)
	if !strings.Contains(list, `"items":[]`) {
		t.Fatalf("a refused deployment was made: %s", list)
	}

	for name, ps := range map[string][]map[string]any{
		"the primary itself":   {placement(primary, "db")},
		"a block not in it":    {placement(mine, "gone")},
		"no block":             {placement(mine)},
		"a block on two":       {placement(mine, "db"), {"connection_id": theirs, "node_ids": []string{"db"}}},
		"a setting it ignores": {{"connection_id": mine, "node_ids": []string{"db"}, "settings": map[string]any{"flavor": "x"}}},
	} {
		if code, body := e.createSplit(t, &a, primary, ps); code != http.StatusUnprocessableEntity {
			t.Errorf("%s: %d %s", name, code, body)
		}
	}

	code, body := e.createSplit(t, &a, primary, []map[string]any{placement(mine, "db")})
	if code != http.StatusCreated {
		t.Fatalf("create: %d %s", code, body)
	}
	var d struct {
		Placements []placementView `json:"placements"`
	}
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Placements) != 1 || d.Placements[0].ConnectionID != mine || d.Placements[0].ConnectionName != "lab" ||
		d.Placements[0].NodeIDs[0] != "db" || d.Placements[0].Settings["image"] != "ubuntu-24.04" {
		t.Fatalf("placements %+v", d.Placements)
	}
	if strings.Contains(body, "secret_ref") || strings.Contains(body, secretToken) {
		t.Fatal("a deployment answer holds a secret")
	}

	// A PATCH cannot move a block onto another organization's connection.
	dep := field(t, body, "id")
	if code, body := e.call(t, &a, "PATCH", "/v1/organizations/"+a.org+"/deployments/"+dep,
		map[string]any{"placements": []map[string]any{placement(theirs, "db")}}); code != http.StatusNotFound {
		t.Fatalf("patch onto another organization's connection: %d %s", code, body)
	}
	// Nor can the other organization see the deployment.
	if code, _ := e.call(t, &b, "GET", "/v1/organizations/"+b.org+"/deployments/"+dep+"/compatibility", nil); code != http.StatusNotFound {
		t.Fatalf("the other organization reads the split: %d", code)
	}
}

func TestASplitDeploymentRunsOnBothClouds(t *testing.T) {
	m := mock.New()
	e := newEnvWith(t, m)
	a := e.tenant(t, splitDoc)
	primary, lab := e.connect(t, &a, "main"), e.connect(t, &a, "lab")
	code, body := e.createSplit(t, &a, primary, []map[string]any{placement(lab, "db")})
	if code != http.StatusCreated {
		t.Fatalf("create: %d %s", code, body)
	}
	dep := field(t, body, "id")

	_, body = e.call(t, &a, "GET", "/v1/organizations/"+a.org+"/deployments/"+dep+"/compatibility", nil)
	var rep compatibilityView
	if err := json.Unmarshal([]byte(body), &rep); err != nil {
		t.Fatal(err)
	}
	if !rep.Ready || len(rep.Clouds) != 2 || rep.Clouds[0].ConnectionID != lab || rep.Clouds[0].Position != 0 || rep.Clouds[1].ConnectionID != primary {
		t.Fatalf("compatibility %s", body)
	}
	hasNote := false
	for _, n := range rep.Notes {
		hasNote = hasNote || n.Code == split.CodeCrossAddress
		if n.Code == "edge_dropped" {
			t.Fatalf("a cross-cloud edge is noted as dropped: %s", body)
		}
	}
	if !hasNote {
		t.Fatalf("no note for the address edge: %s", body)
	}

	runID, seen := e.runAndWait(t, &a, dep, "apply")
	if seen[len(seen)-1] != "completed" {
		_, log := e.call(t, &a, "GET", "/v1/organizations/"+a.org+"/runs/"+runID+"/logs", nil)
		t.Fatalf("apply: %v\n%s", seen, log)
	}
	_, body = e.call(t, &a, "GET", "/v1/organizations/"+a.org+"/runs/"+runID, nil)
	var run runView
	if err := json.Unmarshal([]byte(body), &run); err != nil {
		t.Fatal(err)
	}
	if len(run.Parts) != 2 || run.Parts[0].ConnectionID != lab || run.Parts[0].Status != "completed" || run.Parts[1].Status != "completed" {
		t.Fatalf("parts %s", body)
	}
	// The primary's VM got the placed VM's address.
	imported := m.Imported(dep)
	got := imported[connectors.CrossKey("web", "db")][connectors.ValueAddress].Text
	if !strings.HasPrefix(got, "203.0.113.") {
		t.Fatalf("imported %v", imported)
	}
	// The placed cloud has its own state: the db VM is there, not on the primary.
	if _, ok := m.Built(split.PartID(dep, lab, false))["db"]; !ok {
		t.Fatal("the placed cloud did not build its block")
	}
	if _, ok := m.Built(dep)["db"]; ok {
		t.Fatal("the primary built a block placed elsewhere")
	}

	// While it holds resources, its blocks stay where they are and the
	// placed connection cannot go.
	if code, body := e.call(t, &a, "PATCH", "/v1/organizations/"+a.org+"/deployments/"+dep,
		map[string]any{"placements": []map[string]any{}}); code != http.StatusConflict || !strings.Contains(body, "deployment_live") {
		t.Fatalf("moving blocks of a live deployment: %d %s", code, body)
	}
	if code, body := e.call(t, &a, "DELETE", "/v1/organizations/"+a.org+"/connections/"+lab, nil); code != http.StatusConflict {
		t.Fatalf("deleting a placed connection of a live deployment: %d %s", code, body)
	}
	// A placed cloud's settings may change while live; its blocks stay.
	code, body = e.call(t, &a, "PATCH", "/v1/organizations/"+a.org+"/deployments/"+dep,
		map[string]any{"placements": []map[string]any{{"connection_id": lab, "node_ids": []string{"db"}, "settings": map[string]any{"image": "debian-12"}}}})
	if code != http.StatusOK || !strings.Contains(body, `"image":"debian-12"`) {
		t.Fatalf("changing a placed cloud's settings while live: %d %s", code, body)
	}

	if _, seen := e.runAndWait(t, &a, dep, "destroy"); seen[len(seen)-1] != "completed" {
		t.Fatalf("destroy: %v", seen)
	}
	if len(m.Built(dep)) != 0 || len(m.Built(split.PartID(dep, lab, false))) != 0 {
		t.Fatal("destroy left something on a cloud")
	}
	// Destroyed, the blocks may move again.
	if code, body := e.call(t, &a, "PATCH", "/v1/organizations/"+a.org+"/deployments/"+dep,
		map[string]any{"placements": []map[string]any{}}); code != http.StatusOK || !strings.Contains(body, `"placements":[]`) {
		t.Fatalf("moving blocks back: %d %s", code, body)
	}
}

func TestAnEdgeThatCannotCrossCloudsStopsTheRun(t *testing.T) {
	e := newEnv(t)
	a := e.tenant(t, splitDoc)
	primary, lab := e.connect(t, &a, "main"), e.connect(t, &a, "lab")
	code, body := e.createSplit(t, &a, primary, []map[string]any{placement(lab, "db", "net")})
	if code != http.StatusCreated {
		t.Fatalf("create: %d %s", code, body)
	}
	dep := field(t, body, "id")
	_, body = e.call(t, &a, "GET", "/v1/organizations/"+a.org+"/deployments/"+dep+"/compatibility", nil)
	if !strings.Contains(body, `"ready":false`) || !strings.Contains(body, split.CodeCrossEdge) {
		t.Fatalf("a VM and its network on two clouds: %s", body)
	}
	if code, body := e.call(t, &a, "POST", "/v1/organizations/"+a.org+"/deployments/"+dep+"/runs", map[string]string{"action": "apply"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("apply: %d %s", code, body)
	}
}
