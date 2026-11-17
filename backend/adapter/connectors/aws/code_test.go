package aws

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/codestore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/google/uuid"
)

// hostileScript would break out of an HCL string or heredoc if it reached
// the file as text.
const hostileScript = "echo \"}\n${file(\"/etc/passwd\")} %{ if true }\nEOT\n<<EOF\n"

func testZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("app.py")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("def main(event, context):\n    return {'statusCode': 200}\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// uploaded stores body in a code store and returns the function code record
// B's core would put on the block.
func uploaded(t *testing.T, store codestore.Objects, body []byte) *connectors.FunctionCode {
	t.Helper()
	sum := sha256.Sum256(body)
	sha := hex.EncodeToString(sum[:])
	key := codestore.Key(uuid.NewString(), uuid.NewString(), "fn", sha)
	if store != nil {
		if err := store.Put(context.Background(), key, bytes.NewReader(body), int64(len(body))); err != nil {
			t.Fatal(err)
		}
	}
	return &connectors.FunctionCode{Key: key, SHA256: sha, Size: int64(len(body))}
}

// withCodeAndScript is fullGraph with uploaded code and a handler on the
// function, and a startup script on the web VM.
func withCodeAndScript(code *connectors.FunctionCode) connectors.Graph {
	g := fullGraph()
	fn := g.Blocks["fn"]
	fn.Code = code
	fn.Fields[connectors.FieldHandler] = "app.main"
	g.Blocks["fn"] = fn
	g.Blocks["web"].Fields[connectors.FieldStartupScript] = hostileScript
	return g
}

func TestUploadedCodeReplacesThePlaceholder(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	code := uploaded(t, nil, testZip(t))
	files := renderFor(t, c, withCodeAndScript(code), d)
	addr := d.Prefix + "_fn"

	if _, ok := files["code.tf"]; ok {
		t.Fatalf("code.tf is rendered although the only function has uploaded code:\n%s", files["code.tf"])
	}
	sum, _ := hex.DecodeString(code.SHA256)
	override := string(files["override.tf"])
	for _, want := range []string{
		`filename         = "` + terraform.ArchiveName(addr) + `"`,
		`source_code_hash = "` + base64.StdEncoding.EncodeToString(sum) + `"`,
	} {
		if !strings.Contains(override, want) {
			t.Errorf("override.tf lacks %q:\n%s", want, override)
		}
	}
	if strings.Contains(override, "data.archive_file."+addr) {
		t.Error("the function still points at the placeholder zip")
	}
	if main := string(files["main.tf"]); !strings.Contains(main, `handler       = "app.main"`) {
		t.Errorf("main.tf does not use the block's handler")
	}

	m := newModel(withCodeAndScript(code), d)
	if got := m.archives(); len(got) != 1 || got[0] != (terraform.Archive{ID: addr, Key: code.Key, SHA256: code.SHA256}) {
		t.Fatalf("archives = %+v", got)
	}
}

func TestNoUploadedCodeKeepsThePlaceholder(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	files := renderFor(t, c, fullGraph(), d)
	addr := d.Prefix + "_fn"
	if !strings.Contains(string(files["code.tf"]), `data "archive_file" "`+addr+`_code"`) {
		t.Fatalf("no placeholder zip:\n%s", files["code.tf"])
	}
	if !strings.Contains(string(files["override.tf"]), "source_code_hash = data.archive_file."+addr+"_code.output_base64sha256") {
		t.Fatal("the function does not use the placeholder's hash")
	}
	// python3.12's default handler, from generic.LambdaRuntimes.
	if !strings.Contains(string(files["main.tf"]), `handler       = "lambda_function.lambda_handler"`) {
		t.Fatal("an empty handler does not fall back to the runtime's default")
	}
	if m := newModel(fullGraph(), d); len(m.archives()) != 0 {
		t.Fatal("a run without uploaded code stages archives")
	}
	if strings.Contains(string(files["override.tf"]), "user_data") {
		t.Fatal("a VM with no script has user data")
	}
}

var userDataLine = regexp.MustCompile(`user_data_base64\s+= "([A-Za-z0-9+/=]+)"`)

func TestStartupScriptBecomesBase64UserData(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	files := renderFor(t, c, withCodeAndScript(nil), d)
	override := string(files["override.tf"])
	m := userDataLine.FindStringSubmatch(override)
	if m == nil {
		t.Fatalf("no user_data_base64:\n%s", override)
	}
	got, _ := base64.StdEncoding.DecodeString(m[1])
	if want := "#!/bin/sh\n" + hostileScript; string(got) != want {
		t.Fatalf("user data = %q, want %q", got, want)
	}
	if !strings.Contains(override, "user_data_replace_on_change = true") {
		t.Error("a changed script would not build a new instance")
	}
	if n := len(userDataLine.FindAllString(override, -1)); n != 1 {
		t.Errorf("%d VMs have user data, want only the one with a script", n)
	}
	for _, raw := range []string{"/etc/passwd", "EOT"} {
		if strings.Contains(joined(files), raw) {
			t.Errorf("the files hold the script's text %q", raw)
		}
	}
}

func TestHandlerAndScriptAreChecked(t *testing.T) {
	c, _, _ := newTestConnector(t)
	cases := map[string]struct {
		node, field string
		value       any
	}{
		"handler with a quote":         {"fn", connectors.FieldHandler, `app" }`},
		"handler with interpolation":   {"fn", connectors.FieldHandler, "${file(\"x\")}"},
		"handler with a space":         {"fn", connectors.FieldHandler, "app main"},
		"handler too long":             {"fn", connectors.FieldHandler, strings.Repeat("a", 129)},
		"script over 16 KB":            {"web", connectors.FieldStartupScript, strings.Repeat("x", connectors.MaxTextField+1)},
		"script with a NUL":            {"web", connectors.FieldStartupScript, "a\x00b"},
		"script that is not text":      {"web", connectors.FieldStartupScript, true},
		"script plus shell over 16 KB": {"web", connectors.FieldStartupScript, strings.Repeat("x", connectors.MaxTextField-3)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			g := fullGraph()
			g.Blocks[tc.node].Fields[tc.field] = tc.value
			found := false
			for _, p := range c.Validate(g, testDeployment("ca-central-1")) {
				found = found || (p.NodeID == tc.node && p.Field == tc.field)
			}
			if !found {
				t.Fatal("not refused on the field")
			}
		})
	}
	g := fullGraph()
	fn := g.Blocks["fn"]
	fn.Code = &connectors.FunctionCode{Key: "code/x.zip", SHA256: "not-hex"}
	g.Blocks["fn"] = fn
	if !contract.HasProblem(c.Validate(g, testDeployment("ca-central-1")), "fn", connectors.CodeInvalidValue) {
		t.Fatal("a code record with a bad digest is accepted")
	}
}

