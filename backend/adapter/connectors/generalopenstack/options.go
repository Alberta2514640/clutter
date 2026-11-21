package generalopenstack

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

// optionsTTL is how long a connection's lists are kept before the cloud is
// asked again.
const optionsTTL = 10 * time.Minute

// maxCachedConnections bounds the option-list cache.
const maxCachedConnections = 1024

// maxListResponse caps one list reply Clutter reads.
const maxListResponse = 8 << 20

// maxChoices caps one list. A cloud with more still takes a typed name.
const maxChoices = 500

// The catalog service types the lists come from, and how messages name them.
const (
	serviceCompute = "compute"
	serviceImage   = "image"
	serviceNetwork = "network"
)

var serviceTitle = map[string]string{
	serviceCompute: "compute (Nova)",
	serviceImage:   "image (Glance)",
	serviceNetwork: "network (Neutron)",
}

// Options implements connectors.OptionLister. With one Keystone token it
// lists the flavors (Nova GET /flavors/detail), the active images (Glance GET
// /v2/images?status=active) and the external networks (Neutron GET
// /v2.0/networks?router:external=true) the credential's project can see, in
// the connection's region. Requests go through the same guarded client as
// VerifyConnection, to the public endpoints the catalog names; an endpoint
// must be https (or http with AllowPrivateTargets).
//
// A list that fails is left out and the others are returned; only when every
// list fails is the first error returned. Only names the renderer accepts
// (validCloudName) are offered. For a setting whose DevStack default the
// cloud does not offer, Suggested names a choice that works there instead
// (suggestFlavors, suggestImage, suggestNetwork). A complete answer is cached
// per connection for optionsTTL.
func (c *Connector) Options(ctx context.Context, conn connectors.Connection) (connectors.Options, error) {
	authURL, err := c.checkConnection(ctx, conn)
	if err != nil {
		return connectors.Options{}, err
	}
	key := optionsKey(conn, authURL)
	if o, ok := c.options.get(key); ok {
		return o, nil
	}

	actx, cancel := context.WithTimeout(ctx, c.verifyTimeout)
	token, catalog, err := c.authenticate(actx, authURL, conn.Credentials[credID], conn.Credentials[credSecret])
	cancel()
	if err != nil {
		return connectors.Options{}, err
	}
	if token == "" {
		return connectors.Options{}, &connectors.UserError{Message: "The cloud accepted the credential but sent no token. Check that the auth URL points at the Identity service."}
	}

	lists := []struct {
		service string
		list    func(ctx context.Context, base, token string) (connectors.Options, error)
	}{
		{serviceCompute, c.flavorChoices},
		{serviceImage, c.imageChoices},
		{serviceNetwork, c.networkChoices},
	}
	type result struct {
		opts connectors.Options
		err  error
	}
	results := make([]result, len(lists))
	var wg sync.WaitGroup
	for i, l := range lists {
		wg.Go(func() {
			base, err := serviceURL(catalog, l.service, conn.Region, c.allowPrivate)
			if err != nil {
				results[i] = result{err: err}
				return
			}
			lctx, cancel := context.WithTimeout(ctx, c.verifyTimeout)
			defer cancel()
			o, err := l.list(lctx, base, token)
			results[i] = result{opts: o, err: err}
		})
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return connectors.Options{}, err
	}

	out := connectors.Options{Settings: map[string][]connectors.Choice{}}
	var firstErr error
	for _, r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		for name, choices := range r.opts.Settings {
			out.Settings[name] = choices
		}
		for name, value := range r.opts.Suggested {
			if out.Suggested == nil {
				out.Suggested = map[string]string{}
			}
			out.Suggested[name] = value
		}
	}
	if firstErr != nil {
		if len(out.Settings) == 0 {
			return connectors.Options{}, firstErr
		}
		slog.Default().Info("listing setting choices was partial", "connector", ID, "connection_id", conn.ID, "err", connectors.UserMessage(firstErr, "not a user error"))
		return out, nil
	}
	c.options.put(key, out)
	return out, nil
}

