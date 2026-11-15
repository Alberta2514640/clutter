package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/google/uuid"
)

func TestListADiagramsDeployments(t *testing.T) {
	e := newEnv(t)
	a := e.tenant(t, blocksDoc)
	b := e.tenant(t, blocksDoc)

	path := "/v1/organizations/" + a.org + "/diagrams/" + a.diagram + "/deployments"
	code, body := e.call(t, &a, "GET", path, nil)
	if code != http.StatusOK || strings.TrimSpace(body) != `{"items":[]}` {
		t.Fatalf("no deployments yet: %d %s", code, body)
	}

	connID := e.connect(t, &a, "lab")
	depID := e.deploy(t, &a, connID, []string{"n3"})
	runID, _ := e.runAndWait(t, &a, depID, "plan")

	code, body = e.call(t, &a, "GET", path, nil)
	if code != http.StatusOK {
		t.Fatalf("list: %d %s", code, body)
	}
	var out struct {
		Items []struct {
			ID             string   `json:"id"`
			DiagramID      string   `json:"diagram_id"`
			ConnectionID   string   `json:"connection_id"`
			Connector      string   `json:"connector"`
			ConnectionName string   `json:"connection_name"`
			Region         string   `json:"region"`
			Exclusions     []string `json:"exclusions"`
			LatestRun      *struct {
				ID     string `json:"id"`
				Action string `json:"action"`
				Status string `json:"status"`
			} `json:"latest_run"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 1 {
		t.Fatalf("want one deployment: %s", body)
	}
	got := out.Items[0]
	if got.ID != depID || got.DiagramID != a.diagram || got.ConnectionID != connID || got.Connector != "mock" ||
		got.ConnectionName != "lab" || got.Region != "mock-1" || len(got.Exclusions) != 1 || got.Exclusions[0] != "n3" {
		t.Fatalf("deployment row: %+v", got)
	}
	if got.LatestRun == nil || got.LatestRun.ID != runID || got.LatestRun.Action != "plan" || got.LatestRun.Status != "completed" {
		t.Fatalf("latest run: %+v", got.LatestRun)
	}
	if strings.Contains(body, secretToken) || strings.Contains(body, "secret_ref") || strings.Contains(body, "local:") {
		t.Fatalf("the list carries a credential or its reference: %s", body)
	}

	// Another organization sees nothing of A's, through its own org or A's.
	refused := func(code int) bool { return code == http.StatusForbidden || code == http.StatusNotFound }
	for _, p := range []string{
		"/v1/organizations/" + b.org + "/diagrams/" + a.diagram + "/deployments",
		"/v1/organizations/" + a.org + "/diagrams/" + a.diagram + "/deployments",
		"/v1/organizations/" + b.org + "/diagrams/not-a-uuid/deployments",
	} {
		code, body := e.call(t, &b, "GET", p, nil)
		if !refused(code) || strings.Contains(body, depID) || strings.Contains(body, a.org) {
			t.Errorf("%s: %d %s", p, code, body)
		}
	}

	// A deployment of A's diagram on B's connection, written behind the API's
	// back, is listed for neither.
	connB := e.connect(t, &b, "b-cloud")
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO deployments (id, diagram_id, connection_id) VALUES ($1, $2, $3)`,
		uuid.NewString(), a.diagram, connB); err != nil {
		t.Fatal(err)
	}
	_, body = e.call(t, &a, "GET", path, nil)
	if strings.Contains(body, connB) {
		t.Fatalf("a mixed deployment is listed: %s", body)
	}
}

// A new deployment's settings are checked as a PATCH checks them, against
// the manifest of its own connection's connector.
func TestCreateADeploymentChecksItsSettings(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)
	connID := e.connect(t, &tn, "lab")
	path := "/v1/organizations/" + tn.org + "/diagrams/" + tn.diagram + "/deployments"

	tooMany := map[string]any{}
	for i := range maxSettings + 1 {
		tooMany[fmt.Sprintf("s%d", i)] = "x"
	}
	refusals := []struct {
		name     string
		settings map[string]any
		want     string
	}{
		{"a setting the cloud does not use", map[string]any{"image": "debian-12", "flavor": "big"}, `"field":"flavor"`},
		{"a value outside the allowed list", map[string]any{"image": "windows"}, `"field":"image"`},
		{"too many settings", tooMany, "too many settings"},
	}
	for _, c := range refusals {
		code, body := e.call(t, &tn, "POST", path, map[string]any{"connection_id": connID, "settings": c.settings})
		if code != http.StatusUnprocessableEntity || !strings.Contains(body, `"code":"invalid_settings"`) || !strings.Contains(body, c.want) {
			t.Errorf("%s: %d %s", c.name, code, body)
		}
		if strings.Contains(body, "windows") || strings.Contains(body, "big") {
			t.Errorf("%s: the refusal echoes a value: %s", c.name, body)
		}
	}
	if code, body := e.call(t, &tn, "GET", path, nil); code != http.StatusOK || strings.TrimSpace(body) != `{"items":[]}` {
		t.Fatalf("a refused deployment was created: %d %s", code, body)
	}

	// A null value is left out, as a PATCH unsets it.
	code, body := e.call(t, &tn, "POST", path, map[string]any{"connection_id": connID, "settings": map[string]any{"image": nil}})
	if code != http.StatusCreated || !strings.Contains(body, `"settings":{}`) {
		t.Fatalf("create with a null setting: %d %s", code, body)
	}
	// A connection of no organization of the caller's is not found, before
	// its settings are read.
	code, _ = e.call(t, &tn, "POST", path, map[string]any{"connection_id": uuid.NewString(), "settings": map[string]any{"flavor": "big"}})
	if code != http.StatusNotFound {
		t.Fatalf("an unknown connection: %d", code)
	}
}