// stagingLauncher records which files each Terraform step saw.
type stagingLauncher struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (s *stagingLauncher) Launch(_ context.Context, task runtask.Task) error {
	entries, err := os.ReadDir(task.Dir)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = map[string][]byte{}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(task.Dir, e.Name()))
		if err != nil {
			return err
		}
		s.files[e.Name()] = b
	}
	return nil
}

func TestARunStagesTheUploadedZip(t *testing.T) {
	root := t.TempDir()
	store, err := codestore.NewDisk(filepath.Join(root, "code"))
	if err != nil {
		t.Fatal(err)
	}
	l := &stagingLauncher{}
	tf := &terraform.Runner{
		Bin: "/usr/bin/false", WorkDir: filepath.Join(root, "work"), StateDir: filepath.Join(root, "state"),
		PluginCacheDir: filepath.Join(root, "plugins"), Launcher: l, Code: store,
	}
	c := New(tf, diskTemplates(t), &fakeSTS{})
	body := testZip(t)
	g := withCodeAndScript(uploaded(t, store, body))
	d := testDeployment("ca-central-1")
	// The fake launcher prints no plan, so Plan reports Terraform's error
	// after every step ran; the folder is what this test checks.
	_, _ = c.Plan(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionPlan, Log: &bytes.Buffer{}}, g, d)
	got, ok := l.files[terraform.ArchiveName(d.Prefix+"_fn")]
	if !ok || !bytes.Equal(got, body) {
		names := make([]string, 0, len(l.files))
		for n := range l.files {
			names = append(names, n)
		}
		t.Fatalf("the run folder holds %v, without the uploaded zip", names)
	}
}
