package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
)

// linking is the mock with accounts linked outside B, per organization. Its
// one secret is the mock's valid token, so a leak would show in a response.
type linking struct {
	*mock.Connector
	byOrg map[string][]connectors.LinkedAccount
}

func (l linking) Linked(_ context.Context, orgID string) ([]connectors.LinkedAccount, error) {
	return l.byOrg[orgID], nil
}

// linkedTemplateVersion is the template version the mock's links report.
const linkedTemplateVersion = "7"

func (l linking) LinkedSecrets(_ context.Context, orgID, linkedID string) (connectors.LinkedCredentials, error) {
	for _, acc := range l.byOrg[orgID] {
		if acc.ID == linkedID {
			return connectors.LinkedCredentials{Secrets: connectors.Secrets{"token": secretToken}, TemplateVersion: linkedTemplateVersion}, nil
		}
	}
	return connectors.LinkedCredentials{}, connectors.ErrLinkedNotFound
}

// linkedOnly is linking with a credential spec that takes no pasted
// credentials, as the AWS connector's.
type linkedOnly struct{ linking }

func (l linkedOnly) Credentials() connectors.CredentialSpec {
	s := l.linking.Credentials()
	s.LinkedOnly = true
	return s
}

// TestALinkedOnlyCloudTakesNoPastedCredentials: a connection to a linked-only
// cloud (AWS) comes only from a link of the caller's own organization. A
// pasted secret is refused even when it works, so a former member who kept a
// stack link's external id cannot connect that account to another
// organization; another organization's link is not found; and the secret
// never reaches a response.
func TestALinkedOnlyCloudTakesNoPastedCredentials(t *testing.T) {
	c := linkedOnly{linking{Connector: mock.New(), byOrg: map[string][]connectors.LinkedAccount{}}}
	e := newEnvWith(t, c)
	a := e.tenant(t, blocksDoc)
	b := e.tenant(t, blocksDoc)
	c.byOrg[a.org] = []connectors.LinkedAccount{{ID: "acct-a", Name: "prod"}}
	c.byOrg[b.org] = []connectors.LinkedAccount{{ID: "acct-b", Name: "theirs"}}

	var bodies []string
	call := func(tn *tenant, method, path string, body any) (int, string) {
		code, out := e.call(t, tn, method, path, body)
		bodies = append(bodies, out)
		return code, out
	}
	// The catalog is left out of the sweep below: the mock's help text names
	// its one valid token in public.
	if _, body := e.call(t, &a, "GET", "/v1/catalog", nil); !strings.Contains(body, `"linked_only":true`) {
		t.Fatalf("catalog: %s", body)
	}
	create := func(tn *tenant, name string, extra map[string]any) (int, string) {
		body := map[string]any{"connector": "mock", "name": name, "region": "mock-1"}
		for k, v := range extra {
			body[k] = v
		}
		return call(tn, "POST", "/v1/organizations/"+tn.org+"/connections", body)
	}
	// The pasted secret works, so only the linked-only rule refuses it.
	if code, body := create(&a, "pasted", map[string]any{"credentials": map[string]string{"token": secretToken}}); code != http.StatusBadRequest || !strings.Contains(body, "linked_only") {
		t.Fatalf("pasted credentials: %d %s", code, body)
	}
	if code, body := create(&a, "both", map[string]any{"linked_id": "acct-a", "credentials": map[string]string{"token": secretToken}}); code != http.StatusBadRequest || !strings.Contains(body, "linked_only") {
		t.Fatalf("a linked account with pasted credentials: %d %s", code, body)
	}
	if code, body := create(&a, "nothing", nil); code != http.StatusBadRequest {
		t.Fatalf("no credentials and no linked account: %d %s", code, body)
	}
	if code, _ := create(&a, "stolen", map[string]any{"linked_id": "acct-b"}); code != http.StatusNotFound {
		t.Fatalf("another organization's linked account: %d", code)
	}
	code, body := create(&a, "from-link", map[string]any{"linked_id": "acct-a"})
	if code != http.StatusCreated {
		t.Fatalf("connection from a linked account: %d %s", code, body)
	}
	id := field(t, body, "id")
	call(&a, "GET", "/v1/organizations/"+a.org+"/connections", nil)
	call(&a, "POST", "/v1/organizations/"+a.org+"/connections/"+id+"/verify", nil)
	call(&a, "GET", "/v1/organizations/"+a.org+"/connectors/mock/linked-accounts", nil)
	for _, b := range bodies {
		if strings.Contains(b, secretToken) {
			t.Fatalf("a response holds the linked secret: %s", b)
		}
	}
	if strings.Contains(e.logBuf.String(), secretToken) {
		t.Fatal("the service log holds the linked secret")
	}

	// The connection records the template version its link reported.
	var version *string
	if err := e.pool.QueryRow(context.Background(), `SELECT role_template_version FROM connections WHERE id = $1`, id).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version == nil || *version != linkedTemplateVersion {
		t.Fatalf("role template version %v", version)
	}
}

