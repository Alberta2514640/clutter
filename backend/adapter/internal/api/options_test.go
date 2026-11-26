package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
)

// listing is the mock with setting lists. It records the credentials it was
// handed, and answers with opts or err.
type listing struct {
	*mock.Connector
	mu   sync.Mutex
	seen connectors.Secrets
	opts connectors.Options
	err  error
}

func (l *listing) Options(_ context.Context, c connectors.Connection) (connectors.Options, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seen = c.Credentials
	return l.opts, l.err
}

func TestConnectionOptions(t *testing.T) {
	l := &listing{Connector: mock.New(), opts: connectors.Options{Settings: map[string][]connectors.Choice{
		// The mock's one setting is an enum of ubuntu-24.04 and debian-12.
		"image": {
			{Value: "ubuntu-24.04", Label: "Ubuntu 24.04"},
			{Value: "windows-11", Label: "not an allowed value"},
			{Value: "ubuntu-24.04", Label: "a repeat"},
			{Value: "debian-12", Label: "a label\nwith a new line"},
			{Value: "", Label: "empty"},
		},
		"flavor_small": {{Value: "m1.tiny", Label: "not a setting the manifest declares"}},
	}, Suggested: map[string]string{"image": "debian-12", "flavor_small": "m1.tiny"}}}
	e := newEnvWith(t, l)
	a := e.tenant(t, blocksDoc)
	b := e.tenant(t, blocksDoc)
	id := e.connect(t, &a, "lab")
	path := func(org, conn string) string { return "/v1/organizations/" + org + "/connections/" + conn + "/options" }

	code, body := e.call(t, &a, "GET", path(a.org, id), nil)
	if code != http.StatusOK {
		t.Fatalf("options: %d %s", code, body)
	}
	var got connectors.Options
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatal(err)
	}
	want := []connectors.Choice{{Value: "ubuntu-24.04", Label: "Ubuntu 24.04"}, {Value: "debian-12", Label: "debian-12"}}
	if len(got.Settings) != 1 || len(got.Settings["image"]) != 2 || got.Settings["image"][0] != want[0] || got.Settings["image"][1] != want[1] {
		t.Fatalf("options %s", body)
	}
	if len(got.Suggested) != 1 || got.Suggested["image"] != "debian-12" {
		t.Fatalf("suggested %v: only a declared setting's suggestion among its choices is kept", got.Suggested)
	}
	l.mu.Lock()
	seen := l.seen["token"]
	l.mu.Unlock()
	if seen != secretToken {
		t.Fatal("the lister was not handed the connection's credentials")
	}

	// Membership, then ownership of the connection.
	if code, _ := e.call(t, &b, "GET", path(a.org, id), nil); code != http.StatusForbidden {
		t.Fatalf("another organization's path: %d", code)
	}
	if code, _ := e.call(t, &b, "GET", path(b.org, id), nil); code != http.StatusNotFound {
		t.Fatalf("another organization's connection: %d", code)
	}
	if code, _ := e.call(t, &a, "GET", path(a.org, "not-a-uuid"), nil); code != http.StatusNotFound {
		t.Fatalf("a bad id: %d", code)
	}
	if code, _ := e.call(t, nil, "GET", path(a.org, id), nil); code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", code)
	}

	// The cloud's refusal is passed on as its user message.
	l.mu.Lock()
	l.err = &connectors.UserError{Message: "The cloud's compute (Nova) service did not answer in time."}
	l.mu.Unlock()
	code, failed := e.call(t, &a, "GET", path(a.org, id), nil)
	if code != http.StatusBadGateway || !strings.Contains(failed, "options_failed") || !strings.Contains(failed, "did not answer in time") {
		t.Fatalf("a refused list: %d %s", code, failed)
	}

	for _, out := range []string{body, failed} {
		if strings.Contains(out, secretToken) {
			t.Fatalf("a response holds the credential: %s", out)
		}
	}
	if strings.Contains(e.logBuf.String(), secretToken) {
		t.Fatal("the service log holds the credential")
	}
}

// TestCleanOptionsSuggestions: a suggestion is kept only for a declared
// setting, and only when it is one of the choices kept for that setting.
func TestCleanOptionsSuggestions(t *testing.T) {
	m := mock.New().Manifest() // one setting: "image", an enum of ubuntu-24.04 and debian-12
	cases := []struct {
		name    string
		choices []connectors.Choice
		suggest map[string]string
		want    string // "" means no suggestion
	}{
		{"among the choices", []connectors.Choice{{Value: "ubuntu-24.04"}, {Value: "debian-12"}}, map[string]string{"image": "debian-12"}, "debian-12"},
		{"allowed by the manifest but not offered", []connectors.Choice{{Value: "ubuntu-24.04"}}, map[string]string{"image": "debian-12"}, ""},
		{"offered but refused by CheckValue", []connectors.Choice{{Value: "ubuntu-24.04"}, {Value: "windows-11"}}, map[string]string{"image": "windows-11"}, ""},
		{"an undeclared setting", []connectors.Choice{{Value: "ubuntu-24.04"}}, map[string]string{"flavor_small": "ubuntu-24.04"}, ""},
		{"none", []connectors.Choice{{Value: "ubuntu-24.04"}}, nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanOptions(m, connectors.Options{Settings: map[string][]connectors.Choice{"image": tc.choices}, Suggested: tc.suggest})
			if tc.want == "" {
				if got.Suggested != nil {
					t.Fatalf("suggested %v", got.Suggested)
				}
				return
			}
			if len(got.Suggested) != 1 || got.Suggested["image"] != tc.want {
				t.Fatalf("suggested %v, want image %q", got.Suggested, tc.want)
			}
		})
	}
	// No suggestion leaves the field out of the answer.
	raw, _ := json.Marshal(cleanOptions(m, connectors.Options{Settings: map[string][]connectors.Choice{"image": {{Value: "debian-12"}}}}))
	if strings.Contains(string(raw), "suggested") {
		t.Fatalf("answer %s", raw)
	}
}

func TestConnectionOptionsWithoutALister(t *testing.T) {
	e := newEnv(t)
	tn := e.tenant(t, blocksDoc)
	id := e.connect(t, &tn, "lab")
	code, body := e.call(t, &tn, "GET", "/v1/organizations/"+tn.org+"/connections/"+id+"/options", nil)
	if code != http.StatusNotImplemented || !strings.Contains(body, "no_options") {
		t.Fatalf("options on the mock: %d %s", code, body)
	}
}
