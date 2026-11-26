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
	// FieldSSHKey is a VM's OpenSSH public key (SSHPublicKey).
	FieldSSHKey = "ssh_public_key"
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
		{Name: FieldStartupScript, Title: "Startup script", Type: FieldText,
			Help: "Runs once as root when the VM first starts. Use it to install and start your app."},
		// Text, not a string: an RSA key is longer than MaxStringField.
		{Name: FieldSSHKey, Title: "SSH public key", Type: FieldText, Help: "Paste your public key to log in with SSH."},
	},
	KindNetwork: {
		{Name: "cidr", Title: "Address range (CIDR)", Type: FieldString, Required: true, Default: "10.0.0.0/24"},
		// What it does and costs differs by cloud, so each connector says so
		// in its network block's help.
		{Name: "private_internet", Title: "Internet access for private servers", Type: FieldBool, Default: true,
			Help: "Lets servers without a public address reach the internet."},
	},
	KindFirewall: {
		// No default rule: a new firewall lets nothing in until the user says
		// what. The source range applies only to the rules listed.
		{Name: "ingress", Title: "Allowed incoming traffic (for example tcp/22,tcp/443,udp/1000-2000)", Type: FieldString},
		{Name: "source_cidr", Title: "Allowed source addresses (CIDR)", Type: FieldString, Required: true, Default: "0.0.0.0/0"},
	},
	KindVolume: {
		{Name: "size_gb", Title: "Size in GB", Type: FieldInt, Required: true, Default: 10},
	},
	KindBucket: {
		{Name: "versioning", Title: "Keep old versions", Type: FieldBool, Default: false},
		{Name: "force_destroy", Title: "Delete files on destroy", Type: FieldBool, Default: false,
			Help: "Lets destroy remove the bucket even when it still holds files. The files are deleted for good."},
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

// standardTitles owns what each building block is called. Every connector
// names a standard block the same way (StandardBlock, RemovedBlock); the
// cloud's own product name is its BlockDisplay.
var standardTitles = map[BlockKind]string{
	KindVM:       "VM",
	KindNetwork:  "Network",
	KindFirewall: "Firewall",
	KindVolume:   "Volume",
	KindBucket:   "Bucket",
	KindFunction: "Function",
	KindTable:    "Table",
	KindHTTPAPI:  "HTTP API",
}

// StandardTitle returns a building block's title, or "" for a kind that is
// not standard.
func StandardTitle(kind BlockKind) string { return standardTitles[kind] }

// standardSummaries owns what each building block is for, in a few plain
// words for a person with no cloud background. Every connector that builds a
// standard block shows the same line.
var standardSummaries = map[BlockKind]string{
	KindVM:       "A server that runs your app",
	KindNetwork:  "A private area for servers",
	KindFirewall: "Who may reach a server",
	KindVolume:   "An extra disk for a server",
	KindBucket:   "Stores files",
	KindFunction: "Code that runs when called",
	KindTable:    "Stores records, such as users",
	KindHTTPAPI:  "A web address for your code",
}

// StandardSummary returns a building block's summary, or "" for a kind that
// is not standard.
func StandardSummary(kind BlockKind) string { return standardSummaries[kind] }

// StandardBlock is a block a connector offers as-is, with the standard title,
// portable fields and summary.
func StandardBlock(kind BlockKind) BlockSpec {
	return BlockSpec{Kind: kind, Treatment: TreatmentAsIs, Title: StandardTitle(kind), Summary: StandardSummary(kind), Fields: StandardFields(kind)}
}

// RemovedBlock is a standard block a connector cannot build, under its
// standard title.
func RemovedBlock(kind BlockKind) BlockSpec {
	return BlockSpec{Kind: kind, Treatment: TreatmentRemoved, Title: StandardTitle(kind)}
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

// The SSH key types every cloud Clutter builds on imports: EC2 takes only
// these two.
const (
	sshEd25519 = "ssh-ed25519"
	sshRSA     = "ssh-rsa"
	minRSABits = 2048
	maxRSABits = 16384
)

var errSSHKey = errors.New("The SSH public key must be one line that starts with ssh-ed25519 or ssh-rsa, such as the contents of ~/.ssh/id_ed25519.pub.")

// SSHPublicKey reads a VM block's SSH public key: one OpenSSH line, an
// ssh-ed25519 key or an ssh-rsa key of 2048 bits or more, whose base64 body
// holds a key of its named type. It returns the key as "<type> <base64>",
// without the comment, so only checked text reaches Terraform; "" when the
// block has none. The key is public, but a connector still never logs it.
// The error's message is safe to show the user.
func SSHPublicKey(fields map[string]any) (string, error) {
	v, set := fields[FieldSSHKey]
	if !set || v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", errSSHKey
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if strings.ContainsAny(s, "\r\n") {
		return "", errSSHKey
	}
	parts := strings.Fields(s)
	if len(parts) < 2 || (parts[0] != sshEd25519 && parts[0] != sshRSA) {
		return "", errSSHKey
	}
	blob, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errSSHKey
	}
	typ, rest, ok := sshString(blob)
	if !ok || string(typ) != parts[0] {
		return "", errSSHKey
	}
	switch parts[0] {
	case sshEd25519:
		key, rest, ok := sshString(rest)
		if !ok || len(key) != 32 || len(rest) != 0 {
			return "", errSSHKey
		}
	case sshRSA:
		e, rest, ok := sshString(rest)
		if !ok || len(e) == 0 {
			return "", errSSHKey
		}
		n, rest, ok := sshString(rest)
		if !ok || len(n) == 0 || len(rest) != 0 {
			return "", errSSHKey
		}
		// n is a positive mpint: a leading zero byte only keeps its sign.
		n = []byte(strings.TrimLeft(string(n), "\x00"))
		bits := len(n) * 8
		if len(n) > 0 {
			for b := n[0]; b&0x80 == 0; b <<= 1 {
				bits--
			}
		}
		if bits < minRSABits || bits > maxRSABits {
			return "", fmt.Errorf("An RSA key must have from %d to %d bits. An ssh-ed25519 key is shorter and safer.", minRSABits, maxRSABits)
		}
	}
	return parts[0] + " " + parts[1], nil
}

// sshString reads one length-prefixed string of the SSH wire format.
func sshString(b []byte) (s, rest []byte, ok bool) {
	if len(b) < 4 {
		return nil, nil, false
	}
	n := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if n < 0 || n > len(b)-4 {
		return nil, nil, false
	}
	return b[4 : 4+n], b[4+n:], true
}