func TestUpdateADeployment(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)
	other := e.tenant(t, blocksDoc)
	connID := e.connect(t, &tn, "lab")
	depID := e.deploy(t, &tn, connID, nil)
	path := "/v1/organizations/" + tn.org + "/deployments/" + depID
	compat := path + "/compatibility"

	if _, body := e.call(t, &tn, "GET", compat, nil); !strings.Contains(body, `"ready":false`) {
		t.Fatalf("the bucket should make it not ready: %s", body)
	}
	code, body := e.call(t, &tn, "PATCH", path, map[string]any{"exclusions": []string{"n3"}})
	if code != http.StatusOK || !strings.Contains(body, `"exclusions":["n3"]`) || !strings.Contains(body, `"image":"debian-12"`) {
		t.Fatalf("exclude the bucket: %d %s", code, body)
	}
	if _, body := e.call(t, &tn, "GET", compat, nil); !strings.Contains(body, `"ready":true`) {
		t.Fatalf("after leaving the bucket out: %s", body)
	}
	code, body = e.call(t, &tn, "PATCH", path, map[string]any{"settings": map[string]any{"image": "ubuntu-24.04"}})
	if code != http.StatusOK || !strings.Contains(body, `"image":"ubuntu-24.04"`) || !strings.Contains(body, `"exclusions":["n3"]`) {
		t.Fatalf("change a setting: %d %s", code, body)
	}

	refusals := []struct {
		name string
		body any
		code int
		want string
	}{
		{"a value outside the allowed list", map[string]any{"settings": map[string]any{"image": "windows"}}, http.StatusUnprocessableEntity, "invalid_settings"},
		{"a setting the cloud does not use", map[string]any{"settings": map[string]any{"flavor": "big"}}, http.StatusUnprocessableEntity, `"field":"flavor"`},
		{"a block not in the diagram", map[string]any{"exclusions": []string{"n9"}}, http.StatusUnprocessableEntity, "unknown_exclusion"},
		{"a block id that is not valid", map[string]any{"exclusions": []string{"bad id!"}}, http.StatusUnprocessableEntity, "invalid_exclusions"},
		{"an unknown body field", map[string]any{"connection_id": connID}, http.StatusBadRequest, "bad_request"},
	}
	for _, c := range refusals {
		code, body := e.call(t, &tn, "PATCH", path, c.body)
		if code != c.code || !strings.Contains(body, c.want) {
			t.Errorf("%s: %d %s", c.name, code, body)
		}
		if strings.Contains(body, "windows") {
			t.Errorf("%s: the refusal echoes the value: %s", c.name, body)
		}
	}
	// A null value unsets a setting.
	if code, body := e.call(t, &tn, "PATCH", path, map[string]any{"settings": map[string]any{"image": nil}}); code != http.StatusOK || strings.Contains(body, "ubuntu") {
		t.Fatalf("unset a setting: %d %s", code, body)
	}

	// Another organization cannot change it, through its own org or this one.
	for _, p := range []string{"/v1/organizations/" + other.org + "/deployments/" + depID, path} {
		if code, _ := e.call(t, &other, "PATCH", p, map[string]any{"exclusions": []string{}}); code != http.StatusForbidden && code != http.StatusNotFound {
			t.Errorf("%s by another organization: %d", p, code)
		}
	}

	// Nothing changes while a run is queued or running.
	runID := uuid.NewString()
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO runs (id, deployment_id, action, log_key) VALUES ($1, $2, 'apply', $3)`,
		runID, depID, logstore.KeyForRun(runID)); err != nil {
		t.Fatal(err)
	}
	if code, body := e.call(t, &tn, "PATCH", path, map[string]any{"exclusions": []string{}}); code != http.StatusConflict || !strings.Contains(body, "run_active") {
		t.Fatalf("change during a run: %d %s", code, body)
	}
}

func TestRemoveADeployment(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)
	other := e.tenant(t, blocksDoc)
	connID := e.connect(t, &tn, "lab")
	list := "/v1/organizations/" + tn.org + "/diagrams/" + tn.diagram + "/deployments"

	// A new deployment holds nothing, so it can go.
	fresh := e.deploy(t, &tn, connID, []string{"n3"})
	if code, body := e.call(t, &tn, "DELETE", "/v1/organizations/"+tn.org+"/deployments/"+fresh, nil); code != http.StatusNoContent {
		t.Fatalf("remove a new deployment: %d %s", code, body)
	}
	if _, body := e.call(t, &tn, "GET", list, nil); strings.TrimSpace(body) != `{"items":[]}` {
		t.Fatalf("the removed deployment is still listed: %s", body)
	}
	if code, _ := e.call(t, &tn, "DELETE", "/v1/organizations/"+tn.org+"/deployments/"+fresh, nil); code != http.StatusNotFound {
		t.Fatalf("remove it twice: %d", code)
	}

	// The connection is free again: the same diagram deploys on it once more.
	depID := e.deploy(t, &tn, connID, []string{"n3"})
	path := "/v1/organizations/" + tn.org + "/deployments/" + depID
	runID, _ := e.runAndWait(t, &tn, depID, "apply")

	// A live deployment may hold resources: refused until it is destroyed.
	if code, body := e.call(t, &tn, "DELETE", path, nil); code != http.StatusConflict || !strings.Contains(body, "deployment_live") {
		t.Fatalf("remove a live deployment: %d %s", code, body)
	}

	// Another organization cannot remove it, through its own org or this one.
	for _, p := range []string{"/v1/organizations/" + other.org + "/deployments/" + depID, path,
		"/v1/organizations/" + other.org + "/deployments/not-a-uuid"} {
		code, body := e.call(t, &other, "DELETE", p, nil)
		if code != http.StatusForbidden && code != http.StatusNotFound {
			t.Errorf("%s by another organization: %d", p, code)
		}
		if strings.Contains(body, tn.org) || strings.Contains(body, depID) {
			t.Errorf("%s: the refusal names the owner's ids: %s", p, body)
		}
	}

	e.runAndWait(t, &tn, depID, "destroy")

	// A destroyed deployment with a run queued or running is still refused.
	queued := uuid.NewString()
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO runs (id, deployment_id, action, log_key) VALUES ($1, $2, 'plan', $3)`,
		queued, depID, logstore.KeyForRun(queued)); err != nil {
		t.Fatal(err)
	}
	if code, body := e.call(t, &tn, "DELETE", path, nil); code != http.StatusConflict || !strings.Contains(body, "run_active") {
		t.Fatalf("remove during a run: %d %s", code, body)
	}
	if _, err := e.pool.Exec(context.Background(), `UPDATE runs SET status = 'completed', finished_at = now() WHERE id = $1`, queued); err != nil {
		t.Fatal(err)
	}

	if code, body := e.call(t, &tn, "DELETE", path, nil); code != http.StatusNoContent {
		t.Fatalf("remove a destroyed deployment: %d %s", code, body)
	}
	// Its runs go with it.
	if code, _ := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/runs/"+runID, nil); code != http.StatusNotFound {
		t.Fatalf("a removed deployment's run: %d", code)
	}
	again := e.deploy(t, &tn, connID, nil)
	if again == depID || again == "" {
		t.Fatalf("a new deployment on the freed connection: %q", again)
	}
}

