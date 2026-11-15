package connectors

import (
	"errors"
	"fmt"
	"math"
	"regexp"
)

// Treatment is how a connector treats a building block (ADR-001).
type Treatment string

// The treatments.
const (
	TreatmentAsIs     Treatment = "as-is"
	TreatmentExtended Treatment = "extended"
	TreatmentRemoved  Treatment = "removed"
	TreatmentNew      Treatment = "new"
)

// FieldType is the type of a block field or a deployment setting.
type FieldType string

// The field types. An enum field takes one of its Allowed values. A text
// field is multi-line text, such as a script, and the editor shows it in a
// text area.
const (
	FieldString FieldType = "string"
	FieldInt    FieldType = "int"
	FieldBool   FieldType = "bool"
	FieldEnum   FieldType = "enum"
	FieldText   FieldType = "text"
)

// MaxStringField caps a string field's length.
const MaxStringField = 256

// MaxTextField caps a text field's length in bytes: 16 KB, the most user data
// AWS accepts for one instance.
const MaxTextField = 16384

// Manifest is everything a connector accepts. It is the one owner of a
// target's blocks, fields, defaults, allowed values and regions; the catalog
// route, the palette and the config panel read it.
type Manifest struct {
	Connector string      `json:"connector"`
	Title     string      `json:"title"`
	Version   string      `json:"version"`
	Regions   []string    `json:"regions,omitempty"`
	Blocks    []BlockSpec `json:"blocks"`
	Edges     []EdgeSpec  `json:"edges"`
	// Settings are the per-deployment settings, such as an image or a flavor.
	Settings []FieldSpec `json:"settings"`
	// Display is how the UI names and marks the cloud.
	Display Display `json:"display"`
}

// Display is how the UI shows a connector: a short name for tight places
// (a badge, a "View as" switch) and the key of its logo. A key is not a path:
// the frontend resolves it to public/clouds/<connector id>/<key>.svg, and B's
// core never reads it.
type Display struct {
	ShortName string `json:"short_name"`
	Logo      string `json:"logo,omitempty"`
}

