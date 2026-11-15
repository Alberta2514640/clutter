package connectors

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Portable field names a connector reads beyond the plain values.
const (
	// FieldStartupScript is a VM's startup script (StartupUserData).
	FieldStartupScript = "startup_script"
	// FieldHandler is a function's entry point inside its code.
	FieldHandler = "handler"
)

// standardFields owns the portable fields of each building block. A diagram
// holds only these, so one diagram can pass every connector's manifest; what is
// specific to one cloud (an image, an instance type) is a deployment setting.
// A connector that offers a standard block as-is declares exactly these fields
// (StandardBlock); the contract tests check it.
var standardFields = map[BlockKind][]FieldSpec{
	KindVM: {
		{Name: "size", Title: "Size", Type: FieldEnum, Required: true, Default: "small", Allowed: []string{"small", "medium", "large"}},
		{Name: "public", Title: "Reachable from the internet", Type: FieldBool, Default: false},
		// Every cloud runs it through cloud-init at first boot (StartupUserData).
		{Name: FieldStartupScript, Title: "Startup script", Type: FieldText},
	},
	KindNetwork: {
		{Name: "cidr", Title: "Address range (CIDR)", Type: FieldString, Required: true, Default: "10.0.0.0/24"},
	},
	KindFirewall: {
		{Name: "ingress", Title: "Allowed incoming traffic (for example tcp/22,tcp/443,udp/1000-2000)", Type: FieldString, Default: "tcp/22"},
		{Name: "source_cidr", Title: "Allowed source addresses (CIDR)", Type: FieldString, Required: true, Default: "0.0.0.0/0"},
	},
	KindVolume: {
		{Name: "size_gb", Title: "Size in GB", Type: FieldInt, Required: true, Default: 10},
	},
	KindBucket: {
		{Name: "versioning", Title: "Keep old versions", Type: FieldBool, Default: false},
	},
	KindFunction: {
		{Name: "memory_mb", Title: "Memory in MB", Type: FieldInt, Default: 128},
		{Name: "timeout_s", Title: "Time limit in seconds", Type: FieldInt, Default: 3},
		// The runtime is a deployment setting; an empty handler means the
		// runtime's own default.
		{Name: FieldHandler, Title: "Handler (empty uses the runtime's default)", Type: FieldString},
	},
	KindTable: {
		{Name: "key", Title: "Key name", Type: FieldString, Default: "id"},
		{Name: "key_type", Title: "Key type", Type: FieldEnum, Default: "string", Allowed: []string{"string", "number", "binary"}},
	},
	KindHTTPAPI: {
		{Name: "description", Title: "Description", Type: FieldString},
		{Name: "cors", Title: "Allow calls from web pages on other sites", Type: FieldBool, Default: true},
	},
}

// StandardFields returns a copy of a building block's portable fields, or nil
// for a kind that is not standard.
func StandardFields(kind BlockKind) []FieldSpec {
	fields := standardFields[kind]
	if fields == nil {
		return nil
	}
	out := make([]FieldSpec, len(fields))
	for i, f := range fields {
		f.Allowed = append([]string(nil), f.Allowed...)
		out[i] = f
	}
	return out
}

// StandardField returns one portable field of a building block. It panics on a
// name the block does not have, so a typo fails at start-up and in every test.
func StandardField(kind BlockKind, name string) FieldSpec {
	for _, f := range StandardFields(kind) {
		if f.Name == name {
			return f
		}
	}
	panic("connectors: no standard field " + string(kind) + "." + name)
}

// StandardBlock is a block a connector offers as-is, with the portable fields.
func StandardBlock(kind BlockKind, title string) BlockSpec {
	return BlockSpec{Kind: kind, Treatment: TreatmentAsIs, Title: title, Fields: StandardFields(kind)}
}

// FunctionCode is a zip a user uploaded for one function block through B
// (internal/codestore). A connector that builds functions renders it as the
// function's code; with none, it renders placeholder code.
type FunctionCode struct {
	// Key is where B's code store keeps the zip. A connector passes it to the
	// Terraform runner, which stages the zip; it never opens it itself.
	Key string
	// SHA256 is the zip's SHA-256 digest in lowercase hex.
	SHA256 string
	// Size is the zip's size in bytes.
	Size int64
}

// SHA256Pattern is a SHA-256 digest in lowercase hex, as B's code store
// records it.
var SHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Base64SHA256 returns the digest in standard base64, the form AWS Lambda's
// source_code_hash takes.
func (c FunctionCode) Base64SHA256() (string, error) {
	if !SHA256Pattern.MatchString(c.SHA256) {
		return "", errors.New("the code's SHA-256 is not 64 lowercase hex digits")
	}
	sum, err := hex.DecodeString(c.SHA256)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sum), nil
}

// startupShell runs a startup script that does not name its own interpreter.
const startupShell = "#!/bin/sh\n"

// StartupUserData turns a VM block's startup script into cloud-init user data,
// encoded in base64 so it reaches Terraform as plain base64 text and never as
// raw text inside an HCL string or heredoc. It returns "" when the block has
// no script. Cloud-init runs the script once, as root, at first boot. Windows
// line ends become Unix ones, and a script whose first line does not start
// with "#!" runs with /bin/sh. The error's message is safe to show the user.
func StartupUserData(fields map[string]any) (string, error) {
	v, set := fields[FieldStartupScript]
	if !set || v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", errors.New("The startup script must be text.")
	}
	if strings.TrimSpace(s) == "" {
		return "", nil
	}
	if !utf8.ValidString(s) {
		return "", errors.New("The startup script must be plain UTF-8 text.")
	}
	for _, r := range s {
		if (r < 0x20 && r != '\t' && r != '\n' && r != '\r') || r == 0x7f {
			return "", errors.New("The startup script may hold no control characters other than tabs and line breaks.")
		}
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.HasPrefix(s, "#!") {
		s = startupShell + s
	}
	if len(s) > MaxTextField {
		return "", fmt.Errorf("The startup script must be at most %d bytes, counting the #!/bin/sh line Clutter adds when the script has none.", MaxTextField)
	}
	return base64.StdEncoding.EncodeToString([]byte(s)), nil
}
