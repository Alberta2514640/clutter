package generalopenstack

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
)

// hostileScript would break out of an HCL string or heredoc if it reached
// the file as text.
const hostileScript = "echo \"}\n${file(\"/etc/passwd\")} %{ if true }\nEOT\n<<EOF\nprovisioner \"local-exec\" {}\n"

var userDataLine = regexp.MustCompile(`user_data\s+= base64decode\("([A-Za-z0-9+/=]+)"\)`)

func TestStartupScriptReachesTheServerAsBase64(t *testing.T) {
	g := sampleGraph()
	g.Blocks["n1"].Fields[connectors.FieldStartupScript] = hostileScript
	c := newConnector(nil)
	d := testDeployment(fixedDeployment)
	if ps := c.Validate(g, d); len(ps) > 0 {
		t.Fatalf("refused: %+v", ps)
	}
	files, err := c.Render(g, d)
	if err != nil {
		t.Fatal(err)
	}
	if err := terraform.CheckFiles(files, providers); err != nil {
		t.Fatalf("the runner refuses the rendered files: %v", err)
	}
	main := string(files["main.tf"])
	m := userDataLine.FindStringSubmatch(main)
	if m == nil {
		t.Fatalf("no base64 user_data on the server:\n%s", main)
	}
	got, err := base64.StdEncoding.DecodeString(m[1])
	if err != nil {
		t.Fatal(err)
	}
	if want := "#!/bin/sh\n" + hostileScript; string(got) != want {
		t.Fatalf("user data decodes to %q, want %q", got, want)
	}
	for _, raw := range []string{"/etc/passwd", "local-exec", "EOT"} {
		if strings.Contains(main, raw) {
			t.Errorf("main.tf holds the script's text %q", raw)
		}
	}
}

func TestNoStartupScriptNoUserData(t *testing.T) {
	for name, v := range map[string]any{"unset": nil, "empty": "", "blank": " \n\t"} {
		t.Run(name, func(t *testing.T) {
			g := sampleGraph()
			if v != nil {
				g.Blocks["n1"].Fields[connectors.FieldStartupScript] = v
			}
			files, err := newConnector(nil).Render(g, testDeployment(fixedDeployment))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(files["main.tf"]), "user_data") {
				t.Fatal("a VM with no script has user_data")
			}
		})
	}
}

func TestStartupScriptKeepsItsOwnInterpreter(t *testing.T) {
	g := sampleGraph()
	g.Blocks["n1"].Fields[connectors.FieldStartupScript] = "#!/bin/bash\r\nset -e\r\napt-get install -y nginx\r\n"
	files, err := newConnector(nil).Render(g, testDeployment(fixedDeployment))
	if err != nil {
		t.Fatal(err)
	}
	m := userDataLine.FindStringSubmatch(string(files["main.tf"]))
	if m == nil {
		t.Fatal("no user_data")
	}
	got, _ := base64.StdEncoding.DecodeString(m[1])
	if want := "#!/bin/bash\nset -e\napt-get install -y nginx\n"; string(got) != want {
		t.Fatalf("user data = %q, want %q", got, want)
	}
}

func TestStartupScriptLimits(t *testing.T) {
	cases := map[string]any{
		"over 16 KB":           strings.Repeat("x", connectors.MaxTextField+1),
		"16 KB plus the shell": strings.Repeat("x", connectors.MaxTextField-5),
		"a NUL byte":           "echo a\x00b",
		"an escape character":  "echo \x1b[31m",
		"not text":             float64(3),
	}
	c := newConnector(nil)
	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			g := sampleGraph()
			g.Blocks["n1"].Fields[connectors.FieldStartupScript] = v
			ps := c.Validate(g, testDeployment(fixedDeployment))
			found := false
			for _, p := range ps {
				found = found || (p.NodeID == "n1" && p.Field == connectors.FieldStartupScript)
			}
			if !found {
				t.Fatalf("not refused on the field: %+v", ps)
			}
			if _, err := c.Render(g, testDeployment(fixedDeployment)); err == nil {
				t.Fatal("rendered it")
			}
		})
	}
}
