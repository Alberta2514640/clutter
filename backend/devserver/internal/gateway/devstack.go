package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LinkDevstackPath is the dev-only route behind the website's "Use this
// machine's DevStack" button on the OpenStack connection form.
const LinkDevstackPath = "/dev/link-devstack"

// Plain answers the website shows as they stand.
const (
	devstackOff        = "DevStack linking works only in Clutter started with npm run dev."
	devstackNotMade    = "DevStack is not set up on this machine. Make it with npm run devstack -- create."
	devstackNotRunning = "DevStack is not running. Start it with npm run devstack -- start."
	devstackNoAddress  = "Could not read DevStack's address. Check the devstack VM, then try again."
	devstackNoCred     = "Could not make a DevStack credential. Check the gateway log."
	devstackNoB        = "Clutter's deployment service is not running. Start Clutter with npm run dev."
	devstackBRefused   = "B API refused the connection."
)

// devstackStepTimeout bounds one command in the VM; the openstack client
// takes a few seconds per call.
const devstackStepTimeout = 90 * time.Second

// Runner runs one command on this machine and returns its standard output.
// That output can hold a credential secret: it is never logged, and an error
// never carries it.
type Runner interface {
	Run(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error)
}

// DevstackConfig is what the DevStack link route needs.
type DevstackConfig struct {
	// Enabled is ADAPTER_ALLOW_PRIVATE_TARGETS=1, the flag dev.sh gives B so
	// it accepts DevStack's plain-http auth URL at a private address. Without
	// it B refuses such a connection, so the route refuses first.
	Enabled bool
	// AdapterURL is B API on loopback, for example http://127.0.0.1:8790.
	AdapterURL string
	// VM is the Lima VM that runs DevStack (lib.sh DEVSTACK_VM).
	VM string
	// Script is devstack/credential.sh, which runs inside the VM.
	Script string
	Run    Runner
	Client *http.Client
}

// connectorID matches B's connector ids.
var connectorID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

