package connectors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
)

// A split deployment places some of its blocks on other clouds of the same
// organization (docs/architecture/system-b-adapter-layer.md § Split
// deployments, ADR-020). Each cloud is one part of the deployment, applied in
// its own run task with only its own credentials. An edge whose two blocks sit
// on different parts is a cross-cloud edge, and it is one of two classes.

// EdgeClass is what a cross-cloud edge carries from its target's cloud to its
// source's cloud.
type EdgeClass string

// The edge classes.
const (
	// EdgeAddress hands the target's address to the source VM at first boot:
	// a VM to a VM ("the VM gets the other VM's address"). The target must
	// be reachable from the internet. No credential crosses.
	EdgeAddress EdgeClass = "address"
	// EdgePermission hands the source VM a key that can only read the
	// target, with the target's name and region: a VM to a bucket or a table
	// ("the VM can read the bucket"). The target's cloud creates the key.
	EdgePermission EdgeClass = "permission"
)

// CrossClass returns the class an edge between these kinds has when its two
// blocks sit on different clouds, or "" when such an edge cannot cross
// clouds: a VM and its network, a volume and its VM, a firewall and its VM
// are one cloud's wiring; a function reading a bucket or table, and an HTTP
// API sending to a function, are wired inside one cloud's IAM. Only a VM is a
// source across clouds, because only a VM takes values at first boot.
func CrossClass(source, target BlockKind) EdgeClass {
	if source != KindVM {
		return ""
	}
	switch target {
	case KindVM:
		return EdgeAddress
	case KindBucket, KindTable:
		return EdgePermission
	}
	return ""
}

// The names of the values a cross-cloud edge carries.
const (
	ValueAddress         = "address"
	ValueBucket          = "bucket"
	ValueTable           = "table"
	ValueRegion          = "region"
	ValueAccessKeyID     = "access_key_id"
	ValueSecretAccessKey = "secret_access_key"
)

// ExportValues are the values the target's cloud must return for a
// cross-cloud edge of this class to a block of this kind, in a fixed order.
func ExportValues(class EdgeClass, target BlockKind) []string {
	switch {
	case class == EdgeAddress && target == KindVM:
		return []string{ValueAddress}
	case class == EdgePermission && target == KindBucket:
		return []string{ValueBucket, ValueRegion, ValueAccessKeyID, ValueSecretAccessKey}
	case class == EdgePermission && target == KindTable:
		return []string{ValueTable, ValueRegion, ValueAccessKeyID, ValueSecretAccessKey}
	}
	return nil
}

// SensitiveValue reports whether a value of this name is a secret: it never
// reaches a log, a response, a preview, a Problem or an unmarked output.
func SensitiveValue(name string) bool { return name == ValueSecretAccessKey }

// CrossEdge is an edge of a split deployment whose two blocks sit on
// different clouds.
type CrossEdge struct {
	Source string    `json:"source"`
	Target string    `json:"target"`
	Class  EdgeClass `json:"class"`
	// Key names the edge in rendered Terraform, in outputs and in the cloud
	// names of what it creates (CrossKey).
	Key string `json:"key"`
}

// CrossKey is a cross-cloud edge's key: "x" and ten hex digits of the
// SHA-256 of its two node ids. It is a valid Terraform name part and cloud
// name part on every cloud.
func CrossKey(source, target string) string {
	sum := sha256.Sum256([]byte(source + "\x00" + target))
	return "x" + hex.EncodeToString(sum[:5])
}

// crossKeyPattern is the shape of CrossKey's result.
var crossKeyPattern = regexp.MustCompile(`^x[0-9a-f]{10}$`)

// ValidCrossKey reports whether k has CrossKey's shape.
func ValidCrossKey(k string) bool { return crossKeyPattern.MatchString(k) }

// Value is one value a cloud hands to the next.
type Value struct {
	Text      string
	Sensitive bool
}

// String hides a sensitive value.
func (v Value) String() string {
	if v.Sensitive {
		return Redacted
	}
	return v.Text
}