var _ connectors.OptionLister = (*Connector)(nil)

// optionsKey keys a connection's lists. It holds a hash of the secret, never
// the secret, so lists made with one credential are never served for another.
// A connection without an id is not cached.
func optionsKey(conn connectors.Connection, authURL string) string {
	if conn.ID == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(conn.Credentials[credSecret]))
	return strings.Join([]string{conn.ID, conn.Region, authURL, conn.Credentials[credID], hex.EncodeToString(sum[:])}, "\x00")
}

// serviceURL returns the public endpoint of a service type in a region, from
// the token's catalog, without a trailing slash. It must be https (or http
// when allowHTTP is set) with no user name, query or fragment; the guarded
// client then checks the address it resolves to.
func serviceURL(catalog []catalogService, service, region string, allowHTTP bool) (string, error) {
	for _, svc := range catalog {
		if svc.Type != service {
			continue
		}
		for _, ep := range svc.Endpoints {
			if ep.Interface != "public" || ep.region() != region {
				continue
			}
			u, err := url.Parse(strings.TrimSpace(ep.URL))
			if err != nil || (u.Scheme != "https" && !(allowHTTP && u.Scheme == "http")) || u.Host == "" || u.Hostname() == "" ||
				u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
				return "", &connectors.UserError{Message: fmt.Sprintf("The cloud's catalog lists a %s address Clutter does not use. It must be an https address with no user name or query.", serviceTitle[service])}
			}
			u.Path = strings.TrimRight(u.Path, "/")
			u.RawPath = ""
			return u.String(), nil
		}
	}
	return "", &connectors.UserError{Message: fmt.Sprintf("The cloud's catalog has no public %s endpoint in region %s.", serviceTitle[service], region)}
}

// withVersion appends an API version to a service endpoint unless the
// endpoint already ends in it.
func withVersion(base, version string) string {
	if strings.HasSuffix(base, "/"+version) {
		return base
	}
	return base + "/" + version
}

// flavor is one Nova flavor: RAM in MiB, disk in GB.
type flavor struct {
	Name  string `json:"name"`
	VCPUs int    `json:"vcpus"`
	RAM   int    `json:"ram"`
	Disk  int    `json:"disk"`
}

// flavorChoices lists the flavors for the three flavor settings, smallest
// first, labelled with their size, with a suggestion for each flavor setting
// whose default the cloud lacks.
func (c *Connector) flavorChoices(ctx context.Context, base, token string) (connectors.Options, error) {
	var resp struct {
		Flavors []flavor `json:"flavors"`
	}
	if err := c.getJSON(ctx, base+"/flavors/detail", token, serviceCompute, &resp); err != nil {
		return connectors.Options{}, err
	}
	fl := resp.Flavors
	sort.SliceStable(fl, func(i, j int) bool {
		a, b := fl[i], fl[j]
		if a.VCPUs != b.VCPUs {
			return a.VCPUs < b.VCPUs
		}
		if a.RAM != b.RAM {
			return a.RAM < b.RAM
		}
		if a.Disk != b.Disk {
			return a.Disk < b.Disk
		}
		return a.Name < b.Name
	})
	choices := []connectors.Choice{}
	kept := []flavor{}
	seen := map[string]bool{}
	for _, f := range fl {
		if !validCloudName(f.Name) || seen[f.Name] || len(choices) >= maxChoices {
			continue
		}
		seen[f.Name] = true
		kept = append(kept, f)
		cpus := "vCPUs"
		if f.VCPUs == 1 {
			cpus = "vCPU"
		}
		choices = append(choices, connectors.Choice{Value: f.Name,
			Label: fmt.Sprintf("%s (%d %s, %s RAM, %d GB disk)", f.Name, f.VCPUs, cpus, memory(f.RAM), f.Disk)})
	}
	out := connectors.Options{Settings: map[string][]connectors.Choice{}, Suggested: suggestFlavors(kept)}
	for _, s := range []connectors.FieldSpec{settingFlavorSmall, settingFlavorMedium, settingFlavorLarge} {
		out.Settings[s.Name] = append([]connectors.Choice(nil), choices...)
	}
	return out, nil
}

