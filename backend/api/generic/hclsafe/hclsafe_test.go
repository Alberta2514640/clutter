package hclsafe

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{"uploads", true},
		{"my-api", true},
		{"my_users_table", true},
		{"a", true},
		{"a1-b2_c3", true},
		{strings.Repeat("a", MaxNameLength), true},
		{"", false},
		{strings.Repeat("a", MaxNameLength+1), false},
		{"1bucket", false},
		{"-api", false},
		{"_api", false},
		{"MyApi", false},
		{"my api", false},
		{"my.api", false},
		{`my"api`, false},
		{"my${api}", false},
		{"my%{api}", false},
		{"my\napi", false},
		{"my\\api", false},
		{"caf\u00e9", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.name), func(t *testing.T) {
			err := CheckName(tt.name)
			if tt.ok && err != nil {
				t.Fatalf("refused: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

// escapeCases are the values the escaper must carry through HCL unchanged.
var escapeCases = []struct {
	in, want string
}{
	{"", ""},
	{"plain text", "plain text"},
	{`say "hi"`, `say \"hi\"`},
	{`C:\path`, `C:\\path`},
	{"line one\nline two", `line one\nline two`},
	{"cr\rtab\t", `cr\rtab\t`},
	{"bell\x07del\x7f", `bell\u0007del\u007F`},
	{"sep\u2028para\u2029", `sep\u2028para\u2029`},
	{"${var.secret}", "$${var.secret}"},
	{"%{ if true }x%{ endif }", "%%{ if true }x%%{ endif }"},
	{"$${already}", "$$${already}"},
	{"%%{already}", "%%%{already}"},
	{"cost: $5 and 50% off", "cost: $5 and 50% off"},
	{"trailing $", "trailing $"},
	{"trailing %", "trailing %"},
	{"{braces} alone", "{braces} alone"},
	{`"}${file("/etc/passwd")}"`, `\"}$${file(\"/etc/passwd\")}\"`},
	{"caf\u00e9 \u65e5\u672c", "caf\u00e9 \u65e5\u672c"},
	{"bad\xffbyte", "bad\uFFFDbyte"},
}

func TestEscape(t *testing.T) {
	for _, tt := range escapeCases {
		t.Run(fmt.Sprintf("%q", tt.in), func(t *testing.T) {
			got := Escape(tt.in)
			if got != tt.want {
				t.Fatalf("Escape(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if strings.ContainsAny(got, "\n\r") {
				t.Fatalf("escaped value holds a line break: %q", got)
			}
		})
	}
}

func TestAllowList(t *testing.T) {
	list := NewAllowList("b", "a", "c")
	for _, v := range []string{"a", "b", "c"} {
		if !list.Contains(v) {
			t.Fatalf("%q refused", v)
		}
	}
	for _, v := range []string{"", "A", " a", "a ", "d", "a,b"} {
		if list.Contains(v) {
			t.Fatalf("%q accepted", v)
		}
	}
	if got := list.String(); got != "b, a, c" {
		t.Fatalf("String() = %q", got)
	}
	values := list.Values()
	values[0] = "changed"
	if !list.Contains("b") || list.Contains("changed") {
		t.Fatal("Values() exposed the list's own slice")
	}
}

// TestEscapeRoundTripsThroughTerraform asks Terraform itself what each escaped
// value means. It runs only when CLUTTER_TF_VALIDATE is set and terraform is on
// the PATH; it plans a configuration with outputs only, so it needs no provider,
// no network and no cloud account.
func TestEscapeRoundTripsThroughTerraform(t *testing.T) {
	if os.Getenv("CLUTTER_TF_VALIDATE") == "" {
		t.Skip("set CLUTTER_TF_VALIDATE=1 to check escaping with terraform")
	}
	tf, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform is not on the PATH")
	}

	dir := t.TempDir()
	var config strings.Builder
	for i, tt := range escapeCases {
		// Invalid UTF-8 cannot come back unchanged; TestEscape covers it.
		if !strings.Contains(tt.in, "\xff") {
			fmt.Fprintf(&config, "output \"v%d\" {\n  value = \"%s\"\n}\n", i, Escape(tt.in))
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(config.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) []byte {
		cmd := exec.Command(tf, append([]string{"-chdir=" + dir}, args...)...)
		cmd.Env = append(os.Environ(), "TF_IN_AUTOMATION=1")
		out, err := cmd.Output()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				t.Fatalf("terraform %v: %v\n%s%s", args, err, out, exitErr.Stderr)
			}
			t.Fatalf("terraform %v: %v", args, err)
		}
		return out
	}
	run("init", "-backend=false", "-input=false", "-no-color")
	run("plan", "-refresh=false", "-input=false", "-no-color", "-out=plan.bin")
	shown := run("show", "-json", "plan.bin")

	var plan struct {
		PlannedValues struct {
			Outputs map[string]struct {
				Value string `json:"value"`
			} `json:"outputs"`
		} `json:"planned_values"`
	}
	if err := json.Unmarshal(shown, &plan); err != nil {
		t.Fatal(err)
	}
	for i, tt := range escapeCases {
		if strings.Contains(tt.in, "\xff") {
			continue
		}
		got, ok := plan.PlannedValues.Outputs[fmt.Sprintf("v%d", i)]
		if !ok {
			t.Fatalf("no output for %q", tt.in)
		}
		if got.Value != tt.in {
			t.Errorf("terraform read Escape(%q) as %q", tt.in, got.Value)
		}
	}
}
