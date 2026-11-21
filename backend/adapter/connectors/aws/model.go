package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"sort"
	"strings"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
	"github.com/google/uuid"
)

// Problem codes only this connector reports; the shared ones are in
// connectors.
const (
	CodeVMTwoNetworks    = "vm_on_two_networks"
	CodeVolumeNeedsVM    = "volume_needs_vm"
	CodeTooManyVolumes   = "too_many_volumes"
	CodeFirewallNetworks = "firewall_spans_networks"
	CodeAPIWithoutRoute  = "http_api_without_function"
)

// Name limits. AWS caps names per service; the lambda template appends
// "-lambda-role" (12) to the function name for an IAM role (64), the ec2
// template appends "-ssm-role" (9) to the Terraform address for an IAM role
// (64), and the s3 template appends "-" and 8 hex digits to the bucket name
// (63).
const (
	maxBucketName   = 63 - 9
	maxFunctionName = 64 - 12
	maxName         = 64
	maxNodePart     = 40
	maxIngressRules = 50
	maxVolumesPerVM = 11 // /dev/sdf to /dev/sdp
)

var (
	plainNodePart   = regexp.MustCompile(`^[A-Za-z0-9-]{1,40}$`)
	unsafeNodePart  = regexp.MustCompile(`[^A-Za-z0-9-]`)
	tableKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)
	// handlerPattern is a Lambda handler: "index.handler", "main",
	// "example.Handler::handleRequest". No quote, brace, dollar or space.
	handlerPattern   = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.:/-]{0,127}$`)
	unsafeBucketPart = regexp.MustCompile(`[^a-z0-9-]`)
)

// block is one included block with its fields resolved against the manifest's
// defaults and its two names.
type block struct {
	connectors.Block
	// Addr is the Terraform address: the deployment prefix and the node id.
	// A rename does not change it, so a rename never re-creates the resource.
	Addr string
	// Part is the node id as it appears in Addr, after the prefix.
	Part   string
	fields map[string]any
}

func (b *block) str(name string) string {
	s, _ := b.fields[name].(string)
	return s
}

func (b *block) boolean(name string) bool {
	v, _ := b.fields[name].(bool)
	return v
}

func (b *block) integer(name string) (int, bool) {
	n, ok := connectors.IntValue(b.fields[name])
	return int(n), ok
}

// physical returns a cloud-side name: the deployment prefix, a hyphen and the
// block's name, cut to max with a hash of the name so two long names stay apart.
func (m *model) physical(b *block, max int) string {
	full := m.d.Prefix + "-" + b.Name
	if len(full) <= max {
		return full
	}
	cut := max - len(m.d.Prefix) - 1 - 9
	return m.d.Prefix + "-" + strings.TrimRight(b.Name[:cut], "-_") + "-" + shortHash(b.Name)
}

// stableName is a cloud name built from the node id, not the block's name,
// for a resource whose name forces a new one and whose replacement is costly:
// a security group AWS will not delete while an instance uses it, and a key
// pair whose new name replaces the instance. A rename then changes only the
// resource's Name tag. A block name holds no dot, so this name is never one
// physical gives. Templates that use it ignore later changes to the name, so a
// resource named before it keeps its name.
func (m *model) stableName(b *block, what string) string {
	return m.d.Prefix + "." + b.Part + "." + what
}

// The table key a new table block starts with. A table with this key keeps
// its plain name (tableName).
var (
	defaultTableKey, _     = connectors.StandardField(connectors.KindTable, "key").Default.(string)
	defaultTableKeyType, _ = connectors.StandardField(connectors.KindTable, "key_type").Default.(string)
)

// tableName is a table's cloud name. DynamoDB fixes a table's key when it is
// made, so a changed key or key type replaces the table. The provider plans
// that replacement create-before-destroy whenever an API reaches the table
// through a function (A's API deployment sets create_before_destroy, and
// Terraform carries it to everything the deployment depends on), and the new
// table cannot take the old one's name. So a table whose key is not the
// default carries a hash of its key and key type, and any key change is a new
// name. The hash also makes a key type change visible: the provider compares
// attribute blocks by name only, so it never plans a type change on its own
// (hashicorp/terraform-provider-aws#22852).
func (m *model) tableName(b *block) string {
	key, keyType := b.str(fieldKey), b.str(fieldKeyType)
	if key == defaultTableKey && keyType == defaultTableKeyType {
		return m.physical(b, maxName)
	}
	return m.physical(b, maxName-9) + "-" + shortHash(key+":"+keyType)
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:4])
}

// nodePart turns a node id into the part of a Terraform address after the
// prefix. It holds no underscore, so the suffixes A's templates append with an
// underscore ("_role", "_ssm_role", "_subnet") can never make two blocks'
// addresses meet. Any other id (an underscore, over 40 characters, or a
// character translate would have refused) keeps only its safe characters and
// gets a hash of the whole id.
func nodePart(id string) string {
	if plainNodePart.MatchString(id) {
		return id
	}
	s := unsafeNodePart.ReplaceAllString(id, "-")
	if len(s) > maxNodePart-9 {
		s = s[:maxNodePart-9]
	}
	return s + "-" + shortHash(id)
}

// ingressRule is one aws_vpc_security_group_ingress_rule as the firewall
// template renders it. ICMP allows every type and code (-1), as "icmp" for an
// IPv4 source and "icmpv6" for an IPv6 one; IPv6 says which CIDR argument
// carries the source.
type ingressRule struct {
	Protocol string
	FromPort int
	ToPort   int
	CIDR     string
	IPv6     bool
}

// model is one deployment of one graph, as AWS builds it.
type model struct {
	d        connectors.Deployment
	region   string
	blocks   map[string]*block
	order    []string // included node ids, sorted
	problems []connectors.Problem

	vmNetwork   map[string]string   // vm -> network
	vmFirewalls map[string][]string // vm -> firewalls
	firewallVMs map[string][]string // firewall -> vms
	volumeVM    map[string]string   // volume -> vm
	vmVolumes   map[string][]string // vm -> volumes
	reads       [][2]string         // function -> bucket or table
	apiRoutes   map[string][]string // http api -> functions
	vmReads     [][2]string         // vm -> bucket or table, on this cloud
	vmPeers     map[string][]string // vm -> vms whose address it gets, on this cloud

	// g is the whole graph: an import's target is a block on another cloud,
	// excluded here, whose name and kind still name the VM's values.
	g connectors.Graph
	// cross is the part's cross-cloud edges, or nil outside a split.
	cross *connectors.Cross
	// crossProblems are what CrossCheck reports. A model with any renders
	// nothing.
	crossProblems []connectors.Problem

	instanceType string
	runtime      string
	architecture string
}

func (m *model) problem(p connectors.Problem) { m.problems = append(m.problems, p) }

func (m *model) fieldProblem(nodeID, field, msg string) {
	m.problem(connectors.Problem{NodeID: nodeID, Field: field, Code: connectors.CodeInvalidValue, Message: msg})
}

// newModel resolves g for d. It reports only what B's capability check, which
// the core runs beside it, cannot know: AWS's ranges and limits, the wiring
// rules, and names that would collide. Blocks of kinds this connector does
// not build are skipped here, because the capability check reports them.
func newModel(g connectors.Graph, d connectors.Deployment) *model {
	mf := manifest()
	m := &model{
		d:           d,
		region:      d.Connection.Region,
		blocks:      map[string]*block{},
		vmNetwork:   map[string]string{},
		vmFirewalls: map[string][]string{},
		firewallVMs: map[string][]string{},
		volumeVM:    map[string]string{},
		vmVolumes:   map[string][]string{},
		apiRoutes:   map[string][]string{},
		vmPeers:     map[string][]string{},
		g:           g,
		cross:       d.Cross,
	}
	if _, err := uuid.Parse(d.ID); err != nil {
		m.problem(connectors.Problem{Code: connectors.CodeDeploymentInvalid, Message: "The deployment's id is not valid."})
	}
	if !connectors.ValidPrefix(d.Prefix) {
		m.problem(connectors.Problem{Code: connectors.CodeDeploymentInvalid, Message: "The deployment's name prefix is not valid."})
	}
	if !mf.AllowsRegion(m.region) {
		m.problem(connectors.Problem{Code: connectors.CodeDeploymentInvalid, Message: "AWS does not offer this region through Clutter. Choose another region for the connection."})
	}
	m.settings(mf)

	for id, b := range g.Blocks {
		if d.Excluded(id) {
			continue
		}
		spec, ok := mf.Block(b.Kind)
		if !ok {
			continue
		}
		fields := map[string]any{}
		for _, f := range spec.Fields {
			if f.Default != nil {
				fields[f.Name] = f.Default
			}
		}
		for k, v := range b.Fields {
			if v != nil {
				fields[k] = v
			}
		}
		part := nodePart(id)
		m.blocks[id] = &block{Block: b, Addr: d.Prefix + "_" + part, Part: part, fields: fields}
		m.order = append(m.order, id)
	}
	sort.Strings(m.order)

	m.problems = append(m.problems, connectors.DuplicateNames(g, d)...)
	for _, id := range m.order {
		m.checkFields(m.blocks[id])
	}
	m.edges(g)
	m.crossProblems = connectors.CheckCross(g, d, crossRules)
	// Each VM's user data must hold the values its edges hand it. (A VM with
	// none has its script checked by checkFields.)
	for _, id := range m.order {
		if b := m.blocks[id]; b.Kind == connectors.KindVM {
			if env, _ := m.vmEnv(id); len(env) == 0 {
				continue
			}
			if _, err := m.userData(b); err != nil {
				m.fieldProblem(id, connectors.FieldStartupScript, err.Error())
			}
		}
	}
	return m
}

// crossRules: AWS hands another cloud a public VM's address, or a key that
// can only read a bucket or table, and its VMs take any such values.
var crossRules = connectors.CrossRules{
	Cloud: "AWS",
	CanExport: func(class connectors.EdgeClass, target connectors.BlockKind) bool {
		switch class {
		case connectors.EdgeAddress:
			return target == connectors.KindVM
		case connectors.EdgePermission:
			return target == connectors.KindBucket || target == connectors.KindTable
		}
		return false
	},
	ExportRefusal: "AWS cannot hand this block to another cloud.",
}

// userData is a VM's user data argument: the base64 of its startup script, as
// an HCL string, when no edge hands it values; otherwise an expression that
// builds the startup wrapper (connectors.StartupHCL), marked sensitive when a
// value is a secret. It is "" for a VM with neither.
func (m *model) userData(b *block) (string, error) {
	env, err := m.vmEnv(b.ID)
	if err != nil {
		return "", err
	}
	if len(env) == 0 {
		ud, err := connectors.StartupUserData(b.fields)
		if err != nil || ud == "" {
			return "", err
		}
		return `"` + ud + `"`, nil
	}
	expr, sensitive, err := connectors.StartupHCL(b.fields, env)
	if err != nil {
		return "", err
	}
	expr = "base64encode(" + expr + ")"
	if sensitive {
		expr = "sensitive(" + expr + ")"
	}
	return expr, nil
}

// vmEnv is what a VM's edges hand it at first boot: the address of each VM it
// connects to, the name and region of each bucket or table it reads, and every
// value another cloud returned for it (a placeholder until that cloud is
// built).
func (m *model) vmEnv(id string) ([]connectors.EnvValue, error) {
	var env []connectors.EnvValue
	for _, t := range m.vmPeers[id] {
		tb := m.blocks[t]
		env = append(env, connectors.EnvValue{Name: connectors.EnvName(tb.Name, connectors.ValueAddress), Ref: "aws_instance." + tb.Addr + ".private_ip"})
	}
	for _, r := range m.vmReads {
		if r[0] != id {
			continue
		}
		tb := m.blocks[r[1]]
		if tb.Kind == connectors.KindBucket {
			env = append(env, connectors.EnvValue{Name: connectors.EnvName(tb.Name, connectors.ValueBucket), Ref: "aws_s3_bucket." + tb.Addr + ".bucket"})
		} else {
			env = append(env, connectors.EnvValue{Name: connectors.EnvName(tb.Name, connectors.ValueTable), Ref: "aws_dynamodb_table." + tb.Addr + ".name"})
		}
		env = append(env, connectors.EnvValue{Name: connectors.EnvName(tb.Name, connectors.ValueRegion), Text: m.region})
	}
	imports, _ := connectors.ImportEnv(m.g, m.d, id)
	return append(env, imports...), nil
}

// functionEnv is what a function's "reads" edges hand it as environment
// variables: the name and region of each bucket or table, under the names a
// VM gets them (connectors.EnvName). Each value is a Terraform reference or
// an allow-listed region, inside the HCL string A's lambda template writes.
func (m *model) functionEnv(id string) map[string]string {
	env := map[string]string{}
	for _, r := range m.reads {
		if r[0] != id {
			continue
		}
		tb := m.blocks[r[1]]
		if tb.Kind == connectors.KindBucket {
			env[connectors.EnvName(tb.Name, connectors.ValueBucket)] = "${aws_s3_bucket." + tb.Addr + ".bucket}"
		} else {
			env[connectors.EnvName(tb.Name, connectors.ValueTable)] = "${aws_dynamodb_table." + tb.Addr + ".name}"
		}
		env[connectors.EnvName(tb.Name, connectors.ValueRegion)] = hclsafe.Escape(m.region)
	}
	return env
}

// missingImports reports whether an import has no value yet, which an apply
// must never render.
func (m *model) missingImports() bool {
	if m.cross == nil {
		return false
	}
	for _, imp := range m.cross.Imports {
		if _, missing := connectors.ImportEnv(m.g, m.d, imp.Source); missing {
			return true
		}
	}
	return false
}

func (m *model) settings(mf connectors.Manifest) {
	get := func(name string) string {
		if v, ok := m.d.Settings[name].(string); ok {
			return v
		}
		for _, s := range mf.Settings {
			if s.Name == name {
				if v, ok := s.Default.(string); ok {
					return v
				}
			}
		}
		return ""
	}
	m.instanceType = get(settingInstanceType)
	m.runtime = get(settingRuntime)
	m.architecture = get(settingArchitecture)
}

// vmInstanceType is the instance type a VM gets: the deployment's choice, or
// the type its size maps to.
func (m *model) vmInstanceType(b *block) string {
	if m.instanceType != "" {
		return m.instanceType
	}
	return sizeTypes[b.str(fieldSize)]
}

func (m *model) checkFields(b *block) {
	switch b.Kind {
	case connectors.KindVM:
		if _, err := connectors.StartupUserData(b.fields); err != nil {
			m.fieldProblem(b.ID, connectors.FieldStartupScript, err.Error())
		}
		if _, err := connectors.SSHPublicKey(b.fields); err != nil {
			m.fieldProblem(b.ID, connectors.FieldSSHKey, err.Error())
		}
	case connectors.KindNetwork:
		if _, err := networkCIDR(b.str(fieldCIDR)); err != nil {
			m.fieldProblem(b.ID, fieldCIDR, err.Error())
		}
	case connectors.KindFirewall:
		if _, err := connectors.ParseCIDR(b.str(fieldSourceCIDR)); err != nil {
			m.fieldProblem(b.ID, fieldSourceCIDR, err.Error())
		}
		if _, err := ingressRules(b.str(fieldIngress)); err != nil {
			m.fieldProblem(b.ID, fieldIngress, err.Error())
		}
	case connectors.KindVolume:
		if n, ok := b.integer(fieldSizeGB); !ok || n < 1 || n > 16384 {
			m.fieldProblem(b.ID, fieldSizeGB, "The size must be from 1 to 16384 GB.")
		}
	case connectors.KindFunction:
		if n, ok := b.integer(fieldMemoryMB); !ok || n < 128 || n > 10240 {
			m.fieldProblem(b.ID, fieldMemoryMB, "The memory must be from 128 to 10240 MB.")
		}
		if n, ok := b.integer(fieldTimeoutS); !ok || n < 1 || n > 900 {
			m.fieldProblem(b.ID, fieldTimeoutS, "The time limit must be from 1 to 900 seconds.")
		}
		if h := b.str(connectors.FieldHandler); h != "" && !handlerPattern.MatchString(h) {
			m.fieldProblem(b.ID, connectors.FieldHandler, "The handler must start with a letter, digit or underscore and use only letters, digits and _ . : / - (at most 128), such as index.handler.")
		}
		if b.Code != nil {
			if _, err := b.Code.Base64SHA256(); err != nil {
				m.fieldProblem(b.ID, "code", "Clutter's record of this function's code is not valid. Upload the code again.")
			}
		}
	case connectors.KindTable:
		if !tableKeyPattern.MatchString(b.str(fieldKey)) {
			m.fieldProblem(b.ID, fieldKey, "The key name must start with a letter and use only letters, digits and underscores (at most 64).")
		}
	}
}

// The messages below are shown to the user as problems.
var (
	errNetworkCIDR      = errors.New("On AWS a network's address range must be an IPv4 range from /16 to /27, such as 10.0.0.0/24.")
	errFirewallNetworks = errors.New("On AWS a firewall applies within one network, but this one applies to VMs in different networks. Use one firewall per network.")
)

// networkCIDR checks a network's address range (connectors.ParseCIDR)
// against AWS's VPC limits: IPv4, /16 to /27. The network splits its range
// into two subnets (networkHalves), and AWS's smallest subnet is a /28.
func networkCIDR(s string) (netip.Prefix, error) {
	p, err := connectors.ParseCIDR(s)
	if err != nil {
		return netip.Prefix{}, err
	}
	if !p.Addr().Is4() || p.Bits() < 16 || p.Bits() > 27 {
		return netip.Prefix{}, errNetworkCIDR
	}
	return p, nil
}

// networkHalves splits a network's range into its two subnets: the lower half
// for private VMs, the upper half for public VMs and the NAT gateway.
func networkHalves(p netip.Prefix) (private, public netip.Prefix) {
	private = netip.PrefixFrom(p.Addr(), p.Bits()+1)
	a := p.Addr().As4()
	host := 32 - (p.Bits() + 1)
	n := uint32(a[0])<<24 | uint32(a[1])<<16 | uint32(a[2])<<8 | uint32(a[3])
	n += 1 << host
	public = netip.PrefixFrom(netip.AddrFrom4([4]byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}), p.Bits()+1)
	return private, public
}

// ingressRules reads a firewall's ingress field (connectors.ParseIngress)
// and holds it to the number of rules this connector renders for one
// security group.
func ingressRules(s string) ([]connectors.IngressRule, error) {
	rules, err := connectors.ParseIngress(s)
	if err != nil {
		return nil, err
	}
	if len(rules) > maxIngressRules {
		return nil, fmt.Errorf("On AWS a firewall can allow at most %d entries.", maxIngressRules)
	}
	return rules, nil
}

// firewallRules turns a firewall's ingress field and source range into the
// security group's ingress rules. An empty list allows nothing in.
func firewallRules(ingress, source string) ([]ingressRule, error) {
	src, err := connectors.ParseCIDR(source)
	if err != nil {
		return nil, err
	}
	rules, err := ingressRules(ingress)
	if err != nil {
		return nil, err
	}
	ipv6 := src.Addr().Is6()
	out := make([]ingressRule, 0, len(rules))
	for _, r := range rules {
		rule := ingressRule{Protocol: r.Protocol, FromPort: r.FromPort, ToPort: r.ToPort, CIDR: src.String(), IPv6: ipv6}
		if r.Protocol == connectors.ProtocolICMP {
			rule.FromPort, rule.ToPort = -1, -1
			if ipv6 {
				rule.Protocol = "icmpv6"
			}
		}
		out = append(out, rule)
	}
	return out, nil
}

// edges indexes the included edges and checks the wiring AWS needs.
func (m *model) edges(g connectors.Graph) {
	for _, e := range g.Edges {
		s, t := m.blocks[e.Source], m.blocks[e.Target]
		if s == nil || t == nil {
			continue // a dropped edge; the capability check notes it
		}
		switch {
		case s.Kind == connectors.KindVM && t.Kind == connectors.KindNetwork:
			if prev, ok := m.vmNetwork[s.ID]; ok && prev != t.ID {
				m.problem(connectors.Problem{NodeID: s.ID, Code: CodeVMTwoNetworks, Message: "On AWS a VM runs in one network. Connect it to only one."})
				continue
			}
			m.vmNetwork[s.ID] = t.ID
		case s.Kind == connectors.KindVolume && t.Kind == connectors.KindVM:
			if prev, ok := m.volumeVM[s.ID]; ok && prev != t.ID {
				m.problem(connectors.Problem{NodeID: s.ID, Code: connectors.CodeVolumeTwoVMs, Message: "On AWS a volume is attached to one VM. Connect it to only one."})
				continue
			}
			m.volumeVM[s.ID] = t.ID
			m.vmVolumes[t.ID] = append(m.vmVolumes[t.ID], s.ID)
		case s.Kind == connectors.KindFirewall && t.Kind == connectors.KindVM:
			m.firewallVMs[s.ID] = append(m.firewallVMs[s.ID], t.ID)
			m.vmFirewalls[t.ID] = append(m.vmFirewalls[t.ID], s.ID)
		case s.Kind == connectors.KindFunction && (t.Kind == connectors.KindBucket || t.Kind == connectors.KindTable):
			m.reads = append(m.reads, [2]string{s.ID, t.ID})
		case s.Kind == connectors.KindHTTPAPI && t.Kind == connectors.KindFunction:
			m.apiRoutes[s.ID] = append(m.apiRoutes[s.ID], t.ID)
		case s.Kind == connectors.KindVM && t.Kind == connectors.KindVM:
			m.vmPeers[s.ID] = append(m.vmPeers[s.ID], t.ID)
		case s.Kind == connectors.KindVM && (t.Kind == connectors.KindBucket || t.Kind == connectors.KindTable):
			m.vmReads = append(m.vmReads, [2]string{s.ID, t.ID})
		}
	}
	for _, list := range []map[string][]string{m.vmFirewalls, m.firewallVMs, m.vmVolumes, m.apiRoutes, m.vmPeers} {
		for k := range list {
			sort.Strings(list[k])
		}
	}
	for _, rs := range [][][2]string{m.reads, m.vmReads} {
		sort.Slice(rs, func(i, j int) bool {
			if rs[i][0] != rs[j][0] {
				return rs[i][0] < rs[j][0]
			}
			return rs[i][1] < rs[j][1]
		})
	}
	if cyc := connectors.AddressCycle(m.order, m.vmPeers); cyc != "" {
		m.problem(connectors.Problem{NodeID: cyc, Code: connectors.CodeAddressCycle,
			Message: "VMs that get each other's address form a circle, so none could start first. Remove one of those connections."})
	}

	for _, id := range m.order {
		b := m.blocks[id]
		switch b.Kind {
		case connectors.KindVolume:
			if _, ok := m.volumeVM[id]; !ok {
				m.problem(connectors.Problem{NodeID: id, Code: CodeVolumeNeedsVM, Message: "On AWS a volume must be attached to a VM. Connect it to one."})
			}
		case connectors.KindVM:
			if len(m.vmVolumes[id]) > maxVolumesPerVM {
				m.problem(connectors.Problem{NodeID: id, Code: CodeTooManyVolumes, Message: fmt.Sprintf("Clutter attaches at most %d volumes to one VM.", maxVolumesPerVM)})
			}
			// Each network is its own VPC, and a VM with none is in the
			// account's default VPC: two VPCs do not reach each other.
			for _, peer := range m.vmPeers[id] {
				if m.vmNetwork[id] != m.vmNetwork[peer] {
					m.problem(connectors.Problem{EdgeSource: id, EdgeTarget: peer, Code: connectors.CodePeerUnreachable,
						Message: "On AWS these two VMs are in different networks, which cannot reach each other. Connect them to the same network."})
				}
			}
		case connectors.KindFirewall:
			if _, err := m.firewallNetwork(id); err != nil {
				m.problem(connectors.Problem{NodeID: id, Code: CodeFirewallNetworks, Message: err.Error()})
			}
		case connectors.KindHTTPAPI:
			if len(m.apiRoutes[id]) == 0 {
				m.problem(connectors.Problem{NodeID: id, Code: CodeAPIWithoutRoute, Message: "On AWS an HTTP API needs at least one function to send requests to. Connect it to a function."})
			}
		}
	}
}

// firewallNetwork returns the network a firewall's VMs share, or "" when they
// are in the account's default network. A security group belongs to one VPC.
func (m *model) firewallNetwork(id string) (string, error) {
	net, first := "", true
	for _, vm := range m.firewallVMs[id] {
		n := m.vmNetwork[vm]
		if !first && n != net {
			return "", errFirewallNetworks
		}
		net, first = n, false
	}
	return net, nil
}

// networkVMs reports whether the network holds a VM reachable from the
// internet, and one that is not.
func (m *model) networkVMs(id string) (public, private bool) {
	for vm, net := range m.vmNetwork {
		if net != id {
			continue
		}
		if m.blocks[vm].boolean(fieldPublic) {
			public = true
		} else {
			private = true
		}
	}
	return public, private
}

// networkNAT reports whether the network gets a NAT gateway: its private
// VMs may reach the internet, and it has one.
func (m *model) networkNAT(id string) bool {
	_, private := m.networkVMs(id)
	return private && m.blocks[id].boolean(fieldPrivateInternet)
}

// vmSubnet is the Terraform address of the subnet a VM runs in, or "" for
// the account's default VPC.
func (m *model) vmSubnet(id string) string {
	net, ok := m.vmNetwork[id]
	if !ok {
		return ""
	}
	if m.blocks[id].boolean(fieldPublic) {
		return m.blocks[net].Addr + "_public_subnet"
	}
	return m.blocks[net].Addr + "_subnet"
}

// vmZone is the availability zone a VM with volumes runs in, as an HCL
// expression its volumes read. It never reads the instance, so a replaced VM
// keeps its volumes: a volume's zone forces a new volume when it changes, and
// a replaced instance's zone is unknown until it exists. A VM in a network
// runs in its subnet's zone. A VM in the account's default VPC is pinned to
// the zone the network template picks (the region's first available one),
// because AWS otherwise picks a zone for each new instance (vmPinsZone).
func (m *model) vmZone(id string) string {
	if subnet := m.vmSubnet(id); subnet != "" {
		return "aws_subnet." + subnet + ".availability_zone"
	}
	return "data.aws_availability_zones." + m.blocks[id].Addr + "_azs.names[0]"
}

// vmPinsZone reports whether a VM needs its zone pinned (vmZone): it has
// volumes and no network.
func (m *model) vmPinsZone(id string) bool {
	_, inNetwork := m.vmNetwork[id]
	return !inNetwork && len(m.vmVolumes[id]) > 0
}

// peersOf returns the VMs on this cloud that get a VM's address, sorted: the
// VMs its peer security group lets in.
func (m *model) peersOf(id string) []string {
	var out []string
	for _, src := range m.order {
		for _, t := range m.vmPeers[src] {
			if t == id {
				out = append(out, src)
			}
		}
	}
	return out
}

// deviceName gives each volume on a VM its own device, /dev/sdf upward.
func (m *model) deviceName(volumeID string) string {
	for i, v := range m.vmVolumes[m.volumeVM[volumeID]] {
		if v == volumeID {
			return fmt.Sprintf("/dev/sd%c", 'f'+i)
		}
	}
	return "/dev/sdf"
}

// bucketName is a bucket's cloud name before the template's random suffix: S3
// takes only lowercase letters, digits and hyphens.
func (m *model) bucketName(b *block) string {
	return unsafeBucketPart.ReplaceAllString(m.physical(b, maxBucketName), "-")
}