// flavorMinRAM is the least RAM, in MiB, a suggested flavor has for each
// flavor setting. render.go applies one flavor setting per VM size.
var flavorMinRAM = []struct {
	spec   connectors.FieldSpec
	minRAM int
}{
	{settingFlavorSmall, 1024},
	{settingFlavorMedium, 2048},
	{settingFlavorLarge, 4096},
}

// suggestFlavors picks a flavor for each flavor setting whose default is not
// among the offered flavors: the smallest with at least the setting's RAM
// (flavorMinRAM) and at least one vCPU. A flavor with a local disk comes
// before one without: VMs boot from the image onto the flavor's disk, and
// Nova refuses that on a zero-disk flavor unless the caller is an admin.
// Ties go to fewer vCPUs, then less RAM, then the smaller disk (flavorBefore). A setting with no
// such flavor gets no suggestion. Each size then takes a bigger flavor than
// the size below when the cloud offers one that boots the same way.
func suggestFlavors(offered []flavor) map[string]string {
	byName := map[string]*flavor{}
	for i := range offered {
		byName[offered[i].Name] = &offered[i]
	}
	out := map[string]string{}
	// prev is the flavor the size below got (offered default or suggestion),
	// so each size gets a bigger one when the cloud has it: on VEXXHOST the
	// cheapest flavor has enough memory for small and medium, and both sizes
	// on one flavor made "scale up to medium" change nothing.
	var prev *flavor
	for _, s := range flavorMinRAM {
		if f, ok := byName[defaultString(s.spec)]; ok {
			prev = f
			continue
		}
		best := pickFlavor(offered, s.minRAM)
		if prev != nil {
			// A bigger flavor that boots like this one (a local disk when this
			// has one); otherwise the plain pick stands.
			if bigger := pickFlavor(offered, prev.RAM+1); bigger != nil && (bigger.Disk > 0 || best == nil || best.Disk == 0) {
				best = bigger
			}
		}
		if best != nil {
			out[s.spec.Name] = best.Name
			prev = best
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// pickFlavor is the first flavor, by flavorBefore, with at least minRAM and
// at least one vCPU, or nil.
func pickFlavor(offered []flavor, minRAM int) *flavor {
	var best *flavor
	for i := range offered {
		f := &offered[i]
		if f.RAM < minRAM || f.VCPUs < 1 {
			continue
		}
		if best == nil || flavorBefore(*f, *best) {
			best = f
		}
	}
	return best
}

// flavorBefore orders the flavors that are big enough for a size: one with a
// local disk first (servers boot from the image onto it, and a zero-disk
// flavor needs an admin-only policy), then the fewest vCPUs, which drive the
// price, then the least memory, then the smallest disk. On VEXXHOST that is
// the priced v3-starter generation, not the older, unpriced v1-standard one
// (2026-11-20).
func flavorBefore(a, b flavor) bool {
	if (a.Disk > 0) != (b.Disk > 0) {
		return a.Disk > 0
	}
	if a.VCPUs != b.VCPUs {
		return a.VCPUs < b.VCPUs
	}
	if a.RAM != b.RAM {
		return a.RAM < b.RAM
	}
	if a.Disk != b.Disk {
		return a.Disk < b.Disk
	}
	return a.Name < b.Name
}

// defaultString is a setting's default as text; "" when it has none.
func defaultString(f connectors.FieldSpec) string {
	s, _ := f.Default.(string)
	return s
}

// memory writes a flavor's RAM, which Nova gives in MiB.
func memory(mib int) string {
	if mib >= 1024 && mib%1024 == 0 {
		return fmt.Sprintf("%d GB", mib/1024)
	}
	return fmt.Sprintf("%d MB", mib)
}

// image is one Glance v2 image, with the properties the suggestion reads.
// Glance puts properties at the top level of the image; os_distro,
// os_version and architecture are the ones its docs define for this.
type image struct {
	Name         string     `json:"name"`
	Status       propString `json:"status"`
	OSDistro     propString `json:"os_distro"`
	OSVersion    propString `json:"os_version"`
	Architecture propString `json:"architecture"`
	CreatedAt    propString `json:"created_at"`
}

// propString reads an image property that should be text. Any other JSON
// value reads as "", so one odd property never fails the whole list.
type propString string

func (p *propString) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*p = propString(s)
	}
	return nil
}

// imageChoices lists the active images for the image setting, by name, with
// a suggestion when no image name starts with the default.
func (c *Connector) imageChoices(ctx context.Context, base, token string) (connectors.Options, error) {
	var resp struct {
		Images []image `json:"images"`
	}
	u := withVersion(base, "v2") + fmt.Sprintf("/images?status=active&limit=%d", maxChoices)
	if err := c.getJSON(ctx, u, token, serviceImage, &resp); err != nil {
		return connectors.Options{}, err
	}
	names := make([]string, 0, len(resp.Images))
	for _, im := range resp.Images {
		names = append(names, im.Name)
	}
	choices := nameChoices(names)
	out := connectors.Options{Settings: map[string][]connectors.Choice{settingImage.Name: choices}}
	if s := suggestImage(choices, resp.Images); s != "" {
		out.Suggested = map[string]string{settingImage.Name: s}
	}
	return out, nil
}

// suggestImage picks an image when no offered name starts with the image
// setting's default (render.go matches the setting as the start of a name).
// It picks among offered, active Ubuntu images (os_distro ubuntu, or a name
// that starts with "ubuntu") that are not Kubernetes images (a name holding
// "kube") and are built for x86_64 (ubuntuX86): an LTS release before any
// other, then the newest release, then the newest upload, then the name.
// With no such image it returns "".
func suggestImage(offered []connectors.Choice, images []image) string {
	def := defaultString(settingImage)
	listed := map[string]bool{}
	for _, ch := range offered {
		if strings.HasPrefix(ch.Value, def) {
			return ""
		}
		listed[ch.Value] = true
	}
	var best *rankedImage
	for _, im := range images {
		if !listed[im.Name] || !ubuntuX86(im) {
			continue
		}
		r := rank(im)
		if best == nil || r.before(*best) {
			best = &r
		}
	}
	if best == nil {
		return ""
	}
	return best.name
}

// ubuntuX86 reports whether an image is an active Ubuntu image for x86_64
// that is not a Kubernetes image. An architecture property of x86_64 or
// amd64 counts; without the property, the name must say x86_64 or amd64. A
// name that says arm64 or aarch64 never counts.
func ubuntuX86(im image) bool {
	name := strings.ToLower(im.Name)
	if !strings.EqualFold(string(im.Status), "active") {
		return false
	}
	if !strings.EqualFold(string(im.OSDistro), "ubuntu") && !strings.HasPrefix(name, "ubuntu") {
		return false
	}
	if strings.Contains(name, "kube") || strings.Contains(name, "arm64") || strings.Contains(name, "aarch64") {
		return false
	}
	switch strings.ToLower(string(im.Architecture)) {
	case "x86_64", "amd64":
		return true
	case "":
		return strings.Contains(name, "x86_64") || strings.Contains(name, "amd64")
	default:
		return false
	}
}

// ubuntuRelease finds a release such as 24.04 or 24.04.2 in an image name.
var ubuntuRelease = regexp.MustCompile(`\b(\d{2})\.(\d{2})(?:\.(\d+))?`)

type rankedImage struct {
	name    string
	lts     bool
	release [3]int
	created time.Time
}

// rank reads an image's release from its name, else from os_version. A
// release is LTS when the name says "LTS" or it is an even year's April
// release (Ubuntu's LTS rule).
func rank(im image) rankedImage {
	r := rankedImage{name: im.Name}
	m := ubuntuRelease.FindStringSubmatch(im.Name)
	if m == nil {
		m = ubuntuRelease.FindStringSubmatch(string(im.OSVersion))
	}
	if m != nil {
		for i := range 3 {
			r.release[i], _ = strconv.Atoi(m[i+1])
		}
	}
	r.lts = strings.Contains(strings.ToLower(im.Name), "lts") || (m != nil && r.release[0]%2 == 0 && r.release[1] == 4)
	r.created, _ = time.Parse(time.RFC3339, string(im.CreatedAt))
	return r
}

func (a rankedImage) before(b rankedImage) bool {
	if a.lts != b.lts {
		return a.lts
	}
	for i := range 3 {
		if a.release[i] != b.release[i] {
			return a.release[i] > b.release[i]
		}
	}
	if !a.created.Equal(b.created) {
		return a.created.After(b.created)
	}
	return a.name < b.name
}

// networkChoices lists the external networks for the external network
// setting, by name, with a suggestion when the default is not among them.
func (c *Connector) networkChoices(ctx context.Context, base, token string) (connectors.Options, error) {
	var resp struct {
		Networks []struct {
			Name string `json:"name"`
		} `json:"networks"`
	}
	if err := c.getJSON(ctx, withVersion(base, "v2.0")+"/networks?router:external=true", token, serviceNetwork, &resp); err != nil {
		return connectors.Options{}, err
	}
	names := make([]string, 0, len(resp.Networks))
	for _, n := range resp.Networks {
		names = append(names, n.Name)
	}
	choices := nameChoices(names)
	out := connectors.Options{Settings: map[string][]connectors.Choice{settingExternalNetwork.Name: choices}}
	if s := suggestNetwork(choices, len(resp.Networks)); s != "" {
		out.Suggested = map[string]string{settingExternalNetwork.Name: s}
	}
	return out, nil
}

// suggestNetwork picks the cloud's one external network when the default is
// not offered. With several external networks (total counts them all, named
// or not) it picks none: which one routes to the internet is the user's call.
func suggestNetwork(offered []connectors.Choice, total int) string {
	if total != 1 || len(offered) != 1 || offered[0].Value == defaultString(settingExternalNetwork) {
		return ""
	}
	return offered[0].Value
}

// nameChoices keeps the distinct names the renderer accepts, sorted, each
// labelled with itself.
func nameChoices(names []string) []connectors.Choice {
	seen := map[string]bool{}
	kept := []string{}
	for _, n := range names {
		if validCloudName(n) && !seen[n] {
			seen[n] = true
			kept = append(kept, n)
		}
	}
	sort.Slice(kept, func(i, j int) bool { return strings.ToLower(kept[i]) < strings.ToLower(kept[j]) })
	if len(kept) > maxChoices {
		kept = kept[:maxChoices]
	}
	out := make([]connectors.Choice, 0, len(kept))
	for _, n := range kept {
		out = append(out, connectors.Choice{Value: n, Label: n})
	}
	return out
}

// getJSON reads one list from a service with the token. Errors say which
// service failed, never the address, the status or the cloud's own text.
func (c *Connector) getJSON(ctx context.Context, rawURL, token, service string, v any) error {
	title := serviceTitle[service]
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return &connectors.UserError{Message: fmt.Sprintf("The cloud's catalog lists a %s address that is not valid.", title)}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "clutter-adapter")
	req.Header.Set("X-Auth-Token", token)

	resp, err := c.client.Do(req)
	if err != nil {
		return serviceReachError(title, err)
	}
	defer resp.Body.Close()
	defer func() { _, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxListResponse)) }()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return &connectors.UserError{Message: fmt.Sprintf("The cloud's %s service refused to list for this credential. Check that the credential's role can read it; an application credential with access rules may not.", title)}
	default:
		slog.Default().Info("a list call failed", "connector", ID, "service", service, "status", resp.StatusCode, "host", req.URL.Host)
		return &connectors.UserError{Message: fmt.Sprintf("The cloud's %s service did not answer the list request as expected.", title)}
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxListResponse)).Decode(v); err != nil {
		return &connectors.UserError{Message: fmt.Sprintf("The cloud's %s service sent a list Clutter could not read.", title)}
	}
	return nil
}

