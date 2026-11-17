package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// mockToken is the one token B's mock connector accepts (mock.ValidToken in
// backend/adapter/connectors/mock, a module this one does not import). It opens
// nothing real.
const mockToken = "mock-valid-token"

// blocksDiagram is a building-block diagram document, in the shape
// backend/adapter/internal/translate owns: a VM on a network, and a bucket the mock
// cloud cannot build, which the deployment leaves out.
func blocksDiagram() map[string]any {
	node := func(id, kind, name string, fields map[string]any) map[string]any {
		return map[string]any{"id": id, "type": kind, "position": map[string]any{"x": 0, "y": 0},
			"data": map[string]any{"name": name, "fields": fields}}
	}
	return map[string]any{
		"engine": "blocks",
		"nodes": []any{
			node("n1", "vm", "web", map[string]any{"size": "small"}),
			node("n2", "network", "net", map[string]any{}),
			node("n3", "bucket", "files", map[string]any{}),
		},
		"edges": []any{map[string]any{"id": "e1", "source": "n1", "target": "n2"}},
	}
}

// adapterSmoke checks B API's main paths end to end: health, sign-in, a mock
// connection, a building-block diagram saved through A's API, a deployment,
// plan, apply and destroy with logs, the delete guard, and refusals for a
// member of another organization. A's calls here go through r.do, so A's route
// table is unchanged. The results land in r.adapterChecks.
func adapterSmoke(r *runner, adapterAPI, conn, jwtSecret, sfx string) {
	add := func(name string, ok bool, note string) bool {
		r.adapterChecks = append(r.adapterChecks, check{name, ok, note})
		return ok
	}
	b := func(method, path string, body any, token string) resp {
		return r.doAt(adapterAPI, method, path, body, token)
	}
	status := func(rs resp) string { return fmt.Sprintf("%d %s", rs.status, summary(rs)) }

	hz := b("GET", "/healthz", nil, "")
	add("GET /healthz", hz.status == http.StatusOK, status(hz))

	anon := b("GET", "/v1/catalog", nil, "")
	add("GET /v1/catalog with no token", anon.status == http.StatusUnauthorized, status(anon))
	cat := b("GET", "/v1/catalog", nil, r.token)
	add("GET /v1/catalog with the smoke user's token", cat.status == http.StatusOK && strings.Contains(string(cat.body), `"connector":"mock"`),
		fmt.Sprintf("%d, mock listed: %v", cat.status, strings.Contains(string(cat.body), `"connector":"mock"`)))
	adapterCORS(r, adapterAPI, add)
	adapterExamples(b, r.token, add)

	// An organization, project and building-block diagram, through A's API.
	org := r.do("POST", "/organization", map[string]string{"organizationName": "Smoke B Org " + sfx, "description": "Created by the devserver smoke test for B"}, r.token)
	orgID := str(org.json, "data", "id")
	if !add("A: create an organization for B", orgID != "", status(org)) {
		return
	}
	defer func() {
		rs := r.do("DELETE", "/organization/"+orgID, nil, r.token)
		add("A: delete the organization (cascades to B's rows)", rs.status == http.StatusOK, status(rs))
	}()
	proj := r.do("POST", "/project", map[string]string{"organizationId": orgID, "name": "Smoke B Project", "description": "Created by the devserver smoke test for B"}, r.token)
	projID := str(proj.json, "data", "id")
	if !add("A: create a project", projID != "", status(proj)) {
		return
	}
	dia := r.do("POST", "/diagram", map[string]string{"projectId": projID, "name": "Smoke Blocks"}, r.token)
	diaID := str(dia.json, "data", "id")
	if !add("A: create a diagram", diaID != "", status(dia)) {
		return
	}
	saved := r.do("PUT", "/diagram", map[string]any{"projectId": projID, "diagramId": diaID, "name": "Smoke Blocks", "data": blocksDiagram()}, r.token)
	got := r.do("GET", "/diagram?projectId="+projID+"&diagramId="+diaID, nil, r.token)
	engine := savedEngine(got)
	add(`A's PUT /diagram keeps engine: "blocks"`, saved.status == http.StatusOK && engine == "blocks",
		fmt.Sprintf("save %d, read back engine %q", saved.status, engine))

	// A mock connection.
	connPath := "/v1/organizations/" + orgID + "/connections"
	badCred := "smoke-wrong-token-" + sfx
	bad := b("POST", connPath, map[string]any{"connector": "mock", "name": "smoke-bad", "region": "mock-1", "credentials": map[string]string{"token": badCred}}, r.token)
	add("POST connections with a wrong token is refused", bad.status == http.StatusUnprocessableEntity && !strings.Contains(string(bad.body), badCred), fmt.Sprintf("%d", bad.status))
	created := b("POST", connPath, map[string]any{"connector": "mock", "name": "smoke-mock", "region": "mock-1", "credentials": map[string]string{"token": mockToken}}, r.token)
	connID := str(created.json, "id")
	leaks := strings.Contains(string(created.body), mockToken) || strings.Contains(string(created.body), "secret_ref")
	if !add("POST connections (mock) verifies and creates", created.status == http.StatusCreated && connID != "" && str(created.json, "status") == "working" && !leaks,
		fmt.Sprintf("%d, status %q, credential in body: %v", created.status, str(created.json, "status"), leaks)) {
		return
	}
	connDeleted := false
	defer func() {
		if !connDeleted {
			b("DELETE", connPath+"/"+connID, nil, r.token)
		}
	}()
	ver := b("POST", connPath+"/"+connID+"/verify", nil, r.token)
	add("POST connections/{id}/verify", ver.status == http.StatusOK && str(ver.json, "status") == "working", fmt.Sprintf("%d, status %q", ver.status, str(ver.json, "status")))
	list := b("GET", connPath, nil, r.token)
	add("GET connections lists it", list.status == http.StatusOK && strings.Contains(string(list.body), connID), fmt.Sprintf("%d", list.status))

	// A deployment that leaves out the bucket, then plan and apply.
	depResp := b("POST", "/v1/organizations/"+orgID+"/diagrams/"+diaID+"/deployments", map[string]any{
		"connection_id": connID, "settings": map[string]any{"image": "debian-12"}, "exclusions": []string{"n3"},
	}, r.token)
	depID := str(depResp.json, "id")
	if !add("POST deployments for the blocks diagram", depResp.status == http.StatusCreated && depID != "", status(depResp)) {
		return
	}
	compat := b("GET", "/v1/organizations/"+orgID+"/deployments/"+depID+"/compatibility", nil, r.token)
	add("GET deployments/{id}/compatibility is ready", compat.status == http.StatusOK && compat.json["ready"] == true, status(compat))

	var applyRun string
	for _, action := range []string{"plan", "apply"} {
		runID, ok := adapterRun(r, adapterAPI, orgID, depID, action, add)
		if !ok {
			return
		}
		if action == "apply" {
			applyRun = runID
		}
	}

	// A member of another organization sees and uses none of it.
	second, err := newSecondUser(r, conn, jwtSecret, sfx+"-b")
	if !add("a second user with an organization of their own", err == nil, fmt.Sprint(err)) {
		return
	}
	adapterRefusals(b, orgID, second, connID, depID, diaID, applyRun, add)
	if err := second.remove(r, conn); err != nil {
		add("remove the second user", false, err.Error())
	}

	// A live deployment holds its connection; destroy releases it.
	held := b("DELETE", connPath+"/"+connID, nil, r.token)
	add("DELETE connection while the deployment is live is refused", held.status == http.StatusConflict, status(held))
	if _, ok := adapterRun(r, adapterAPI, orgID, depID, "destroy", add); !ok {
		return
	}
	del := b("DELETE", connPath+"/"+connID, nil, r.token)
	connDeleted = del.status == http.StatusNoContent
	after := b("GET", connPath, nil, r.token)
	add("DELETE connection after destroy", connDeleted && !strings.Contains(string(after.body), connID), fmt.Sprintf("%d, still listed: %v", del.status, strings.Contains(string(after.body), connID)))
}