// GoString hides a sensitive value from %#v.
func (v Value) GoString() string { return v.String() }

// MarshalJSON hides a sensitive value.
func (v Value) MarshalJSON() ([]byte, error) { return json.Marshal(v.String()) }

// CrossImport is a cross-cloud edge whose source block is on this part, with
// the values its target's cloud returned. Values is empty at a preview, and at
// a plan before the target's cloud was first built; the connector then renders
// a placeholder for each value.
type CrossImport struct {
	CrossEdge
	Values map[string]Value
}

// Cross is what one part of a split deployment exchanges with the others. The
// part's graph is the whole diagram, and every block on another part is in
// the part's Deployment.Exclusions, so a connector still reads the other
// block's kind and name from the graph.
type Cross struct {
	// Exports are cross-cloud edges whose target block is on this part. The
	// part renders what each needs (a key that can only read the target) and
	// returns ExportValues for each as Outputs.
	Exports []CrossEdge
	// Imports are cross-cloud edges whose source block (a VM) is on this
	// part. The part hands their values to the VM at first boot.
	Imports []CrossImport
}

// Empty reports whether the part exchanges nothing.
func (x *Cross) Empty() bool { return x == nil || (len(x.Exports) == 0 && len(x.Imports) == 0) }

// Outputs are what one part's run returned for its exports: by edge key, then
// by value name.
type Outputs map[string]map[string]Value

// SecretValues returns every sensitive value, for redacting run logs.
func (o Outputs) SecretValues() []string {
	var out []string
	for _, vs := range o {
		for _, v := range vs {
			if v.Sensitive && v.Text != "" {
				out = append(out, v.Text)
			}
		}
	}
	return out
}

// CrossCloud is optional. A connector whose clouds can be one part of a split
// deployment implements it; the core finds it by type assertion, and refuses
// to place a block on, or keep the primary of a split on, a cloud whose
// connector lacks it.
//
// Each method reads d.Cross, which the core sets on every part of a split
// deployment and never elsewhere. Validate and Preview read it too: a
// connector that implements CrossCloud renders d.Cross in every method, and
// Preview renders a placeholder for every import value (no secret is in a
// preview). Plan and Apply on a part behave as PlanCross and ApplyCross
// without their Outputs.
type CrossCloud interface {
	// CrossCheck reports the cross-cloud edges of d.Cross this cloud cannot
	// serve: an export it cannot create (a key on a cloud with no scoped
	// keys, the address of a VM that is not public) or an import its VM
	// cannot take.
	CrossCheck(g Graph, d Deployment) []Problem
	// PlanCross plans the part and returns its exports' values as the part's
	// state holds them now: none before the part was first applied.
	PlanCross(ctx context.Context, run Run, g Graph, d Deployment) (PlanResult, Outputs, error)
	// ApplyCross applies the part and returns every export's ExportValues.
	ApplyCross(ctx context.Context, run Run, g Graph, d Deployment) (RunResult, Outputs, error)
}

// crossValuePattern is the alphabet of a value that crosses clouds: an IP
// address, a cloud name, a region, an access key or its secret. A value from
// a cloud's output is checked against it before any other cloud sees it, so
// it can sit in single quotes in a shell and inside an HCL string.
var crossValuePattern = regexp.MustCompile(`^[A-Za-z0-9._:/+=@-]+$`)

// maxCrossValue caps a value that crosses clouds, in bytes.
const maxCrossValue = 1024

// CheckCrossValue returns an error unless v may cross clouds. The error never
// repeats the value.
func CheckCrossValue(v string) error {
	if len(v) > maxCrossValue || !crossValuePattern.MatchString(v) {
		return errors.New("the value holds characters Clutter does not pass between clouds")
	}
	return nil
}

// CrossOutputName is the Terraform output that carries one export value. A
// connector renders it, and ParseCrossOutputs reads it back.
func CrossOutputName(key, value string) string { return "cross_" + key + "_" + value }

