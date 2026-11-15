package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const testSecret = "s3cr3t-never-shown"

// fakeRunner plays limactl and the VM.
type fakeRunner struct {
	status    string
	statusErr error
	ip        string
	link      string
	linkErr   error
	calls     []string
	stdin     map[string][]byte
}

func (f *fakeRunner) Run(_ context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, call)
	if f.stdin == nil {
		f.stdin = map[string][]byte{}
	}
	f.stdin[call] = stdin
	switch {
	case strings.HasPrefix(call, "limactl list "):
		return []byte(f.status + "\n"), f.statusErr
	case strings.Contains(call, " ip -4 -o addr show dev lima0"):
		return []byte(f.ip), nil
	case strings.Contains(call, " -- link "):
		if f.linkErr != nil {
			return nil, f.linkErr
		}
		return []byte(f.link), nil
	case strings.Contains(call, " -- remove "):
		return nil, nil
	}
	return nil, errors.New("unexpected command: " + call)
}

func (f *fakeRunner) called(part string) []string {
	var out []string
	for _, c := range f.calls {
		if strings.Contains(c, part) {
			out = append(out, c)
		}
	}
	return out
}

func healthyVM() *fakeRunner {
	return &fakeRunner{
		status: "Running",
		ip:     "2: lima0    inet 192.168.105.3/24 metric 100 brd 192.168.105.255 scope global dynamic lima0\\       valid_lft 86000sec\n",
		link:   "a login banner\n" + `{"id": "abc123", "secret": "` + testSecret + `", "region": "RegionOne"}` + "\n",
	}
}

// fakeB plays B API's create-connection route and keeps what it was sent.
type fakeB struct {
	status int
	answer string
	got    map[string]any
	auth   string
	hits   int
}

func (b *fakeB) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.hits++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/organizations/"+testOrg+"/connections" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		b.auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&b.got)
		w.WriteHeader(b.status)
		_, _ = io.WriteString(w, b.answer)
	}))
	t.Cleanup(srv.Close)
	return srv
}

type devstackCase struct {
	enabled bool
	member  bool
	vm      *fakeRunner
	b       *fakeB
	body    string
}

