package generalopenstack

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/translate"
)

// Problem codes only this connector reports; the shared ones are in
// connectors.
const (
	codeNoNetwork      = "vm_without_network"
	codeInvalidNodeID  = "invalid_node_id"
	codePublicUnrouted = "public_vm_without_internet"
)

// OpenStack's limits as this connector builds.
const (
	maxIngressRules    = 20
	minNetworkPrefix   = 8
	maxNetworkPrefix   = 29
	maxVolumeSizeGB    = 16384
	maxCloudNameLength = 100
)

// cloudNamePattern is the shape of a name that already exists in the cloud:
// an image, a flavor or the external network. OpenStack allows more; this
// keeps quotes, braces, dollar and percent signs, backslashes and newlines
// out. Parentheses and square brackets stay in: public clouds name images
// like "Ubuntu 24.04.2 LTS (x86_64) [2025-04-03]" (VEXXHOST), and the image's
// name_regex escapes them (regexp.QuoteMeta).
var cloudNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._:()\[\]+-]*$`)

// model is a checked deployment: only included blocks, every value parsed.
type model struct {
	prefix string
	// deploymentID tags what the deployment creates (ClutterDeployment).
	deploymentID    string
	image           string
	externalNetwork string
	vms             []vm
	networks        []network
	firewalls       []firewall
	volumes         []volume
	// exports are the address edges this part of a split deployment hands
	// another cloud: by edge key, the VM whose public address goes.
	exports []export
	// crossProblems are what CrossCheck reports. A model with any renders
	// nothing.
	crossProblems []connectors.Problem
	// missingImport is true when a value another cloud hands a VM is not
	// known yet (a preview, or a plan before that cloud was built).
	missingImport bool
}

type export struct{ key, vm string }

type vm struct {
	id, name     string
	size, flavor string
	public       bool
	networks     []string // node ids, sorted; the first carries the public address
	firewalls    []string // node ids, sorted
	peers        []string // node ids of VMs whose address it gets, sorted
	// reachedBy are the node ids of VMs that get this VM's address, sorted:
	// its peer security group lets them in.
	reachedBy []string
	// sshKey is the block's checked public key (connectors.SSHPublicKey), or "".
	sshKey string
	// userData is the startup script as base64 (connectors.StartupUserData),
	// or "" for none. userDataExpr, when set instead, is the expression that
	// builds the startup wrapper with the values the VM's edges hand it
	// (connectors.StartupHCL), and sensitive says one of them is a secret.
	userData     string
	userDataExpr string
	sensitive    bool
}

type network struct {
	id, name, cidr string
	// routed is the block's private_internet field: the network joins the
	// deployment's router, which gives its VMs the internet (SNAT), floating
	// IPs and the deployment's other routed networks.
	routed bool
}

type firewall struct {
	id, name  string
	ethertype string // IPv4 or IPv6, from the source CIDR
	source    string
	rules     []connectors.IngressRule
}

type volume struct {
	id, name string
	sizeGB   int64
	vm       string // node id, or empty
}

// build checks graph g for deployment d and, when nothing is wrong, returns
// the model to render. It reports only what this connector knows; B's core
// runs the capability check beside it, so a value of the wrong type or outside
// the manifest's allowed values is skipped here, never rendered.
func build(g connectors.Graph, d connectors.Deployment) (model, []connectors.Problem) {
	var problems []connectors.Problem
	add := func(p connectors.Problem) { problems = append(problems, p) }

	out := model{prefix: d.Prefix, deploymentID: d.ID}
	if !connectors.ValidPrefix(d.Prefix) {
		add(connectors.Problem{Code: connectors.CodeDeploymentInvalid, Message: "The deployment's name prefix is not valid."})
	}
	problems = append(problems, connectors.DuplicateNames(g, d)...)

	ids := make([]string, 0, len(g.Blocks))
	for id := range g.Blocks {
		if !d.Excluded(id) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	vms := map[string]*vm{}
	for _, id := range ids {
		b := g.Blocks[id]
		if !translate.NodeIDPattern.MatchString(id) || b.ID != id {
			add(connectors.Problem{NodeID: id, Code: codeInvalidNodeID, Message: "The block's id is not valid."})
			continue
		}
		name := d.Prefix + "-" + b.Name
		switch b.Kind {
		case connectors.KindVM:
			v := vm{id: id, name: name}
			v.public, _ = value(b.Fields, fieldPublic).(bool)
			if size, ok := value(b.Fields, fieldSize).(string); ok && sizes.Contains(size) {
				v.size = size
			}
			ud, err := connectors.StartupUserData(b.Fields)
			if err != nil {
				add(fieldProblem(id, connectors.FieldStartupScript, err.Error()))
			}
			v.userData = ud
			key, err := connectors.SSHPublicKey(b.Fields)
			if err != nil {
				add(fieldProblem(id, connectors.FieldSSHKey, err.Error()))
			}
			v.sshKey = key
			out.vms = append(out.vms, v)
		case connectors.KindNetwork:
			n := network{id: id, name: name}
			n.routed, _ = value(b.Fields, fieldPrivateInternet).(bool)
			if s, ok := value(b.Fields, fieldCIDR).(string); ok {
				cidr, err := networkCIDR(s)
				if err != nil {
					add(fieldProblem(id, fieldCIDR.Name, err.Error()))
				}
				n.cidr = cidr
			}
			out.networks = append(out.networks, n)
		case connectors.KindFirewall:
			// Names differ across blocks (connectors.DuplicateNames), which
			// matters here: a VM finds its firewalls by name.
			f := firewall{id: id, name: name}
			if s, ok := value(b.Fields, fieldSourceCIDR).(string); ok {
				src, err := connectors.ParseCIDR(s)
				if err != nil {
					add(fieldProblem(id, fieldSourceCIDR.Name, err.Error()))
				}
				f.source, f.ethertype = src.String(), "IPv4"
				if src.Addr().Is6() {
					f.ethertype = "IPv6"
				}
			}
			if s, ok := value(b.Fields, fieldIngress).(string); ok {
				rules, err := connectors.ParseIngress(s)
				if err == nil && len(rules) > maxIngressRules {
					err = fmt.Errorf("On OpenStack a firewall can allow at most %d entries.", maxIngressRules)
				}
				if err != nil {
					add(fieldProblem(id, fieldIngress.Name, err.Error()))
				}
				// Rules render sorted, so reordering the field changes no file;
				// each rule's address is keyed by its content (ruleSuffix).
				sort.Slice(rules, func(i, j int) bool {
					a, b := rules[i], rules[j]
					if a.Protocol != b.Protocol {
						return a.Protocol < b.Protocol
					}
					if a.FromPort != b.FromPort {
						return a.FromPort < b.FromPort
					}
					return a.ToPort < b.ToPort
				})
				f.rules = rules
			}
			out.firewalls = append(out.firewalls, f)
		case connectors.KindVolume:
			v := volume{id: id, name: name}
			if n, ok := connectors.IntValue(value(b.Fields, fieldSizeGB)); ok {
				if n < 1 || n > maxVolumeSizeGB {
					add(fieldProblem(id, fieldSizeGB.Name, fmt.Sprintf("%s must be from 1 to %d.", fieldSizeGB.Title, maxVolumeSizeGB)))
				}
				v.sizeGB = n
			}
			out.volumes = append(out.volumes, v)
		}
	}
	for i := range out.vms {
		vms[out.vms[i].id] = &out.vms[i]
	}

	// Edges between included blocks. The capability check already refused
	// pairs the manifest does not wire, and noted edges to excluded blocks.
	attachedTo := map[string][]string{}
	for _, e := range g.Edges {
		if d.Excluded(e.Source) || d.Excluded(e.Target) {
			continue
		}
		s, t := g.Blocks[e.Source], g.Blocks[e.Target]
		switch {
		case s.Kind == connectors.KindVM && t.Kind == connectors.KindNetwork:
			if v := vms[e.Source]; v != nil {
				v.networks = append(v.networks, e.Target)
			}
		case s.Kind == connectors.KindFirewall && t.Kind == connectors.KindVM:
			if v := vms[e.Target]; v != nil {
				v.firewalls = append(v.firewalls, e.Source)
			}
		case s.Kind == connectors.KindVolume && t.Kind == connectors.KindVM:
			attachedTo[e.Source] = append(attachedTo[e.Source], e.Target)
		case s.Kind == connectors.KindVM && t.Kind == connectors.KindVM:
			if v := vms[e.Source]; v != nil && vms[e.Target] != nil {
				v.peers = append(v.peers, e.Target)
			}
		}
	}
	peers := map[string][]string{}
	vmIDs := make([]string, 0, len(out.vms))
	for i := range out.vms {
		v := &out.vms[i]
		sort.Strings(v.networks)
		sort.Strings(v.firewalls)
		sort.Strings(v.peers)
		peers[v.id], vmIDs = v.peers, append(vmIDs, v.id)
		if len(v.networks) == 0 {
			add(connectors.Problem{NodeID: v.id, Code: codeNoNetwork,
				Message: "Connect this VM to a network. On OpenStack every VM needs one."})
		}
	}
	routed := map[string]bool{}
	for _, n := range out.networks {
		routed[n.id] = n.routed
	}
	for i := range out.vms {
		v := &out.vms[i]
		if v.public && len(v.networks) > 0 && !routed[v.networks[0]] {
			add(connectors.Problem{NodeID: v.id, Code: codePublicUnrouted,
				Message: "On OpenStack a public VM's first network needs internet access. Turn on \"Internet access for private servers\" for that network."})
		}
		for _, p := range v.peers {
			t := vms[p]
			t.reachedBy = append(t.reachedBy, v.id)
			if len(t.networks) > 0 && !reaches(v.networks, t.networks[0], routed) {
				add(connectors.Problem{EdgeSource: v.id, EdgeTarget: p, Code: connectors.CodePeerUnreachable,
					Message: "On OpenStack these two VMs are in networks that cannot reach each other. Connect them to the same network, or turn on internet access for both networks."})
			}
		}
	}
	if cyc := connectors.AddressCycle(vmIDs, peers); cyc != "" {
		add(connectors.Problem{NodeID: cyc, Code: connectors.CodeAddressCycle,
			Message: "VMs that get each other's address form a circle, so none could start first. Remove one of those connections."})
	}
	out.crossProblems = connectors.CheckCross(g, d, crossRules)
	for _, e := range connectors.CrossExports(d) {
		out.exports = append(out.exports, export{key: e.Key, vm: e.Target})
	}
	for i := range out.vms {
		v := &out.vms[i]
		env, missing := vmEnv(g, d, v)
		out.missingImport = out.missingImport || missing
		if len(env) == 0 {
			continue
		}
		expr, sensitive, err := connectors.StartupHCL(g.Blocks[v.id].Fields, env)
		if err != nil {
			add(fieldProblem(v.id, connectors.FieldStartupScript, err.Error()))
			continue
		}
		v.userDataExpr, v.sensitive = expr, sensitive
	}
	for i := range out.volumes {
		v := &out.volumes[i]
		switch targets := attachedTo[v.id]; {
		case len(targets) > 1:
			add(connectors.Problem{NodeID: v.id, Code: connectors.CodeVolumeTwoVMs,
				Message: "A volume can be attached to one VM only. Remove the extra connections."})
		case len(targets) == 1:
			v.vm = targets[0]
		}
	}

	// Settings. Only those the included blocks use are checked here; the
	// capability check has already checked every setting's type.
	if len(out.vms) > 0 {
		out.image = settingName(d.Settings, settingImage, add)
		flavors := map[string]string{}
		for i := range out.vms {
			size := out.vms[i].size
			if size == "" {
				continue // the capability check reported the size
			}
			if _, done := flavors[size]; !done {
				flavors[size] = settingName(d.Settings, flavorSetting[size], add)
			}
			out.vms[i].flavor = flavors[size]
		}
	}
	if out.anyRouted() || anyPublic(out.vms) {
		out.externalNetwork = settingName(d.Settings, settingExternalNetwork, add)
	}

	return out, problems
}

// reaches reports whether a VM on the networks from reaches an address on
// network to: it shares that network, or both are on the deployment's router.
func reaches(from []string, to string, routed map[string]bool) bool {
	for _, n := range from {
		if n == to || (routed[n] && routed[to]) {
			return true
		}
	}
	return false
}

// anyRouted reports whether the deployment needs its router.
func (m model) anyRouted() bool {
	for _, n := range m.networks {
		if n.routed {
			return true
		}
	}
	return false
}

// crossRules: OpenStack hands another cloud only a public VM's address, since
// it has no keys scoped to one resource here, and only its VMs take values
// from another cloud.
var crossRules = connectors.CrossRules{
	Cloud: "OpenStack",
	CanExport: func(class connectors.EdgeClass, target connectors.BlockKind) bool {
		return class == connectors.EdgeAddress && target == connectors.KindVM
	},
	ExportRefusal: "OpenStack can hand another cloud only a VM's address. Place this block on a cloud that can create a key for it.",
}

// vmEnv is what a VM's edges hand it at first boot: the address of each VM on
// this cloud it connects to, and every value another cloud returned for it (a
// placeholder until that cloud is built; missing then reports it).
func vmEnv(g connectors.Graph, d connectors.Deployment, v *vm) (env []connectors.EnvValue, missing bool) {
	for _, p := range v.peers {
		env = append(env, connectors.EnvValue{Name: connectors.EnvName(g.Blocks[p].Name, connectors.ValueAddress),
			Ref: ref(tInstance, addr(p), "access_ip_v4")})
	}
	imports, missing := connectors.ImportEnv(g, d, v.id)
	return append(env, imports...), missing
}

// value returns a field's value, or its default when it is not set.
func value(fields map[string]any, f connectors.FieldSpec) any {
	if v, ok := fields[f.Name]; ok && v != nil {
		return v
	}
	return f.Default
}

// settingName returns a setting that names something in the cloud, adding a
// problem when it is empty or not a plain name.
func settingName(settings map[string]any, f connectors.FieldSpec, add func(connectors.Problem)) string {
	s, ok := value(settings, f).(string)
	if !ok {
		return "" // the capability check reported the type
	}
	if strings.TrimSpace(s) == "" {
		add(connectors.Problem{Field: f.Name, Code: connectors.CodeMissingValue, Message: fmt.Sprintf("%s is required.", f.Title)})
		return ""
	}
	if !validCloudName(s) {
		add(connectors.Problem{Field: f.Name, Code: connectors.CodeInvalidValue,
			Message: fmt.Sprintf("%s must start with a letter or digit and use only letters, digits, spaces and . _ : ( ) [ ] + - (at most %d characters).", f.Title, maxCloudNameLength)})
		return ""
	}
	return s
}

func validCloudName(s string) bool {
	return len(s) <= maxCloudNameLength && cloudNamePattern.MatchString(s)
}

func fieldProblem(nodeID, field, msg string) connectors.Problem {
	return connectors.Problem{NodeID: nodeID, Field: field, Code: connectors.CodeInvalidValue, Message: msg}
}

// networkCIDR checks a network's address range (connectors.ParseCIDR) against
// what this connector builds: IPv4, with a prefix from /8 to /29.
func networkCIDR(s string) (string, error) {
	p, err := connectors.ParseCIDR(s)
	if err != nil {
		return "", err
	}
	if !p.Addr().Is4() || p.Bits() < minNetworkPrefix || p.Bits() > maxNetworkPrefix {
		return "", fmt.Errorf("On OpenStack a network's address range must be an IPv4 range from /%d to /%d, such as 10.0.0.0/24.", minNetworkPrefix, maxNetworkPrefix)
	}
	return p.String(), nil
}

func anyPublic(vms []vm) bool {
	for _, v := range vms {
		if v.public {
			return true
		}
	}
	return false
}