// outputJSON is one entry of `terraform output -json`.
type outputJSON struct {
	Sensitive bool            `json:"sensitive"`
	Value     json.RawMessage `json:"value"`
}

// ParseCrossOutputs reads the exports' values from `terraform output -json`.
// kinds maps each export's target node id to its kind. With complete set, a
// missing or empty value is an error (an apply returns every value); without
// it, missing values are left out (a plan before the first apply). Errors
// name outputs, never values.
func ParseCrossOutputs(raw []byte, exports []CrossEdge, kinds map[string]BlockKind, complete bool) (Outputs, error) {
	all := map[string]outputJSON{}
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &all); err != nil {
			return nil, errors.New("the Terraform outputs could not be read")
		}
	}
	out := Outputs{}
	for _, e := range exports {
		for _, name := range ExportValues(e.Class, kinds[e.Target]) {
			o, ok := all[CrossOutputName(e.Key, name)]
			var s string
			if ok && json.Unmarshal(o.Value, &s) != nil {
				return nil, fmt.Errorf("the output %s is not text", CrossOutputName(e.Key, name))
			}
			if s == "" {
				if complete {
					return nil, fmt.Errorf("the output %s is missing", CrossOutputName(e.Key, name))
				}
				continue
			}
			if err := CheckCrossValue(s); err != nil {
				return nil, fmt.Errorf("the output %s: %w", CrossOutputName(e.Key, name), err)
			}
			if out[e.Key] == nil {
				out[e.Key] = map[string]Value{}
			}
			out[e.Key][name] = Value{Text: s, Sensitive: o.Sensitive || SensitiveValue(name)}
		}
	}
	return out, nil
}

// AddressCycle returns a VM on a circle of VMs that get each other's address
// on one cloud (next maps a VM to the VMs it gets the address of), or "" when
// there is none. Such a circle cannot be built: each VM's user data needs
// another's address first. ids fixes the order the VMs are tried in.
func AddressCycle(ids []string, next map[string][]string) string {
	const (
		unseen = iota
		open
		closed
	)
	state := map[string]int{}
	var visit func(string) string
	visit = func(id string) string {
		state[id] = open
		for _, n := range next[id] {
			switch state[n] {
			case open:
				return n
			case unseen:
				if c := visit(n); c != "" {
					return c
				}
			}
		}
		state[id] = closed
		return ""
	}
	for _, id := range ids {
		if state[id] == unseen {
			if c := visit(id); c != "" {
				return c
			}
		}
	}
	return ""
}

// EnvName is the environment variable a VM's startup gets one value under:
// CLUTTER_, the target block's name in capitals with hyphens as underscores,
// and the value's name, such as CLUTTER_FILES_BUCKET or CLUTTER_DB_ADDRESS.
func EnvName(targetName, value string) string {
	n := strings.ToUpper(strings.ReplaceAll(targetName, "-", "_"))
	return "CLUTTER_" + n + "_" + strings.ToUpper(value)
}

// EnvValue is one value a VM's startup environment gets from an edge.
type EnvValue struct {
	// Name is EnvName's result.
	Name string
	// Text is a value another cloud returned, checked by CheckCrossValue.
	Text string
	// Ref, when set instead of Text, is a Terraform expression the connector
	// built from checked parts (a resource's attribute on the same cloud).
	Ref       string
	Sensitive bool
}

// placeholder stands in for an import value at a preview and at a plan
// before the target's cloud was built.
const placeholder = "known-after-the-other-cloud-is-built"

// Placeholder returns the text a connector renders for a value it does not
// have yet.
func Placeholder() string { return placeholder }

// refLength is what a Terraform reference counts for in the user data's
// length, before Terraform knows the value.
const refLength = 64

var envNamePattern = regexp.MustCompile(`^CLUTTER_[A-Z0-9_]{1,120}$`)

// errStartupTooLong is the user's message when the wrapper and the script do
// not fit together.
var errStartupTooLong = fmt.Errorf("With the values its connections hand it, this VM's startup script is too long: keep it under about %d KB.", (MaxTextField*3/4-1024)/1024)