// serviceReachError says why a service could not be reached, without its
// address or the transport's own text.
func serviceReachError(title string, err error) error {
	var (
		dnsErr  *net.DNSError
		netErr  net.Error
		certErr *tls.CertificateVerificationError
		unknown x509.UnknownAuthorityError
		host    x509.HostnameError
	)
	switch {
	case errors.Is(err, errBlockedAddress):
		return &connectors.UserError{Message: fmt.Sprintf("The cloud's catalog points its %s service at an address Clutter does not connect to (a private, loopback, link-local or metadata address).", title)}
	case errors.Is(err, context.Canceled):
		return err
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return &connectors.UserError{Message: fmt.Sprintf("The cloud's %s service did not answer in time.", title)}
	case errors.As(err, &dnsErr):
		return &connectors.UserError{Message: fmt.Sprintf("Clutter could not find the host name of the cloud's %s service.", title)}
	case errors.As(err, &certErr), errors.As(err, &unknown), errors.As(err, &host):
		return &connectors.UserError{Message: fmt.Sprintf("The cloud's %s service has a TLS certificate Clutter does not trust.", title)}
	default:
		return &connectors.UserError{Message: fmt.Sprintf("Clutter could not connect to the cloud's %s service.", title)}
	}
}

// optionsCache keeps each connection's complete lists for optionsTTL. It
// holds names from the cloud only, never a credential or a token.
type optionsCache struct {
	mu      sync.Mutex
	now     func() time.Time // nil means time.Now
	entries map[string]cachedOptions
}