// LinkDevstackHandler makes a new OpenStack application credential in this
// machine's DevStack VM and creates a connection from it in B, for one
// organization and the signed-in caller. The caller's token proves
// membership through A's real authorizer and accounts route, and B checks it
// again when it creates the connection. The secret goes from the VM to B in
// this process's memory only: it is never logged or answered. The answer
// names the new connection (connectionName).
func (s *Server) LinkDevstackHandler(cfg DevstackConfig) http.Handler {
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !devPost(w, r) {
			return
		}
		if !cfg.Enabled {
			writeDevJSON(w, http.StatusForbidden, devstackOff)
			return
		}

		var body struct {
			OrganizationID string `json:"organizationId"`
			Connector      string `json:"connector"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&body); err != nil ||
			uuid.Validate(body.OrganizationID) != nil || !connectorID.MatchString(body.Connector) {
			writeDevJSON(w, http.StatusBadRequest, "a valid organizationId and connector are required")
			return
		}
		org := url.PathEscape(body.OrganizationID)

		// Reading the accounts proves the caller first: a stranger stops
		// here, before anything runs in the VM.
		status, raw := s.callRoute(r, http.MethodGet, "/organization/"+org+"/accounts", nil)
		if status != http.StatusOK {
			writeRouteRefusal(w, status, raw)
			return
		}

		ctx := r.Context()
		out, err := runStep(ctx, cfg.Run, nil, "limactl", "list", cfg.VM, "--format", "{{.Status}}")
		state := strings.TrimSpace(string(out))
		if err != nil || state == "" {
			writeDevJSON(w, http.StatusServiceUnavailable, devstackNotMade)
			return
		}
		if state != "Running" {
			writeDevJSON(w, http.StatusServiceUnavailable, devstackNotRunning)
			return
		}

		out, err = runStep(ctx, cfg.Run, nil, "limactl", "shell", "--workdir", "/", cfg.VM, "ip", "-4", "-o", "addr", "show", "dev", "lima0")
		addr, ok := vmAddress(out)
		if err != nil || !ok {
			s.logger.Printf("devstack: could not read the VM's address: %v", err)
			writeDevJSON(w, http.StatusBadGateway, devstackNoAddress)
			return
		}
		authURL := "http://" + addr.String() + "/identity/v3"

		script, err := os.ReadFile(cfg.Script)
		if err != nil {
			s.logger.Printf("devstack: could not read %s: %v", cfg.Script, err)
			writeDevJSON(w, http.StatusInternalServerError, devstackNoCred)
			return
		}
		short := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
		credName, connName := "clutter-"+short, "devstack-"+short
		inVM := func(ctx context.Context, mode string) ([]byte, error) {
			return runStep(ctx, cfg.Run, script, "limactl", "shell", "--workdir", "/", cfg.VM,
				"sudo", "-u", "stack", "-i", "bash", "-s", "--", mode, credName)
		}
		out, err = inVM(ctx, "link")
		cred, ok := parseVMCredential(out)
		if err != nil || !ok {
			if err == nil {
				err = errors.New("its answer was not an id, a secret and a region")
			}
			s.logger.Printf("devstack: could not make the credential %s: %v", credName, err)
			// remove does nothing when the credential was never made.
			s.removeDevstackCredential(ctx, inVM, credName)
			writeDevJSON(w, http.StatusBadGateway, devstackNoCred)
			return
		}

		payload, _ := json.Marshal(map[string]any{
			"connector": body.Connector,
			"name":      connName,
			"region":    cred.Region,
			"credentials": map[string]string{
				"auth_url":                      authURL,
				"application_credential_id":     cred.ID,
				"application_credential_secret": cred.Secret,
			},
		})
		status, raw, err = createConnection(ctx, client, cfg.AdapterURL, org, r.Header.Get("Authorization"), payload)
		if err != nil {
			s.logger.Printf("devstack: B API did not answer: %v", err)
			s.removeDevstackCredential(ctx, inVM, credName)
			writeDevJSON(w, http.StatusBadGateway, devstackNoB)
			return
		}
		if status < 200 || status >= 300 {
			msg := adapterMessage(raw)
			s.logger.Printf("devstack: B API refused the connection %s (%d): %s", connName, status, msg)
			s.removeDevstackCredential(ctx, inVM, credName)
			writeDevJSON(w, status, msg)
			return
		}
		var made struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(raw, &made) != nil || made.Name == "" {
			made.Name = connName
		}
		s.logger.Printf("devstack: made the connection %s with the credential %s, at %s", made.Name, credName, authURL)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "made a DevStack connection", "connectionName": made.Name})
	})
}

// removeDevstackCredential deletes a credential this route made when no
// connection uses it, so a failed link leaves nothing in the VM.
func (s *Server) removeDevstackCredential(ctx context.Context, inVM func(context.Context, string) ([]byte, error), name string) {
	if _, err := inVM(context.WithoutCancel(ctx), "remove"); err != nil {
		s.logger.Printf("devstack: could not remove the credential %s: %v", name, err)
	}
}

func runStep(ctx context.Context, run Runner, stdin []byte, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, devstackStepTimeout)
	defer cancel()
	return run.Run(ctx, stdin, name, args...)
}

// createConnection sends the new connection to B with the caller's token.
func createConnection(ctx context.Context, client *http.Client, base, org, token string, payload []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/organizations/"+org+"/connections", bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, raw, nil
}

// adapterMessage reads the plain sentence out of B's error envelope. B never
// puts a credential in it.
func adapterMessage(raw []byte) string {
	var b struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &b) == nil && strings.TrimSpace(b.Error.Message) != "" {
		return strings.TrimSpace(b.Error.Message)
	}
	return devstackBRefused
}

var inetAddr = regexp.MustCompile(`\binet (\d{1,3}(?:\.\d{1,3}){3})/\d+`)

// vmAddress reads the VM's lima0 address from `ip -4 -o addr show dev
// lima0`. Only a private IPv4 address is taken: that is what Apple's
// virtualization network gives the VM.
func vmAddress(out []byte) (netip.Addr, bool) {
	m := inetAddr.FindSubmatch(out)
	if m == nil {
		return netip.Addr{}, false
	}
	a, err := netip.ParseAddr(string(m[1]))
	if err != nil || !a.Is4() || !a.IsPrivate() {
		return netip.Addr{}, false
	}
	return a, true
}

// vmCredential is what credential.sh link prints. It prints as [redacted]
// so a stray log line cannot carry the secret.
type vmCredential struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
	Region string `json:"region"`
}

func (vmCredential) String() string   { return "[redacted]" }
func (vmCredential) GoString() string { return "[redacted]" }

var (
	credIDPattern     = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)
	credRegionPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
)

// parseVMCredential reads the last line of credential.sh link's output, so a
// login banner before it does no harm.
func parseVMCredential(out []byte) (vmCredential, bool) {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var c vmCredential
	if json.Unmarshal([]byte(strings.TrimSpace(lines[len(lines)-1])), &c) != nil {
		return vmCredential{}, false
	}
	if !credIDPattern.MatchString(c.ID) || c.Secret == "" || len(c.Secret) > 8192 || !credRegionPattern.MatchString(c.Region) {
		return vmCredential{}, false
	}
	return c, true
}

// ExecRunner runs commands on this machine with only PATH, HOME and TMPDIR,
// which limactl needs, so the gateway's own secrets (the JWT secret, the
// database string) never reach the child or the VM. On failure it returns
// no output, and its error carries only the last line of standard error.
type ExecRunner struct{}

// Run implements Runner.
func (ExecRunner) Run(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "TMPDIR=" + os.Getenv("TMPDIR")}
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
		return nil, fmt.Errorf("%s: %w: %s", name, err, lines[len(lines)-1])
	}
	return stdout.Bytes(), nil
}