// StartupHCL returns a Terraform expression for a VM's user data text when
// edges hand the VM values: a /bin/sh script that writes the values to
// /etc/clutter/connections.env (readable by root only), exports them, and then
// runs the block's own startup script (StartupUserData's rules), which reaches
// the expression only as base64. It returns "" when env is empty: the
// connector then renders the plain startup script. sensitive reports whether a
// value is a secret; the connector then wraps the argument in sensitive(), so
// Terraform never prints it. The error's message is safe to show the user.
func StartupHCL(fields map[string]any, env []EnvValue) (expr string, sensitive bool, err error) {
	if len(env) == 0 {
		return "", false, nil
	}
	script, err := StartupUserData(fields)
	if err != nil {
		return "", false, err
	}
	env = append([]EnvValue(nil), env...)
	sort.Slice(env, func(i, j int) bool { return env[i].Name < env[j].Name })
	var parts []string
	size := 0
	lit := func(s string) {
		parts = append(parts, hclsafe.Escape(s))
		size += len(s)
	}
	lit("#!/bin/sh\n# Clutter: the values this server's connections need, then its startup script.\numask 077\nmkdir -p /etc/clutter\ncat > /etc/clutter/connections.env <<'CLUTTER_ENV'\n")
	for i, v := range env {
		if !envNamePattern.MatchString(v.Name) {
			return "", false, fmt.Errorf("Clutter could not name a value for this VM.")
		}
		if i > 0 && env[i-1].Name == v.Name {
			return "", false, fmt.Errorf("Two of this VM's connections hand it a value named %s. Give their blocks names that differ by more than - and _.", v.Name)
		}
		lit(v.Name + "='")
		switch {
		case v.Ref != "":
			parts = append(parts, "${"+v.Ref+"}")
			size += refLength
		default:
			if err := CheckCrossValue(v.Text); err != nil {
				return "", false, fmt.Errorf("Clutter could not hand %s to this VM: %v.", v.Name, err)
			}
			lit(v.Text)
		}
		lit("'\n")
		sensitive = sensitive || v.Sensitive
	}
	lit("CLUTTER_ENV\nset -a\n. /etc/clutter/connections.env\nset +a\n")
	if script != "" {
		// The script is base64 text, so it holds no quote and no ${.
		lit("mkdir -p /var/lib/clutter\necho '" + script + "' | base64 -d > /var/lib/clutter/startup\nchmod 700 /var/lib/clutter/startup\nexec /var/lib/clutter/startup\n")
	}
	if size > MaxTextField {
		return "", false, errStartupTooLong
	}
	return `"` + strings.Join(parts, "") + `"`, sensitive, nil
}

// NewCrossEdge is the cross-cloud edge from source to target, keyed by CrossKey.
func NewCrossEdge(source, target string, class EdgeClass) CrossEdge {
	return CrossEdge{Source: source, Target: target, Class: class, Key: CrossKey(source, target)}
}

// ForPreview returns d with every import's values removed, so a preview of a
// part renders a placeholder for each and never holds another cloud's key.
func ForPreview(d Deployment) Deployment {
	if d.Cross == nil {
		return d
	}
	x := Cross{Exports: d.Cross.Exports}
	for _, imp := range d.Cross.Imports {
		imp.Values = nil
		x.Imports = append(x.Imports, imp)
	}
	d.Cross = &x
	return d
}

// CrossExports is d's exports, or none outside a split.
func CrossExports(d Deployment) []CrossEdge {
	if d.Cross == nil {
		return nil
	}
	return d.Cross.Exports
}

// ExportsWriter is where a run's Terraform outputs go: w for a part that
// exports something, nil otherwise (the run then prints no outputs).
func ExportsWriter(d Deployment, w io.Writer) io.Writer {
	if len(CrossExports(d)) == 0 {
		return nil
	}
	return w
}