func TestConnectionFromALinkedAccount(t *testing.T) {
	c := linking{Connector: mock.New(), byOrg: map[string][]connectors.LinkedAccount{}}
	e := newEnvWith(t, c)
	a := e.tenant(t, blocksDoc)
	b := e.tenant(t, blocksDoc)
	c.byOrg[a.org] = []connectors.LinkedAccount{{ID: "acct-a", Name: "prod"}}
	c.byOrg[b.org] = []connectors.LinkedAccount{{ID: "acct-b", Name: "theirs"}}

	if _, body := e.call(t, &a, "GET", "/v1/catalog", nil); !strings.Contains(body, `"has_linked_accounts":true`) {
		t.Fatalf("catalog: %s", body)
	}
	code, body := e.call(t, &a, "GET", "/v1/organizations/"+a.org+"/connectors/mock/linked-accounts", nil)
	if code != http.StatusOK || !strings.Contains(body, `"name":"prod"`) || strings.Contains(body, "theirs") || strings.Contains(body, secretToken) {
		t.Fatalf("linked accounts: %d %s", code, body)
	}
	if code, _ := e.call(t, &b, "GET", "/v1/organizations/"+a.org+"/connectors/mock/linked-accounts", nil); code != http.StatusForbidden {
		t.Fatalf("another organization's linked accounts: %d", code)
	}
	if code, _ := e.call(t, &a, "GET", "/v1/organizations/"+a.org+"/connectors/nope/linked-accounts", nil); code != http.StatusNotFound {
		t.Fatalf("unknown connector: %d", code)
	}

	create := func(tn *tenant, org, name string, extra map[string]any) (int, string) {
		body := map[string]any{"connector": "mock", "name": name, "region": "mock-1"}
		for k, v := range extra {
			body[k] = v
		}
		return e.call(t, tn, "POST", "/v1/organizations/"+org+"/connections", body)
	}
	code, body = create(&a, a.org, "from-link", map[string]any{"linked_id": "acct-a"})
	if code != http.StatusCreated || field(t, body, "status") != "working" || strings.Contains(body, secretToken) {
		t.Fatalf("connection from a linked account: %d %s", code, body)
	}
	if strings.Contains(e.logBuf.String(), secretToken) {
		t.Fatal("the service log holds the linked secret")
	}
	if code, body := create(&a, a.org, "stolen", map[string]any{"linked_id": "acct-b"}); code != http.StatusNotFound || strings.Contains(body, "theirs") {
		t.Fatalf("another organization's linked account: %d %s", code, body)
	}
	if code, _ := create(&a, a.org, "both", map[string]any{"linked_id": "acct-a", "credentials": map[string]string{"token": "x"}}); code != http.StatusUnprocessableEntity {
		t.Fatalf("credentials and a linked account: %d", code)
	}
}

func TestLinkingIsRefusedWithoutALinker(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)
	if _, body := e.call(t, &tn, "GET", "/v1/catalog", nil); !strings.Contains(body, `"has_linked_accounts":false`) {
		t.Fatalf("catalog: %s", body)
	}
	if code, _ := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/connectors/mock/linked-accounts", nil); code != http.StatusNotImplemented {
		t.Fatalf("linked accounts on the mock: %d", code)
	}
	code, body := e.call(t, &tn, "POST", "/v1/organizations/"+tn.org+"/connections",
		map[string]any{"connector": "mock", "name": "x", "region": "mock-1", "linked_id": "acct"})
	if code != http.StatusUnprocessableEntity || !strings.Contains(body, "no_linking") {
		t.Fatalf("a linked id on the mock: %d %s", code, body)
	}
}