// savedEngine reads the engine marker from GET /diagram's answer for one diagram.
func savedEngine(rs resp) string {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(rs.body, &env)
	var one struct {
		Data struct {
			Engine string `json:"engine"`
		} `json:"data"`
	}
	if err := json.Unmarshal(env.Data, &one); err != nil {
		var many []json.RawMessage
		if json.Unmarshal(env.Data, &many) == nil && len(many) > 0 {
			_ = json.Unmarshal(many[0], &one)
		}
	}
	return one.Data.Engine
}

// adapterRun starts a run, polls it to an end state, and checks it completed with a
// log that names it and holds no credential.
func adapterRun(r *runner, adapterAPI, orgID, depID, action string, add func(string, bool, string) bool) (string, bool) {
	name := "run " + action + " completes with logs"
	start := r.doAt(adapterAPI, "POST", "/v1/organizations/"+orgID+"/deployments/"+depID+"/runs", map[string]string{"action": action}, r.token)
	runID := str(start.json, "id")
	if start.status != http.StatusAccepted || runID == "" {
		return "", add(name, false, fmt.Sprintf("start: %d %s", start.status, summary(start)))
	}
	seen := []string{str(start.json, "status")}
	runPath := "/v1/organizations/" + orgID + "/runs/" + runID
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		s := str(r.doAt(adapterAPI, "GET", runPath, nil, r.token).json, "status")
		if s != seen[len(seen)-1] {
			seen = append(seen, s)
		}
		if s == "completed" || s == "failed" {
			break
		}
	}
	logs := r.doAt(adapterAPI, "GET", runPath+"/logs", nil, r.token)
	ok := seen[0] == "queued" && seen[len(seen)-1] == "completed" &&
		logs.status == http.StatusOK && strings.Contains(string(logs.body), "Run "+runID) && !strings.Contains(string(logs.body), mockToken)
	return runID, add(name, ok, fmt.Sprintf("states %s, log %d bytes", strings.Join(seen, " > "), len(logs.body)))
}

