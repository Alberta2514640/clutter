package generalopenstack

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/netguard"
)

// maxTokenResponse caps the Keystone reply Clutter reads; a catalog is a few
// kilobytes on most clouds.
const maxTokenResponse = 4 << 20

// errBlockedAddress is returned by the dialer for an address Clutter does not
// connect to.
var errBlockedAddress = errors.New("blocked address")

// newHTTPClient returns the client for Keystone requests. It ignores proxy
// variables, follows no redirect, and checks each address it dials with
// netguard.Blocked, after DNS, so a name that resolves to a refused address
// is refused too.
func newHTTPClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return errBlockedAddress
			}
			if netguard.Blocked(net.ParseIP(host), allowPrivate) {
				return errBlockedAddress
			}
			return nil
		},
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                  nil,
			DialContext:            dialer.DialContext,
			TLSHandshakeTimeout:    10 * time.Second,
			ResponseHeaderTimeout:  20 * time.Second,
			MaxResponseHeaderBytes: 64 << 10,
			ForceAttemptHTTP2:      true,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// normalizeAuthURL checks an auth URL and returns it without a trailing
// slash. It must be https (or http when allowHTTP is set, for a laptop's
// DevStack), carry no user name, query or fragment, and end in /v3, the
// Identity API version the token request uses.
func normalizeAuthURL(raw string, allowHTTP bool) (string, error) {
	bad := &connectors.UserError{Message: "The auth URL must be an https address that ends in /v3, for example https://cloud.example.com:5000/v3."}
	if allowHTTP {
		bad = &connectors.UserError{Message: "The auth URL must be an http or https address that ends in /v3, for example https://cloud.example.com:5000/v3."}
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && !(allowHTTP && u.Scheme == "http")) || u.Host == "" || u.Hostname() == "" ||
		u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return "", bad
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	if !strings.HasSuffix(u.Path, "/v3") {
		return "", bad
	}
	return u.String(), nil
}

// checkConnection checks a connection's credentials and region before any
// request, and returns its auth URL in normal form.
func (c *Connector) checkConnection(ctx context.Context, conn connectors.Connection) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := c.Credentials().CheckSupplied(conn.Credentials); err != nil {
		return "", err
	}
	authURL, err := normalizeAuthURL(conn.Credentials[credAuthURL], c.allowPrivate)
	if err != nil {
		return "", err
	}
	if !connectors.RegionPattern.MatchString(conn.Region) {
		return "", &connectors.UserError{Message: "The connection's region name is not valid."}
	}
	return authURL, nil
}

// VerifyConnection implements connectors.Connector. It asks Keystone for a
// token with the application credential (POST /auth/tokens, no scope: the
// credential carries its own project). 201 means the credential works; the
// connection's region must then be in the token's service catalog. The token
// itself is not used. No error repeats a credential value or a raw cloud
// error.
func (c *Connector) VerifyConnection(ctx context.Context, conn connectors.Connection) error {
	authURL, err := c.checkConnection(ctx, conn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.verifyTimeout)
	defer cancel()
	_, catalog, err := c.authenticate(ctx, authURL, conn.Credentials[credID], conn.Credentials[credSecret])
	if err != nil {
		return err
	}
	regions := catalogRegions(catalog)
	for _, r := range regions {
		if r == conn.Region {
			return nil
		}
	}
	if len(regions) == 0 {
		return &connectors.UserError{Message: "The credential works, but the cloud's service catalog lists no region. Check that the credential's project has services."}
	}
	return &connectors.UserError{Message: fmt.Sprintf("The credential works, but this cloud has no region %q. Its regions are: %s.", conn.Region, strings.Join(regions, ", "))}
}

type tokenRequest struct {
	Auth struct {
		Identity struct {
			Methods               []string `json:"methods"`
			ApplicationCredential struct {
				ID     string `json:"id"`
				Secret string `json:"secret"`
			} `json:"application_credential"`
		} `json:"identity"`
	} `json:"auth"`
}

// catalogEndpoint is one endpoint of a service in a token's catalog.
type catalogEndpoint struct {
	Interface string `json:"interface"`
	Region    string `json:"region"`
	RegionID  string `json:"region_id"`
	URL       string `json:"url"`
}

// region is the endpoint's region id, or its older region name.
func (e catalogEndpoint) region() string {
	if e.RegionID != "" {
		return e.RegionID
	}
	return e.Region
}