// BlockDisplay is how the UI shows a block on one cloud: the cloud's own name
// for the service (such as "EC2" or "Server (Nova)") and the key of its icon.
// An empty icon means the block's neutral icon.
type BlockDisplay struct {
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

// BlockSpec is one block a connector declares.
type BlockSpec struct {
	Kind      BlockKind `json:"kind"`
	Treatment Treatment `json:"treatment"`
	Title     string    `json:"title"`
	// Summary says in a few plain words what the block is for, such as "A
	// private area for servers". The palette shows it under the title.
	// StandardBlock sets it from the one owner, standardSummaries.
	Summary string      `json:"summary,omitempty"`
	Fields  []FieldSpec `json:"fields,omitempty"`
	// Display is set on every block the connector builds (the contract tests
	// check it) and never on a removed one.
	Display *BlockDisplay `json:"display,omitempty"`
}

// WithDisplay returns b showing as name, with the icon key icon ("" for the
// block's neutral icon).
func (b BlockSpec) WithDisplay(name, icon string) BlockSpec {
	b.Display = &BlockDisplay{Name: name, Icon: icon}
	return b
}

// FieldSpec is one field of a block, or one deployment setting.
type FieldSpec struct {
	Name     string    `json:"name"`
	Title    string    `json:"title"`
	Type     FieldType `json:"type"`
	Required bool      `json:"required"`
	Default  any       `json:"default,omitempty"`
	Allowed  []string  `json:"allowed,omitempty"`
	// Help is one short line the editor shows with the field, for a field
	// whose title alone does not say what it does.
	Help string `json:"help,omitempty"`
	// AppliesTo, on a deployment setting, names the block kinds the setting
	// is about (a function runtime is about functions), so the editor shows it
	// only for a diagram that has one. Empty means it is about every
	// deployment. A block field leaves it empty.
	AppliesTo []BlockKind `json:"applies_to,omitempty"`
}

// EdgeSpec is a pair of block kinds a connector can wire together.
type EdgeSpec struct {
	Source  BlockKind `json:"source"`
	Target  BlockKind `json:"target"`
	Meaning string    `json:"meaning"`
	// Why says in plain words what the link is good for, for a person with
	// no cloud background, such as "Your app on the server can look up data
	// such as users or orders." The editor shows it when a line is hovered.
	Why string `json:"why,omitempty"`
	// SingleTarget means a source block links to at most one block of the
	// target kind, and Rule says so in one short line ("A volume attaches to
	// one VM."). The editor refuses a second such line with Rule; the
	// connector's Validate refuses it too.
	SingleTarget bool   `json:"single_target,omitempty"`
	Rule         string `json:"rule,omitempty"`
}

// Block returns the spec for kind.
func (m Manifest) Block(kind BlockKind) (BlockSpec, bool) {
	for _, b := range m.Blocks {
		if b.Kind == kind {
			return b, true
		}
	}
	return BlockSpec{}, false
}

// AllowsEdge reports whether the manifest wires source to target.
func (m Manifest) AllowsEdge(source, target BlockKind) bool {
	for _, e := range m.Edges {
		if e.Source == source && e.Target == target {
			return true
		}
	}
	return false
}

// AllowsRegion reports whether region is accepted. A manifest with no region
// list accepts any region name RegionPattern allows.
func (m Manifest) AllowsRegion(region string) bool {
	if !RegionPattern.MatchString(region) {
		return false
	}
	if len(m.Regions) == 0 {
		return true
	}
	for _, r := range m.Regions {
		if r == region {
			return true
		}
	}
	return false
}

var (
	// ConnectorIDPattern is the shape of a connector id.
	ConnectorIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,40}$`)
	// KindPattern is the shape of a block kind.
	KindPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
	// FieldNamePattern is the shape of a field or setting name.
	FieldNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	// RegionPattern is the shape of a region name.
	RegionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
	// IconKeyPattern is the shape of a logo or icon key. It has no dot or
	// slash, so a key can only name a file inside its connector's folder.
	IconKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)
)

// The longest display names. They keep badges and node labels on one line.
const (
	MaxShortName   = 24
	MaxDisplayName = 48
)

// The longest plain-words texts: a block's summary sits on one palette line,
// and an edge's why and rule fit one line of a small card.
const (
	MaxSummary  = 48
	MaxEdgeText = 120
)

// Check returns an error unless the manifest is well formed: a valid id and
// title, a treatment for every standard block, new kinds marked new, valid
// fields and defaults, edges between declared blocks, unique regions, and
// display names and keys of the right shape.
func (m Manifest) Check() error {
	var errs []error
	if !ConnectorIDPattern.MatchString(m.Connector) {
		errs = append(errs, fmt.Errorf("connector id %q is not valid", m.Connector))
	}
	if m.Title == "" {
		errs = append(errs, errors.New("title is empty"))
	}
	if m.Version == "" {
		errs = append(errs, errors.New("version is empty"))
	}
	errs = append(errs, m.Display.check()...)

	seen := map[BlockKind]Treatment{}
	for _, b := range m.Blocks {
		if !KindPattern.MatchString(string(b.Kind)) {
			errs = append(errs, fmt.Errorf("block kind %q is not valid", b.Kind))
			continue
		}
		if _, dup := seen[b.Kind]; dup {
			errs = append(errs, fmt.Errorf("block %q is declared twice", b.Kind))
			continue
		}
		seen[b.Kind] = b.Treatment
		switch b.Treatment {
		case TreatmentAsIs, TreatmentExtended, TreatmentRemoved:
			if !IsStandard(b.Kind) {
				errs = append(errs, fmt.Errorf("block %q is not standard, so its treatment must be %q", b.Kind, TreatmentNew))
			}
		case TreatmentNew:
			if IsStandard(b.Kind) {
				errs = append(errs, fmt.Errorf("block %q is standard and cannot be %q", b.Kind, TreatmentNew))
			}
		default:
			errs = append(errs, fmt.Errorf("block %q has unknown treatment %q", b.Kind, b.Treatment))
		}
		if b.Title == "" {
			errs = append(errs, fmt.Errorf("block %q has no title", b.Kind))
		}
		if b.Treatment == TreatmentRemoved && len(b.Fields) > 0 {
			errs = append(errs, fmt.Errorf("removed block %q declares fields", b.Kind))
		}
		if len(b.Summary) > MaxSummary {
			errs = append(errs, fmt.Errorf("block %q summary is longer than %d characters", b.Kind, MaxSummary))
		}
		if b.Display != nil {
			if b.Treatment == TreatmentRemoved {
				errs = append(errs, fmt.Errorf("removed block %q declares a display", b.Kind))
			}
			errs = append(errs, b.Display.check(b.Kind)...)
		}
		errs = append(errs, checkFields(fmt.Sprintf("block %q", b.Kind), b.Fields)...)
	}
	for _, k := range StandardBlocks() {
		if _, ok := seen[k]; !ok {
			errs = append(errs, fmt.Errorf("standard block %q has no treatment", k))
		}
	}

	type pair struct{ s, t BlockKind }
	edges := map[pair]bool{}
	for _, e := range m.Edges {
		for _, k := range []BlockKind{e.Source, e.Target} {
			t, ok := seen[k]
			if !ok {
				errs = append(errs, fmt.Errorf("edge %s->%s names undeclared block %q", e.Source, e.Target, k))
			} else if t == TreatmentRemoved {
				errs = append(errs, fmt.Errorf("edge %s->%s names removed block %q", e.Source, e.Target, k))
			}
		}
		if len(e.Why) > MaxEdgeText || len(e.Rule) > MaxEdgeText {
			errs = append(errs, fmt.Errorf("edge %s->%s has a why or rule longer than %d characters", e.Source, e.Target, MaxEdgeText))
		}
		if e.SingleTarget != (e.Rule != "") {
			errs = append(errs, fmt.Errorf("edge %s->%s needs a rule exactly when it allows a single target", e.Source, e.Target))
		}
		p := pair{e.Source, e.Target}
		if edges[p] {
			errs = append(errs, fmt.Errorf("edge %s->%s is declared twice", e.Source, e.Target))
		}
		edges[p] = true
	}

	errs = append(errs, checkFields("settings", m.Settings)...)
	for _, f := range m.Settings {
		for _, k := range f.AppliesTo {
			if _, ok := seen[k]; !ok {
				errs = append(errs, fmt.Errorf("setting %q applies to undeclared block %q", f.Name, k))
			}
		}
	}
	for _, b := range m.Blocks {
		for _, f := range b.Fields {
			if len(f.AppliesTo) > 0 {
				errs = append(errs, fmt.Errorf("block %q: field %q names blocks it applies to, which only a setting does", b.Kind, f.Name))
			}
		}
	}

	regions := map[string]bool{}
	for _, r := range m.Regions {
		if !RegionPattern.MatchString(r) {
			errs = append(errs, fmt.Errorf("region %q is not valid", r))
		}
		if regions[r] {
			errs = append(errs, fmt.Errorf("region %q is listed twice", r))
		}
		regions[r] = true
	}
	return errors.Join(errs...)
}

func (d Display) check() []error {
	var errs []error
	if len(d.ShortName) > MaxShortName {
		errs = append(errs, fmt.Errorf("display short name is longer than %d characters", MaxShortName))
	}
	if d.Logo != "" {
		if d.ShortName == "" {
			errs = append(errs, errors.New("display has a logo but no short name"))
		}
		if !IconKeyPattern.MatchString(d.Logo) {
			errs = append(errs, fmt.Errorf("display logo key %q is not valid", d.Logo))
		}
	}
	return errs
}

func (d BlockDisplay) check(kind BlockKind) []error {
	var errs []error
	if d.Name == "" {
		errs = append(errs, fmt.Errorf("block %q has a display without a name", kind))
	}
	if len(d.Name) > MaxDisplayName {
		errs = append(errs, fmt.Errorf("block %q display name is longer than %d characters", kind, MaxDisplayName))
	}
	if d.Icon != "" && !IconKeyPattern.MatchString(d.Icon) {
		errs = append(errs, fmt.Errorf("block %q display icon key %q is not valid", kind, d.Icon))
	}
	return errs
}

func checkFields(owner string, fields []FieldSpec) []error {
	var errs []error
	names := map[string]bool{}
	for _, f := range fields {
		if !FieldNamePattern.MatchString(f.Name) {
			errs = append(errs, fmt.Errorf("%s: field name %q is not valid", owner, f.Name))
		}
		if f.Name == "name" {
			errs = append(errs, fmt.Errorf("%s: field %q is reserved for the block's name", owner, f.Name))
		}
		if names[f.Name] {
			errs = append(errs, fmt.Errorf("%s: field %q is declared twice", owner, f.Name))
		}
		names[f.Name] = true
		if f.Title == "" {
			errs = append(errs, fmt.Errorf("%s: field %q has no title", owner, f.Name))
		}
		switch f.Type {
		case FieldString, FieldInt, FieldBool, FieldText:
			if len(f.Allowed) > 0 {
				errs = append(errs, fmt.Errorf("%s: field %q lists allowed values but is not an enum", owner, f.Name))
			}
		case FieldEnum:
			if len(f.Allowed) == 0 {
				errs = append(errs, fmt.Errorf("%s: enum field %q has no allowed values", owner, f.Name))
			}
		default:
			errs = append(errs, fmt.Errorf("%s: field %q has unknown type %q", owner, f.Name, f.Type))
		}
		if f.Default != nil {
			if err := CheckValue(f, f.Default); err != nil {
				errs = append(errs, fmt.Errorf("%s: field %q default: %w", owner, f.Name, err))
			}
		}
	}
	return errs
}

// CheckValue returns an error unless v is a valid value for f. Numbers may be
// float64, as JSON decodes them, when they are whole.
func CheckValue(f FieldSpec, v any) error {
	switch f.Type {
	case FieldString:
		s, ok := v.(string)
		if !ok {
			return errors.New("must be text")
		}
		if len(s) > MaxStringField {
			return fmt.Errorf("must be at most %d characters", MaxStringField)
		}
	case FieldText:
		s, ok := v.(string)
		if !ok {
			return errors.New("must be text")
		}
		if len(s) > MaxTextField {
			return fmt.Errorf("must be at most %d bytes", MaxTextField)
		}
	case FieldInt:
		if _, ok := IntValue(v); !ok {
			return errors.New("must be a whole number")
		}
	case FieldBool:
		if _, ok := v.(bool); !ok {
			return errors.New("must be true or false")
		}
	case FieldEnum:
		s, ok := v.(string)
		if !ok {
			return errors.New("must be one of the listed values")
		}
		for _, a := range f.Allowed {
			if a == s {
				return nil
			}
		}
		return errors.New("must be one of the listed values")
	default:
		return fmt.Errorf("has unknown type %q", f.Type)
	}
	return nil
}

// IntValue reads a whole number as JSON (float64) or Go code (int, int64)
// gives it. A float64 must be whole and at most 2^53 from zero, the range
// JSON numbers hold exactly. A connector checks its cloud's range after.
func IntValue(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		if n == math.Trunc(n) && math.Abs(n) <= 1<<53 {
			return int64(n), true
		}
	}
	return 0, false
}

// CredentialKind says whether a connection borrows access or stores a secret
// (ADR-016).
type CredentialKind string

// The credential kinds.
const (
	CredentialBorrowed CredentialKind = "borrowed"
	CredentialStored   CredentialKind = "stored"
)

// CredentialSpec is what a connection supplies, and how the UI asks for it.
type CredentialSpec struct {
	Kind   CredentialKind    `json:"kind"`
	Fields []CredentialField `json:"fields"`
	// LinkedOnly means a connection is made only from an account the
	// organization linked (Linker), never from pasted credentials: a pasted
	// value could belong to another organization's link. Fields then name
	// what LinkedSecrets returns, and the UI does not ask for them.
	LinkedOnly bool `json:"linked_only"`
}

// CredentialField is one value a connection supplies.
type CredentialField struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Help     string `json:"help,omitempty"`
	Secret   bool   `json:"secret"`
	Required bool   `json:"required"`
}

// MaxCredentialValue caps one credential value's length.
const MaxCredentialValue = 8192

// Check returns an error unless the spec is well formed.
func (s CredentialSpec) Check() error {
	var errs []error
	if s.Kind != CredentialBorrowed && s.Kind != CredentialStored {
		errs = append(errs, fmt.Errorf("credential kind %q is not valid", s.Kind))
	}
	if len(s.Fields) == 0 {
		errs = append(errs, errors.New("credential spec has no fields"))
	}
	names := map[string]bool{}
	for _, f := range s.Fields {
		if !FieldNamePattern.MatchString(f.Name) {
			errs = append(errs, fmt.Errorf("credential field name %q is not valid", f.Name))
		}
		if names[f.Name] {
			errs = append(errs, fmt.Errorf("credential field %q is declared twice", f.Name))
		}
		names[f.Name] = true
		if f.Label == "" {
			errs = append(errs, fmt.Errorf("credential field %q has no label", f.Name))
		}
	}
	return errors.Join(errs...)
}

// CheckSupplied returns a user-facing error unless supplied holds every
// required field, no unknown field, and no value over MaxCredentialValue. The
// error names fields, never values.
func (s CredentialSpec) CheckSupplied(supplied map[string]string) error {
	known := map[string]bool{}
	for _, f := range s.Fields {
		known[f.Name] = true
		if f.Required && supplied[f.Name] == "" {
			return &UserError{Message: fmt.Sprintf("The credential field %q is required.", f.Name)}
		}
	}
	for name, v := range supplied {
		if !known[name] {
			return &UserError{Message: fmt.Sprintf("The credential field %q is not used by this cloud.", name)}
		}
		if len(v) > MaxCredentialValue {
			return &UserError{Message: fmt.Sprintf("The credential field %q is too long.", name)}
		}
	}
	return nil
}