func runDevstack(t *testing.T, c devstackCase) (*httptest.ResponseRecorder, *accountsInvoker, string) {
	t.Helper()
	accountsAt := http.StatusOK
	if !c.member {
		accountsAt = http.StatusForbidden
	}
	a := &accountsInvoker{fakeInvoker: fakeInvoker{allow: true}, complete: "[]", accountsAt: accountsAt}
	var logs bytes.Buffer
	s := NewServer(a, log.New(&logs, "", 0), nil)
	script := filepath.Join(t.TempDir(), "credential.sh")
	if err := os.WriteFile(script, []byte("echo the script\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapterURL := "http://127.0.0.1:1" // nothing listens here
	if c.b != nil {
		adapterURL = c.b.server(t).URL
	}
	h := s.LinkDevstackHandler(DevstackConfig{Enabled: c.enabled, AdapterURL: adapterURL, VM: "devstack", Script: script, Run: c.vm})
	body := c.body
	if body == "" {
		body = `{"organizationId":"` + testOrg + `","connector":"general-openstack"}`
	}
	req := httptest.NewRequest(http.MethodPost, LinkDevstackPath, strings.NewReader(body))
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Authorization", "token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, a, logs.String()
}

func message(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var b struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("body %q: %v", rec.Body, err)
	}
	return b.Message
}

func TestLinkDevstackMakesAConnectionWithoutShowingTheSecret(t *testing.T) {
	vm, b := healthyVM(), &fakeB{status: http.StatusCreated, answer: `{"id":"c1","name":"devstack-x"}`}
	rec, _, logs := runDevstack(t, devstackCase{enabled: true, member: true, vm: vm, b: b})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		ConnectionName string `json:"connectionName"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.ConnectionName != "devstack-x" {
		t.Fatalf("connectionName %q", out.ConnectionName)
	}
	if strings.Contains(rec.Body.String(), testSecret) || strings.Contains(logs, testSecret) {
		t.Fatalf("the secret leaked: body %q, log %q", rec.Body, logs)
	}

	link := vm.called(" -- link ")
	if len(link) != 1 || !regexp.MustCompile(`^limactl shell --workdir / devstack sudo -u stack -i bash -s -- link clutter-[0-9a-f]{8}$`).MatchString(link[0]) {
		t.Fatalf("link call: %v", link)
	}
	if string(vm.stdin[link[0]]) != "echo the script\n" {
		t.Fatalf("the script did not reach the VM on stdin")
	}
	if len(vm.called(" -- remove ")) != 0 {
		t.Fatalf("removed the credential of a working connection: %v", vm.calls)
	}
	short := strings.TrimPrefix(link[0][strings.LastIndex(link[0], " ")+1:], "clutter-")

	if b.auth != "token" {
		t.Fatalf("B got Authorization %q, want the caller's token", b.auth)
	}
	creds, _ := b.got["credentials"].(map[string]any)
	if short == "" || b.got["connector"] != "general-openstack" || b.got["name"] != devstackName || b.got["region"] != "RegionOne" ||
		creds["auth_url"] != "http://192.168.105.3/identity/v3" || creds["application_credential_id"] != "abc123" ||
		creds["application_credential_secret"] != testSecret {
		t.Fatalf("B got %v", b.got)
	}
}

// The website names the connection when "Local OpenStack" is taken; a name
// that is not plain is refused before anything runs in the VM.
func TestLinkDevstackTakesTheWebsitesName(t *testing.T) {
	vm, b := healthyVM(), &fakeB{status: http.StatusCreated, answer: `{"id":"c1","name":"Local OpenStack 2"}`}
	rec, _, _ := runDevstack(t, devstackCase{enabled: true, member: true, vm: vm, b: b,
		body: `{"organizationId":"` + testOrg + `","connector":"general-openstack","name":"Local OpenStack 2"}`})
	if rec.Code != http.StatusOK || b.got["name"] != "Local OpenStack 2" {
		t.Fatalf("status %d: %s; B got name %v", rec.Code, rec.Body, b.got["name"])
	}

	for _, name := range []string{" Local", "Local\nOpenStack", "${x}", strings.Repeat("a", 65)} {
		vm, b := healthyVM(), &fakeB{status: http.StatusCreated}
		body, _ := json.Marshal(map[string]string{"organizationId": testOrg, "connector": "general-openstack", "name": name})
		rec, _, _ := runDevstack(t, devstackCase{enabled: true, member: true, vm: vm, b: b, body: string(body)})
		if rec.Code != http.StatusBadRequest || len(vm.calls) != 0 || b.hits != 0 {
			t.Fatalf("name %q: status %d, VM %v, B %d", name, rec.Code, vm.calls, b.hits)
		}
	}
}

func TestLinkDevstackIsRefusedWhenNotLocal(t *testing.T) {
	vm, b := healthyVM(), &fakeB{status: http.StatusCreated}
	rec, a, _ := runDevstack(t, devstackCase{enabled: false, member: true, vm: vm, b: b})
	if rec.Code != http.StatusForbidden || message(t, rec) != devstackOff {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if len(a.calls) != 0 || len(vm.calls) != 0 || b.hits != 0 {
		t.Fatalf("calls: A %v, VM %v, B %d", a.calls, vm.calls, b.hits)
	}
}

func TestLinkDevstackRefusesANonMemberBeforeTheVM(t *testing.T) {
	vm, b := healthyVM(), &fakeB{status: http.StatusCreated}
	rec, a, _ := runDevstack(t, devstackCase{enabled: true, member: false, vm: vm, b: b})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if len(a.calls) != 1 || a.calls[0] != "organization-get-accounts" || len(vm.calls) != 0 || b.hits != 0 {
		t.Fatalf("calls: A %v, VM %v, B %d", a.calls, vm.calls, b.hits)
	}
}

func TestLinkDevstackRefusesABadBody(t *testing.T) {
	for _, body := range []string{
		`{"organizationId":"not-a-uuid","connector":"general-openstack"}`,
		`{"organizationId":"` + testOrg + `","connector":"General OpenStack"}`,
		`{"organizationId":"` + testOrg + `"}`,
		// B's ConnectorIDPattern takes 2 to 41 characters.
		`{"organizationId":"` + testOrg + `","connector":"g"}`,
		`{"organizationId":"` + testOrg + `","connector":"` + "g" + strings.Repeat("x", 41) + `"}`,
	} {
		vm := healthyVM()
		rec, a, _ := runDevstack(t, devstackCase{enabled: true, member: true, vm: vm, body: body})
		if rec.Code != http.StatusBadRequest || len(a.calls) != 0 || len(vm.calls) != 0 {
			t.Fatalf("%s: status %d, A %v, VM %v", body, rec.Code, a.calls, vm.calls)
		}
	}
}

func TestLinkDevstackSaysWhenTheVMIsNotRunning(t *testing.T) {
	for name, tc := range map[string]struct {
		status string
		err    error
		want   string
	}{
		"stopped":    {"Stopped", nil, devstackNotRunning},
		"no VM":      {"", nil, devstackNotMade},
		"no limactl": {"", errors.New("exec: limactl: not found"), devstackNotMade},
	} {
		t.Run(name, func(t *testing.T) {
			vm, b := healthyVM(), &fakeB{status: http.StatusCreated}
			vm.status, vm.statusErr = tc.status, tc.err
			rec, _, _ := runDevstack(t, devstackCase{enabled: true, member: true, vm: vm, b: b})
			if rec.Code != http.StatusServiceUnavailable || message(t, rec) != tc.want {
				t.Fatalf("status %d: %s", rec.Code, rec.Body)
			}
			if len(vm.calls) != 1 || b.hits != 0 {
				t.Fatalf("VM %v, B %d", vm.calls, b.hits)
			}
		})
	}
}

func TestLinkDevstackStopsWithoutAPrivateAddress(t *testing.T) {
	for _, ip := range []string{"", "2: lima0    inet 8.8.8.8/24 scope global lima0"} {
		vm, b := healthyVM(), &fakeB{status: http.StatusCreated}
		vm.ip = ip
		rec, _, _ := runDevstack(t, devstackCase{enabled: true, member: true, vm: vm, b: b})
		if rec.Code != http.StatusBadGateway || message(t, rec) != devstackNoAddress {
			t.Fatalf("%q: status %d: %s", ip, rec.Code, rec.Body)
		}
		if len(vm.called(" -- link ")) != 0 || b.hits != 0 {
			t.Fatalf("%q: VM %v, B %d", ip, vm.calls, b.hits)
		}
	}
}

func TestLinkDevstackRemovesTheCredentialWhenBRefuses(t *testing.T) {
	vm := healthyVM()
	b := &fakeB{status: http.StatusUnprocessableEntity, answer: `{"error":{"code":"verification_failed","message":"The cloud refused these credentials. Check them and try again."}}`}
	rec, _, logs := runDevstack(t, devstackCase{enabled: true, member: true, vm: vm, b: b})
	if rec.Code != http.StatusUnprocessableEntity || message(t, rec) != "The cloud refused these credentials. Check them and try again." {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	assertRemovedSame(t, vm)
	if strings.Contains(rec.Body.String(), testSecret) || strings.Contains(logs, testSecret) {
		t.Fatalf("the secret leaked: body %q, log %q", rec.Body, logs)
	}
}

func TestLinkDevstackRemovesTheCredentialWhenBIsDown(t *testing.T) {
	vm := healthyVM()
	rec, _, logs := runDevstack(t, devstackCase{enabled: true, member: true, vm: vm})
	if rec.Code != http.StatusBadGateway || message(t, rec) != devstackNoB {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	assertRemovedSame(t, vm)
	if strings.Contains(logs, testSecret) {
		t.Fatalf("the secret leaked into the log: %q", logs)
	}
}

func TestLinkDevstackStopsWhenTheCredentialAnswerIsWrong(t *testing.T) {
	for name, vm := range map[string]*fakeRunner{
		"script failed": func() *fakeRunner {
			f := healthyVM()
			f.linkErr = errors.New("limactl: exit status 1: DevStack lists no region.")
			return f
		}(),
		"no secret": func() *fakeRunner {
			f := healthyVM()
			f.link = `{"id":"abc123","secret":"","region":"RegionOne"}`
			return f
		}(),
		"not json": func() *fakeRunner { f := healthyVM(); f.link = "id abc123 secret " + testSecret; return f }(),
	} {
		t.Run(name, func(t *testing.T) {
			b := &fakeB{status: http.StatusCreated}
			rec, _, logs := runDevstack(t, devstackCase{enabled: true, member: true, vm: vm, b: b})
			if rec.Code != http.StatusBadGateway || message(t, rec) != devstackNoCred || b.hits != 0 {
				t.Fatalf("status %d: %s, B %d", rec.Code, rec.Body, b.hits)
			}
			assertRemovedSame(t, vm)
			if strings.Contains(logs, testSecret) || strings.Contains(rec.Body.String(), testSecret) {
				t.Fatalf("the secret leaked: body %q, log %q", rec.Body, logs)
			}
		})
	}
}

// assertRemovedSame checks that the credential the route made, and only
// that one, was removed.
func assertRemovedSame(t *testing.T, vm *fakeRunner) {
	t.Helper()
	link, remove := vm.called(" -- link "), vm.called(" -- remove ")
	if len(link) != 1 || len(remove) != 1 || strings.Replace(link[0], " -- link ", " -- remove ", 1) != remove[0] {
		t.Fatalf("link %v, remove %v", link, remove)
	}
}

func TestVMCredentialNeverPrintsItsSecret(t *testing.T) {
	c := vmCredential{ID: "abc", Secret: testSecret, Region: "RegionOne"}
	var buf bytes.Buffer
	log.New(&buf, "", 0).Printf("%v %+v %#v %s", c, c, c, c)
	if strings.Contains(buf.String(), testSecret) {
		t.Fatalf("printed the secret: %q", buf.String())
	}
}