// catalogService is one service in a token's catalog.
type catalogService struct {
	Type      string            `json:"type"`
	Endpoints []catalogEndpoint `json:"endpoints"`
}

type tokenResponse struct {
	Token struct {
		Catalog []catalogService `json:"catalog"`
	} `json:"token"`
}

// maxRegions caps the regions named in a message.
const maxRegions = 20

// catalogRegions returns the distinct, well-formed region names in a
// catalog, sorted.
func catalogRegions(catalog []catalogService) []string {
	seen := map[string]bool{}
	var regions []string
	for _, svc := range catalog {
		for _, ep := range svc.Endpoints {
			r := ep.region()
			if connectors.RegionPattern.MatchString(r) && !seen[r] {
				seen[r] = true
				regions = append(regions, r)
			}
		}
	}
	sort.Strings(regions)
	if len(regions) > maxRegions {
		regions = regions[:maxRegions]
	}
	return regions
}

// authenticate requests a token and returns it with its service catalog. The
// token comes from the X-Subject-Token header and is a secret for the
// project: it goes only into an X-Auth-Token header, never into a log, an
// error or a result. ctx bounds the request.
func (c *Connector) authenticate(ctx context.Context, authURL, id, secret string) (string, []catalogService, error) {
	var body tokenRequest
	body.Auth.Identity.Methods = []string{"application_credential"}
	body.Auth.Identity.ApplicationCredential.ID = id
	body.Auth.Identity.ApplicationCredential.Secret = secret
	raw, err := json.Marshal(body)
	if err != nil {
		return "", nil, errors.New("could not build the token request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL+"/auth/tokens", bytes.NewReader(raw))
	if err != nil {
		return "", nil, &connectors.UserError{Message: "The auth URL is not valid."}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "clutter-adapter")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", nil, reachError(err)
	}
	defer resp.Body.Close()
	// Drain what is left so the connection can be reused.
	defer func() { _, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxTokenResponse)) }()

	// The messages are fixed: echoing the status would tell a user what
	// answers at an address. The status goes to B's own log.
	switch resp.StatusCode {
	case http.StatusCreated:
	case http.StatusUnauthorized, http.StatusNotFound:
		return "", nil, &connectors.UserError{Message: "The cloud refused this application credential. Check its ID and secret, and that it has not expired or been deleted."}
	case http.StatusForbidden:
		return "", nil, &connectors.UserError{Message: "The cloud's identity service refused the request. Check that the credential's user still has a role on its project."}
	default:
		slog.Default().Info("keystone answered without a token", "connector", ID, "status", resp.StatusCode, "host", req.URL.Host)
		return "", nil, &connectors.UserError{Message: "The auth URL did not answer like an OpenStack Identity service. Check that it is your cloud's Keystone v3 address."}
	}

	var tok tokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxTokenResponse)).Decode(&tok); err != nil {
		return "", nil, &connectors.UserError{Message: "The cloud accepted the credential, but its reply could not be read. Check that the auth URL points at the Identity service."}
	}
	return resp.Header.Get("X-Subject-Token"), tok.Token.Catalog, nil
}

// reachError says why the auth URL could not be reached, without the URL or
// the transport's own text.
func reachError(err error) error {
	var (
		dnsErr  *net.DNSError
		netErr  net.Error
		certErr *tls.CertificateVerificationError
		unknown x509.UnknownAuthorityError
		host    x509.HostnameError
	)
	switch {
	case errors.Is(err, errBlockedAddress):
		return &connectors.UserError{Message: "The auth URL points at an address Clutter does not connect to (a private, loopback, link-local or metadata address). Use your cloud's public Identity address."}
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return &connectors.UserError{Message: "The cloud's identity service did not answer in time. Check the auth URL and that the cloud is reachable from the internet."}
	case errors.As(err, &dnsErr):
		return &connectors.UserError{Message: "Clutter could not find the auth URL's host name. Check the auth URL."}
	case errors.As(err, &certErr), errors.As(err, &unknown), errors.As(err, &host):
		return &connectors.UserError{Message: "The cloud's identity service has a TLS certificate Clutter does not trust. Use an address with a valid public certificate."}
	case errors.Is(err, context.Canceled):
		return err
	default:
		return &connectors.UserError{Message: "Clutter could not connect to the auth URL. Check the address and port, and that the cloud is reachable."}
	}
}
