package generalopenstack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

const fakeToken = "gAAAAA-options-token-0123456789"

// osCloud is a fake OpenStack: Keystone at /identity/v3, Nova at
// /compute/v2.1, Glance at /image and Neutron at /networking, all on one
// server that the catalog names. The services answer only with the token
// Keystone issued.
type osCloud struct {
	*httptest.Server
	mu   sync.Mutex
	hits map[string]int
	// glanceStatus, when set, is Glance's answer instead of the list.
	glanceStatus int
	// endpoint, when set, rewrites each catalog URL the fake hands out.
	endpoint func(service, url string) string
}

func newOSCloud(t *testing.T) *osCloud {
	o := &osCloud{hits: map[string]int{}}
	o.Server = httptest.NewServer(http.HandlerFunc(o.serve))
	t.Cleanup(o.Close)
	return o
}

func (o *osCloud) authURL() string { return o.URL + "/identity/v3" }

func (o *osCloud) secrets(secret string) connectors.Secrets {
	return connectors.Secrets{credAuthURL: o.authURL(), credID: goodCredID, credSecret: secret}
}

func (o *osCloud) count(path string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.hits[path]
}

func (o *osCloud) total() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	n := 0
	for _, v := range o.hits {
		n += v
	}
	return n
}

func (o *osCloud) serve(w http.ResponseWriter, r *http.Request) {
	o.mu.Lock()
	o.hits[r.URL.Path]++
	glanceStatus, rewrite := o.glanceStatus, o.endpoint
	o.mu.Unlock()

	if r.URL.Path == "/identity/v3/auth/tokens" {
		var req tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		ac := req.Auth.Identity.ApplicationCredential
		if ac.ID != goodCredID || ac.Secret != goodSecret {
			http.Error(w, `{"error": {"code": 401}}`, http.StatusUnauthorized)
			return
		}
		ep := func(service, path string) string {
			u := o.URL + path
			if rewrite != nil {
				u = rewrite(service, u)
			}
			return u
		}
		w.Header().Set("X-Subject-Token", fakeToken)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"token": {"catalog": [
		  {"type": "compute", "endpoints": [
		    {"interface": "internal", "region_id": %[1]q, "url": "http://169.254.169.254/compute"},
		    {"interface": "public", "region_id": %[1]q, "url": %[2]q},
		    {"interface": "public", "region_id": "RegionTwo", "url": "http://169.254.169.254/two"}]},
		  {"type": "image", "endpoints": [{"interface": "public", "region_id": %[1]q, "url": %[3]q}]},
		  {"type": "network", "endpoints": [{"interface": "public", "region_id": %[1]q, "url": %[4]q}]},
		  {"type": "identity", "endpoints": [{"interface": "public", "region_id": %[1]q, "url": %[5]q}]}
		]}}`, testRegion, ep(serviceCompute, "/compute/v2.1/"), ep(serviceImage, "/image"), ep(serviceNetwork, "/networking"), o.authURL())
		return
	}
	if r.Header.Get("X-Auth-Token") != fakeToken {
		http.Error(w, "no token", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/compute/v2.1/flavors/detail":
		fmt.Fprint(w, `{"flavors": [
		  {"id": "3", "name": "m1.medium", "vcpus": 2, "ram": 4096, "disk": 40},
		  {"id": "1", "name": "m1.tiny", "vcpus": 1, "ram": 512, "disk": 1},
		  {"id": "2", "name": "m1.small", "vcpus": 1, "ram": 2048, "disk": 20},
		  {"id": "9", "name": "bad\"name", "vcpus": 1, "ram": 64, "disk": 1}]}`)
	case "/image/v2/images":
		if glanceStatus != 0 {
			w.WriteHeader(glanceStatus)
			return
		}
		if r.URL.Query().Get("status") != "active" {
			http.Error(w, "the form lists active images only", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, `{"images": [
		  {"id": "a", "name": "ubuntu-24.04", "status": "active"},
		  {"id": "b", "name": "cirros-0.6.3-x86_64-disk", "status": "active"},
		  {"id": "c", "name": "ubuntu-24.04", "status": "active"},
		  {"id": "d", "name": "", "status": "active"},
		  {"id": "e", "name": "${bad}", "status": "active"}]}`)
	case "/networking/v2.0/networks":
		if r.URL.Query().Get("router:external") != "true" {
			http.Error(w, "the form lists external networks only", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, `{"networks": [{"id": "n", "name": "public", "status": "ACTIVE", "router:external": true}]}`)
	default:
		http.NotFound(w, r)
	}
}

func values(cs []connectors.Choice) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Value
	}
	return out
}

func conn(o *osCloud, secret string) connectors.Connection {
	return connectors.Connection{ID: "conn-1", ConnectorID: ID, Region: testRegion, Credentials: o.secrets(secret)}
}

func TestOptionsListsTheCloudsChoices(t *testing.T) {
	o := newOSCloud(t)
	c := newConnector(nil, AllowPrivateTargets())
	got, err := c.Options(context.Background(), conn(o, goodSecret))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"flavor_small":     "m1.tiny,m1.small,m1.medium",
		"flavor_medium":    "m1.tiny,m1.small,m1.medium",
		"flavor_large":     "m1.tiny,m1.small,m1.medium",
		"image":            "cirros-0.6.3-x86_64-disk,ubuntu-24.04",
		"external_network": "public",
	}
	if len(got.Settings) != len(want) {
		t.Fatalf("settings %v", got.Settings)
	}
	for name, w := range want {
		if v := strings.Join(values(got.Settings[name]), ","); v != w {
			t.Errorf("%s: got %s, want %s", name, v, w)
		}
	}
	if l := got.Settings["flavor_small"][0].Label; l != "m1.tiny (1 vCPU, 512 MB RAM, 1 GB disk)" {
		t.Errorf("flavor label %q", l)
	}
	if l := got.Settings["flavor_small"][2].Label; l != "m1.medium (2 vCPUs, 4 GB RAM, 40 GB disk)" {
		t.Errorf("flavor label %q", l)
	}
	// Every list comes from one token.
	if n := o.count("/identity/v3/auth/tokens"); n != 1 {
		t.Errorf("%d token requests", n)
	}
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), fakeToken) || strings.Contains(string(raw), goodSecret) {
		t.Fatal("the options hold the token or the secret")
	}
	// What the connector offers passes its own render check.
	for name, cs := range got.Settings {
		for _, ch := range cs {
			if !validCloudName(ch.Value) {
				t.Errorf("%s offers %q, which the renderer refuses", name, ch.Value)
			}
		}
	}
}

func TestOptionsAreCachedPerConnection(t *testing.T) {
	o := newOSCloud(t)
	c := newConnector(nil, AllowPrivateTargets())
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	c.options.now = func() time.Time { return now }

	first, err := c.Options(context.Background(), conn(o, goodSecret))
	if err != nil {
		t.Fatal(err)
	}
	hits := o.total()
	// A caller that changes its answer does not change the cache.
	first.Settings["image"][0].Value = "changed"

	now = now.Add(9 * time.Minute)
	again, err := c.Options(context.Background(), conn(o, goodSecret))
	if err != nil {
		t.Fatal(err)
	}
	if o.total() != hits {
		t.Fatalf("a second call within ten minutes asked the cloud again (%d, then %d requests)", hits, o.total())
	}
	if again.Settings["image"][0].Value != "cirros-0.6.3-x86_64-disk" {
		t.Fatalf("the cache was changed through a caller's copy: %v", again.Settings["image"])
	}

	// Another credential on the same connection id is not served the lists.
	if _, err := c.Options(context.Background(), conn(o, badSecret)); err == nil || !strings.Contains(err.Error(), "refused this application credential") {
		t.Fatalf("another secret: %v", err)
	}

	now = now.Add(2 * time.Minute)
	if _, err := c.Options(context.Background(), conn(o, goodSecret)); err != nil {
		t.Fatal(err)
	}
	if o.count("/identity/v3/auth/tokens") != 3 {
		t.Fatalf("after ten minutes the cloud was not asked again: %d token requests", o.count("/identity/v3/auth/tokens"))
	}
}

func TestOptionsReturnWhatTheCloudCouldList(t *testing.T) {
	o := newOSCloud(t)
	o.glanceStatus = http.StatusForbidden
	c := newConnector(nil, AllowPrivateTargets())
	got, err := c.Options(context.Background(), conn(o, goodSecret))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Settings["image"]; ok {
		t.Fatal("a refused list is present")
	}
	if len(got.Settings["flavor_small"]) != 3 || len(got.Settings["external_network"]) != 1 {
		t.Fatalf("settings %v", got.Settings)
	}
	// A partial answer is not cached, so the next call can complete it.
	before := o.count("/identity/v3/auth/tokens")
	if _, err := c.Options(context.Background(), conn(o, goodSecret)); err != nil {
		t.Fatal(err)
	}
	if o.count("/identity/v3/auth/tokens") != before+1 {
		t.Fatal("a partial answer was cached")
	}
}

func TestOptionsErrors(t *testing.T) {
	cases := []struct {
		name     string
		secret   string
		region   string
		endpoint func(service, url string) string
		want     string
	}{
		{name: "bad credential", secret: badSecret, region: testRegion, want: "refused this application credential"},
		{name: "region without the services", secret: goodSecret, region: "RegionNine", want: "no public compute (Nova) endpoint in region RegionNine"},
		{name: "services at the metadata address", secret: goodSecret, region: testRegion,
			endpoint: func(_, u string) string { return "http://169.254.169.254/x" },
			want:     "address Clutter does not connect to"},
		{name: "service address with a user name", secret: goodSecret, region: testRegion,
			endpoint: func(_, u string) string { return strings.Replace(u, "http://", "http://admin:pw@", 1) },
			want:     "It must be an https address"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := newOSCloud(t)
			o.endpoint = tc.endpoint
			c := newConnector(nil, AllowPrivateTargets())
			cn := conn(o, tc.secret)
			cn.Region = tc.region
			_, err := c.Options(context.Background(), cn)
			var ue *connectors.UserError
			if !errors.As(err, &ue) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
			for _, v := range append(cn.Credentials.Values(), fakeToken) {
				if strings.Contains(err.Error(), v) {
					t.Fatalf("error %q repeats a secret", err)
				}
			}
		})
	}
}

// TestServiceURLRules: without AllowPrivateTargets, a catalog endpoint must be
// https; it never carries a user name or a query; only a public endpoint in
// the connection's region is used.
func TestServiceURLRules(t *testing.T) {
	cat := func(iface, region, u string) []catalogService {
		return []catalogService{{Type: serviceCompute, Endpoints: []catalogEndpoint{{Interface: iface, RegionID: region, URL: u}}}}
	}
	cases := []struct {
		name      string
		catalog   []catalogService
		allowHTTP bool
		want      string // the URL, or a part of the error
	}{
		{"https", cat("public", testRegion, "https://nova.example.com:8774/v2.1/"), false, "https://nova.example.com:8774/v2.1"},
		{"http refused in the cloud", cat("public", testRegion, "http://nova.example.com/v2.1"), false, "must be an https address"},
		{"http allowed for DevStack", cat("public", testRegion, "http://10.0.0.5/compute/v2.1"), true, "http://10.0.0.5/compute/v2.1"},
		{"user name", cat("public", testRegion, "https://a:b@nova.example.com/v2.1"), false, "must be an https address"},
		{"query", cat("public", testRegion, "https://nova.example.com/v2.1?x=1"), false, "must be an https address"},
		{"another scheme", cat("public", testRegion, "file:///etc/passwd"), true, "must be an https address"},
		{"internal only", cat("internal", testRegion, "https://nova.example.com/v2.1"), false, "no public compute (Nova) endpoint"},
		{"another region", cat("public", "RegionTwo", "https://nova.example.com/v2.1"), false, "no public compute (Nova) endpoint"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := serviceURL(tc.catalog, serviceCompute, testRegion, tc.allowHTTP)
			if err != nil {
				if !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error %q, want %q", err, tc.want)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
	if withVersion("https://g.example.com/image", "v2") != "https://g.example.com/image/v2" ||
		withVersion("https://g.example.com/v2", "v2") != "https://g.example.com/v2" ||
		withVersion("https://n.example.com:9696/v2.0", "v2.0") != "https://n.example.com:9696/v2.0" {
		t.Fatal("withVersion")
	}
}

// TestOptionsRefusePrivateTargetsByDefault: in the cloud, the auth URL rules
// of VerifyConnection apply to the lists too.
func TestOptionsRefusePrivateTargetsByDefault(t *testing.T) {
	o := newOSCloud(t)
	_, err := newConnector(nil).Options(context.Background(), conn(o, goodSecret))
	if err == nil || !strings.Contains(err.Error(), "must be an https address") {
		t.Fatalf("got %v", err)
	}
}