// adapterRefusals checks that a member of another organization can neither see nor
// use the first organization's connection, deployment or run, through either
// organization's path.
func adapterRefusals(b func(string, string, any, string) resp, orgID string, second *secondUser, connID, depID, diaID, runID string, add func(string, bool, string) bool) {
	refused := func(rs resp) bool { return rs.status == http.StatusForbidden || rs.status == http.StatusNotFound }
	own := "/v1/organizations/" + second.orgID
	theirs := "/v1/organizations/" + orgID
	cases := []struct {
		name, method, path string
		body               any
	}{
		{"list the connections", "GET", theirs + "/connections", nil},
		{"verify the connection", "POST", own + "/connections/" + connID + "/verify", nil},
		{"delete the connection", "DELETE", own + "/connections/" + connID, nil},
		{"deploy the diagram on the connection", "POST", own + "/diagrams/" + diaID + "/deployments", map[string]any{"connection_id": connID}},
		{"deploy the diagram through its own organization", "POST", theirs + "/diagrams/" + diaID + "/deployments", map[string]any{"connection_id": connID}},
		{"read the compatibility", "GET", own + "/deployments/" + depID + "/compatibility", nil},
		{"start a run", "POST", own + "/deployments/" + depID + "/runs", map[string]string{"action": "plan"}},
		{"read the run", "GET", own + "/runs/" + runID, nil},
		{"read the run log", "GET", own + "/runs/" + runID + "/logs", nil},
	}
	for _, c := range cases {
		rs := b(c.method, c.path, c.body, second.token)
		leaks := strings.Contains(string(rs.body), connID) || strings.Contains(string(rs.body), orgID)
		add("another organization cannot "+c.name, refused(rs) && !leaks, fmt.Sprintf("%d, names the first org's ids: %v", rs.status, leaks))
	}
	ownList := b("GET", own+"/connections", nil, second.token)
	add("another organization's own list does not hold the connection", ownList.status == http.StatusOK && !strings.Contains(string(ownList.body), connID), fmt.Sprintf("%d", ownList.status))
}

// adapterCORS checks that B answers a browser preflight from the frontend's origin,
// and nothing for another origin.
func adapterCORS(r *runner, adapterAPI string, add func(string, bool, string) bool) {
	preflight := func(origin string) (int, string, string) {
		req, err := http.NewRequest("OPTIONS", adapterAPI+"/v1/catalog", nil)
		if err != nil {
			return 0, "", ""
		}
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
		res, err := r.client.Do(req)
		if err != nil {
			return 0, "", ""
		}
		res.Body.Close()
		return res.StatusCode, res.Header.Get("Access-Control-Allow-Origin"), res.Header.Get("Access-Control-Allow-Headers")
	}
	code, origin, headers := preflight("http://localhost:3000")
	add("OPTIONS /v1/catalog from the frontend (CORS preflight)", code == http.StatusNoContent && origin == "http://localhost:3000" && strings.Contains(headers, "Authorization"),
		fmt.Sprintf("%d, allow-origin %q, allow-headers %q", code, origin, headers))
	code, origin, _ = preflight("http://evil.example")
	add("OPTIONS /v1/catalog from another origin gets no CORS answer", code != http.StatusNoContent && origin == "", fmt.Sprintf("%d, allow-origin %q", code, origin))
}

// adapterExamples checks that B serves its built-in example diagrams to a
// signed-in caller only, each a building-block document.
func adapterExamples(b func(string, string, any, string) resp, token string, add func(string, bool, string) bool) {
	anon := b("GET", "/v1/examples", nil, "")
	add("GET /v1/examples with no token", anon.status == http.StatusUnauthorized, fmt.Sprintf("%d", anon.status))
	got := b("GET", "/v1/examples", nil, token)
	var body struct {
		Items []struct {
			ID       string `json:"id"`
			Clouds   []string
			Document struct {
				Engine string `json:"engine"`
				Nodes  []any  `json:"nodes"`
			} `json:"document"`
		} `json:"items"`
	}
	_ = json.Unmarshal(got.body, &body)
	var ids []string
	ok := got.status == http.StatusOK && len(body.Items) > 0
	for _, e := range body.Items {
		ids = append(ids, e.ID)
		ok = ok && e.Document.Engine == "blocks" && len(e.Document.Nodes) > 0 && len(e.Clouds) > 0
	}
	add("GET /v1/examples lists building-block examples", ok, fmt.Sprintf("%d, %s", got.status, strings.Join(ids, ", ")))
}

// reportAdapter prints B's checks after A's route table.
func reportAdapter(r *runner) {
	if len(r.adapterChecks) == 0 {
		return
	}
	fmt.Println("\nB API checks:")
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	pass := 0
	for _, c := range r.adapterChecks {
		v := "PASS"
		if c.ok {
			pass++
		} else {
			v = "FAIL"
			failed = true
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\n", v, c.name, c.note)
	}
	w.Flush()
	fmt.Printf("B checks: %d  PASS %d  FAIL %d\n", len(r.adapterChecks), pass, len(r.adapterChecks)-pass)
}