// previewing is the mock with a preview that writes out every credential
// value it is handed, so a credential that reached it would show in the files.
type previewing struct {
	*mock.Connector
}

func (p previewing) Preview(_ context.Context, g connectors.Graph, d connectors.Deployment) (map[string][]byte, error) {
	var b strings.Builder
	b.WriteString("# deployment " + d.ID + " prefix " + d.Prefix + "\n")
	ids := make([]string, 0, len(g.Blocks))
	for id := range g.Blocks {
		if !d.Excluded(id) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		b.WriteString("# block " + id + "\n")
	}
	for k, v := range d.Connection.Credentials {
		b.WriteString("# credential " + k + "=" + v + "\n")
	}
	return map[string][]byte{"main.tf": []byte(b.String())}, nil
}

func TestPreviewRendersWithoutCredentials(t *testing.T) {
	e := newEnvWith(t, previewing{mock.New()})
	tn := e.tenant(t, blocksDoc)
	other := e.tenant(t, blocksDoc)
	connID := e.connect(t, &tn, "lab")

	notReady := e.deploy(t, &tn, connID, nil)
	code, body := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/deployments/"+notReady+"/preview", nil)
	if code != http.StatusUnprocessableEntity || !strings.Contains(body, "not_ready") || !strings.Contains(body, `"node_id":"n3"`) {
		t.Fatalf("preview while not ready: %d %s", code, body)
	}
	if _, err := e.pool.Exec(context.Background(), `DELETE FROM deployments WHERE id = $1`, notReady); err != nil {
		t.Fatal(err)
	}

	depID := e.deploy(t, &tn, connID, []string{"n3"})
	path := "/v1/organizations/" + tn.org + "/deployments/" + depID + "/preview"
	code, body = e.call(t, &tn, "GET", path, nil)
	if code != http.StatusOK {
		t.Fatalf("preview: %d %s", code, body)
	}
	var out struct {
		Files map[string]string `json:"files"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatal(err)
	}
	main := out.Files["main.tf"]
	if !strings.Contains(main, "# deployment "+depID) || !strings.Contains(main, "# block n1") || strings.Contains(main, "# block n3") {
		t.Fatalf("preview files: %q", main)
	}
	if strings.Contains(body, secretToken) || strings.Contains(body, "# credential") {
		t.Fatalf("the preview carries a credential: %s", body)
	}

	if code, _ := e.call(t, &other, "GET", "/v1/organizations/"+other.org+"/deployments/"+depID+"/preview", nil); code != http.StatusNotFound {
		t.Errorf("preview through another organization: %d", code)
	}
}

func TestPreviewIsNotImplementedWithoutAPreviewer(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)
	connID := e.connect(t, &tn, "lab")
	depID := e.deploy(t, &tn, connID, []string{"n3"})
	code, body := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/deployments/"+depID+"/preview", nil)
	if code != http.StatusNotImplemented || !strings.Contains(body, "no_preview") {
		t.Fatalf("preview on the mock: %d %s", code, body)
	}
}
