package generalopenstack

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

func TestVerifyConnection(t *testing.T) {
	k := newKeystone(t)
	// The fake Keystone is on 127.0.0.1 over http, as a laptop's DevStack is.
	c := newConnector(nil, AllowPrivateTargets())
	c.verifyTimeout = 300 * time.Millisecond

	cases := []struct {
		name    string
		creds   connectors.Secrets
		region  string
		hang    bool
		wantErr string // empty means accepted
	}{
		{name: "201 accepts", creds: k.secrets(goodSecret), region: testRegion},
		{name: "a second catalog region is accepted", creds: k.secrets(goodSecret), region: "RegionTwo"},
		{name: "401 is a refused credential", creds: k.secrets(badSecret), region: testRegion, wantErr: "refused this application credential"},
		{name: "404 is a refused credential", creds: connectors.Secrets{credAuthURL: k.authURL(), credID: missingID, credSecret: goodSecret}, region: testRegion, wantErr: "refused this application credential"},
		{name: "timeout", creds: k.secrets(goodSecret), region: testRegion, hang: true, wantErr: "did not answer in time"},
		{name: "region not in the catalog", creds: k.secrets(goodSecret), region: "RegionNine", wantErr: "Its regions are: RegionOne, RegionTwo."},
		{name: "auth URL without /v3", creds: connectors.Secrets{credAuthURL: k.URL + "/identity", credID: goodCredID, credSecret: goodSecret}, region: testRegion, wantErr: "ends in /v3"},
		{name: "auth URL with a user name", creds: connectors.Secrets{credAuthURL: "http://admin:pw@" + strings.TrimPrefix(k.URL, "http://") + "/identity/v3", credID: goodCredID, credSecret: goodSecret}, region: testRegion, wantErr: "ends in /v3"},
		{name: "auth URL with another scheme", creds: connectors.Secrets{credAuthURL: "file:///etc/v3", credID: goodCredID, credSecret: goodSecret}, region: testRegion, wantErr: "ends in /v3"},
		{name: "metadata address", creds: connectors.Secrets{credAuthURL: "http://169.254.169.254/v3", credID: goodCredID, credSecret: goodSecret}, region: testRegion, wantErr: "link-local or metadata"},
		{name: "nothing listens", creds: connectors.Secrets{credAuthURL: "http://127.0.0.1:1/v3", credID: goodCredID, credSecret: goodSecret}, region: testRegion, wantErr: "could not connect"},
		{name: "missing secret", creds: connectors.Secrets{credAuthURL: k.authURL(), credID: goodCredID}, region: testRegion, wantErr: "is required"},
		{name: "region with a quote", creds: k.secrets(goodSecret), region: `RegionOne"`, wantErr: "region name is not valid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k.setHang(tc.hang)
			defer k.setHang(false)
			err := c.VerifyConnection(context.Background(), connectors.Connection{ID: "x", ConnectorID: ID, Region: tc.region, Credentials: tc.creds})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("refused: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("accepted")
			}
			var ue *connectors.UserError
			if !errors.As(err, &ue) {
				t.Fatalf("not a user error: %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not say %q", err, tc.wantErr)
			}
			for _, v := range tc.creds.Values() {
				if strings.Contains(err.Error(), v) {
					t.Fatalf("error %q repeats a credential value", err)
				}
			}
		})
	}
}

func TestVerifyHonoursCancelledContext(t *testing.T) {
	k := newKeystone(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := newConnector(nil, AllowPrivateTargets()).VerifyConnection(ctx, connectors.Connection{Region: testRegion, Credentials: k.secrets(goodSecret)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

// TestVerifyRefusesPrivateTargetsByDefault: without AllowPrivateTargets, as
// in the cloud, an auth URL must be https, and verification connects to no
// loopback, private, shared or link-local address, even through a host name.
func TestVerifyRefusesPrivateTargetsByDefault(t *testing.T) {
	c := newConnector(nil)
	c.verifyTimeout = 2 * time.Second
	creds := func(url string) connectors.Secrets {
		return connectors.Secrets{credAuthURL: url, credID: goodCredID, credSecret: goodSecret}
	}
	k := newKeystone(t)
	cases := map[string]struct{ url, want string }{
		"plain http":           {k.authURL(), "must be an https address"},
		"loopback":             {"https://127.0.0.1:5000/v3", "does not connect to"},
		"loopback by name":     {"https://localhost:5000/v3", "does not connect to"},
		"ten":                  {"https://10.0.0.5:5000/v3", "does not connect to"},
		"one seven two":        {"https://172.16.0.1/v3", "does not connect to"},
		"one nine two":         {"https://192.168.1.10/v3", "does not connect to"},
		"shared address space": {"https://100.64.0.1/v3", "does not connect to"},
		"metadata":             {"https://169.254.169.254/v3", "does not connect to"},
		"IPv6 loopback":        {"https://[::1]:5000/v3", "does not connect to"},
		"IPv6 unique local":    {"https://[fd12:3456::1]/v3", "does not connect to"},
		"IPv4 mapped in IPv6":  {"https://[::ffff:10.0.0.5]/v3", "does not connect to"},
		"this network":         {"https://0.0.0.1/v3", "does not connect to"},
		"IPv6 metadata":        {"https://[fd00:ec2::254]/v3", "does not connect to"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := c.VerifyConnection(context.Background(), connectors.Connection{Region: testRegion, Credentials: creds(tc.url)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

// TestVerifyNeverEchoesTheStatus: an answer that is not Keystone's gets a
// fixed message, so a user cannot learn what answers at an address.
func TestVerifyNeverEchoesTheStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(srv.Close)
	c := newConnector(nil, AllowPrivateTargets())
	err := c.VerifyConnection(context.Background(), connectors.Connection{Region: testRegion,
		Credentials: connectors.Secrets{credAuthURL: srv.URL + "/v3", credID: goodCredID, credSecret: goodSecret}})
	var ue *connectors.UserError
	if !errors.As(err, &ue) || strings.Contains(err.Error(), "418") || !strings.Contains(err.Error(), "did not answer like an OpenStack Identity service") {
		t.Fatalf("got %v", err)
	}
}

// TestRunNamesTheAuthHostAsPublic: a run hands the run task the auth URL's
// host to check again before Terraform connects, except on a laptop, where
// private targets are allowed.
func TestRunNamesTheAuthHostAsPublic(t *testing.T) {
	env := map[string]string{"OS_AUTH_URL": "https://cloud.example.com:5000/v3"}
	got, err := runPublicHosts(env, false)
	if err != nil || len(got) != 1 || got[0] != "cloud.example.com" {
		t.Fatalf("runPublicHosts = %v, %v", got, err)
	}
	if got, err := runPublicHosts(map[string]string{"OS_AUTH_URL": "https://[2001:db8::1]:5000/v3"}, false); err != nil || got[0] != "2001:db8::1" {
		t.Fatalf("IPv6 literal: %v, %v", got, err)
	}
	if got, err := runPublicHosts(env, true); err != nil || got != nil {
		t.Fatalf("with private targets allowed: %v, %v", got, err)
	}
	if _, err := runPublicHosts(map[string]string{}, false); err == nil {
		t.Fatal("a run with no auth URL got no error")
	}
}