type cachedOptions struct {
	at   time.Time
	opts connectors.Options
}

func (oc *optionsCache) clock() time.Time {
	if oc.now == nil {
		return time.Now()
	}
	return oc.now()
}

func (oc *optionsCache) get(key string) (connectors.Options, bool) {
	if key == "" {
		return connectors.Options{}, false
	}
	oc.mu.Lock()
	defer oc.mu.Unlock()
	e, ok := oc.entries[key]
	if !ok {
		return connectors.Options{}, false
	}
	if oc.clock().Sub(e.at) >= optionsTTL {
		delete(oc.entries, key)
		return connectors.Options{}, false
	}
	return copyOptions(e.opts), true
}

func (oc *optionsCache) put(key string, o connectors.Options) {
	if key == "" {
		return
	}
	oc.mu.Lock()
	defer oc.mu.Unlock()
	if oc.entries == nil {
		oc.entries = map[string]cachedOptions{}
	}
	now := oc.clock()
	if len(oc.entries) >= maxCachedConnections {
		for k, e := range oc.entries {
			if now.Sub(e.at) >= optionsTTL {
				delete(oc.entries, k)
			}
		}
	}
	for k := range oc.entries {
		if len(oc.entries) < maxCachedConnections {
			break
		}
		delete(oc.entries, k)
	}
	oc.entries[key] = cachedOptions{at: now, opts: copyOptions(o)}
}

// copyOptions copies the lists and the suggestions, so a caller that changes
// what it was given cannot change the cache.
func copyOptions(o connectors.Options) connectors.Options {
	out := connectors.Options{Settings: make(map[string][]connectors.Choice, len(o.Settings))}
	for k, v := range o.Settings {
		out.Settings[k] = append([]connectors.Choice{}, v...)
	}
	out.Suggested = maps.Clone(o.Suggested)
	return out
}