// ReadExports reads a part's export values from its run's `terraform output
// -json` (ParseCrossOutputs), with complete as there. cloud names the cloud in
// the user error. The error holds no value.
func ReadExports(raw []byte, g Graph, d Deployment, complete bool, cloud string) (Outputs, error) {
	exports := CrossExports(d)
	if len(exports) == 0 {
		return Outputs{}, nil
	}
	kinds := map[string]BlockKind{}
	for _, e := range exports {
		kinds[e.Target] = g.Blocks[e.Target].Kind
	}
	out, err := ParseCrossOutputs(raw, exports, kinds, complete)
	if err != nil {
		return nil, &UserError{Message: cloud + " built the deployment but Clutter could not read what it hands the other clouds (" + err.Error() + "). Apply again."}
	}
	return out, nil
}

// ImportEnv is what the imports of d hand the VM vmID at first boot: every
// value another cloud returned, or a placeholder for one it has not returned
// yet (missing then reports it).
func ImportEnv(g Graph, d Deployment, vmID string) (env []EnvValue, missing bool) {
	if d.Cross == nil {
		return nil, false
	}
	for _, imp := range d.Cross.Imports {
		if imp.Source != vmID {
			continue
		}
		t := g.Blocks[imp.Target]
		for _, name := range ExportValues(imp.Class, t.Kind) {
			v, ok := imp.Values[name]
			if !ok || v.Text == "" {
				v, missing = Value{Text: Placeholder()}, true
			}
			env = append(env, EnvValue{Name: EnvName(t.Name, name), Text: v.Text, Sensitive: v.Sensitive || SensitiveValue(name)})
		}
	}
	return env, missing
}

// CrossRules are what one cloud can serve across clouds.
type CrossRules struct {
	// Cloud names the cloud in messages, such as "AWS".
	Cloud string
	// CanExport reports whether the cloud can hand another cloud an edge of
	// this class to a block of this kind.
	CanExport func(class EdgeClass, target BlockKind) bool
	// ExportRefusal is the message for an export CanExport refuses.
	ExportRefusal string
}

// CheckCross checks a part's cross-cloud edges against the cloud's rules
// (CrossCloud.CrossCheck): an export's target is a block this part builds and
// the cloud can export, and a VM whose address goes is public; an import's
// source is a VM this part builds, and its target's values are ones a cloud
// exports.
func CheckCross(g Graph, d Deployment, r CrossRules) []Problem {
	if d.Cross == nil {
		return nil
	}
	var out []Problem
	bad := func(e CrossEdge, msg string) {
		out = append(out, Problem{EdgeSource: e.Source, EdgeTarget: e.Target, Code: CodeCrossInvalid, Message: msg})
	}
	built := func(id string) (Block, bool) {
		b, ok := g.Blocks[id]
		return b, ok && !d.Excluded(id)
	}
	public := StandardField(KindVM, "public").Name
	for _, e := range d.Cross.Exports {
		t, ok := built(e.Target)
		switch {
		case !ValidCrossKey(e.Key):
			bad(e, "Clutter could not name this connection between clouds.")
		case !ok:
			bad(e, "This connection's block is not built on "+r.Cloud+" in this deployment.")
		case !r.CanExport(e.Class, t.Kind):
			bad(e, r.ExportRefusal)
		case e.Class == EdgeAddress && t.Fields[public] != true:
			out = append(out, Problem{NodeID: t.ID, Field: public, Code: CodeCrossNeedsPublic,
				Message: "A VM on another cloud gets this VM's address, so it must be reachable from the internet."})
		}
	}
	for _, imp := range d.Cross.Imports {
		s, built := built(imp.Source)
		t, ok := g.Blocks[imp.Target]
		switch {
		case !ValidCrossKey(imp.Key):
			bad(imp.CrossEdge, "Clutter could not name this connection between clouds.")
		case !built || s.Kind != KindVM:
			bad(imp.CrossEdge, "Only a VM on "+r.Cloud+" can take values from another cloud.")
		case !ok || ExportValues(imp.Class, t.Kind) == nil:
			bad(imp.CrossEdge, r.Cloud+" cannot take this block's values from another cloud.")
		}
	}
	return out
}
