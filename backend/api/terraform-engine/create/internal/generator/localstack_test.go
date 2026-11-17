package generator

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// localstackServices are the AWS services A's templates and the setup below
// call. Each gets LocalStack's endpoint, so no request can reach AWS.
var localstackServices = []string{"apigateway", "dynamodb", "ec2", "iam", "lambda", "logs", "s3", "ssm", "sts"}

// localstackOverride points the AWS provider at LocalStack. It is a separate
// override file beside the writer's files, so A's writer stays unchanged. It
// keeps the account id lookup (LocalStack's STS answers it), so ARNs built
// from the account, such as an API's execution_arn, are whole.
func localstackOverride(endpoint string) string {
	var b strings.Builder
	b.WriteString("provider \"aws\" {\n")
	b.WriteString("  skip_credentials_validation = true\n")
	b.WriteString("  skip_metadata_api_check     = true\n")
	b.WriteString("  s3_use_path_style           = true\n\n")
	b.WriteString("  endpoints {\n")
	for _, s := range localstackServices {
		fmt.Fprintf(&b, "    %s = %q\n", s, endpoint)
	}
	b.WriteString("  }\n}\n")
	return b.String()
}

// localstackRun runs terraform in one folder with an environment built from
// nothing: LocalStack's test keys, never this machine's AWS credentials.
type localstackRun struct {
	t        *testing.T
	bin, dir string
	env      []string
}

func newLocalstackRun(t *testing.T, bin, dir, endpoint, pluginCache string, vars map[string]string) *localstackRun {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "localstack_override.tf"), []byte(localstackOverride(endpoint)), 0o600); err != nil {
		t.Fatal(err)
	}
	env := []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + dir,
		"TF_IN_AUTOMATION=1",
		"TF_INPUT=0",
		"CHECKPOINT_DISABLE=1",
		"TF_PLUGIN_CACHE_DIR=" + pluginCache,
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
		"AWS_REGION=" + localstackRegion,
		"AWS_ENDPOINT_URL=" + endpoint,
		"AWS_EC2_METADATA_DISABLED=true",
	}
	for k, v := range vars {
		env = append(env, "TF_VAR_"+k+"="+v)
	}
	return &localstackRun{t: t, bin: bin, dir: dir, env: env}
}

func (r *localstackRun) terraform(args ...string) (string, error) {
	cmd := exec.Command(r.bin, append([]string{"-chdir=" + r.dir}, args...)...)
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *localstackRun) must(args ...string) string {
	r.t.Helper()
	out, err := r.terraform(args...)
	if err != nil {
		r.t.Fatalf("terraform %s: %v\n%s", args[0], err, out)
	}
	return out
}

// managed lists the AWS resources in the state (not data sources, not random_id).
func (r *localstackRun) managed() []string {
	r.t.Helper()
	var out []string
	for _, a := range strings.Fields(r.must("state", "list")) {
		if strings.HasPrefix(a, "aws_") {
			out = append(out, a)
		}
	}
	sort.Strings(out)
	return out
}

// drift runs a refresh-only plan and returns each resource the provider read
// as changed outside Terraform, with the action ("delete" when it is gone).
func (r *localstackRun) drift() map[string]string {
	r.t.Helper()
	out := r.must("plan", "-refresh-only", "-json", "-input=false", "-lock=false")
	got := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for sc.Scan() {
		var line struct {
			Type   string `json:"type"`
			Change struct {
				Resource struct {
					Addr string `json:"addr"`
				} `json:"resource"`
				Action string `json:"action"`
			} `json:"change"`
		}
		if json.Unmarshal(sc.Bytes(), &line) == nil && line.Type == "resource_drift" {
			got[line.Change.Resource.Addr] = line.Change.Action
		}
	}
	return got
}

const localstackRegion = "ca-central-1"

// TestRenderedTerraformAppliesOnLocalStack applies A's rendered Terraform for
// real, against LocalStack (backend/devserver/localstack.sh), and destroys it.
// For each fixture it checks that every AWS resource exists after apply (a
// refresh-only plan reads none of them as gone) and that every one is gone
// after destroy (a refresh-only plan of the state from before the destroy
// reads every one as deleted). It runs only with CLUTTER_LOCALSTACK=1, and
// only against a loopback endpoint (CLUTTER_LOCALSTACK_ENDPOINT, default
// http://127.0.0.1:4566). LocalStack enforces no IAM, so this proves wiring,
// ordering and that each call is accepted, not that the customer role allows
// it.
func TestRenderedTerraformAppliesOnLocalStack(t *testing.T) {
	if os.Getenv("CLUTTER_LOCALSTACK") == "" {
		t.Skip("set CLUTTER_LOCALSTACK=1 and start LocalStack (npm run localstack -- start) to apply rendered Terraform against it")
	}
	endpoint := os.Getenv("CLUTTER_LOCALSTACK_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:4566"
	}
	if !regexp.MustCompile(`^http://(127\.0\.0\.1|localhost):[0-9]{1,5}$`).MatchString(endpoint) {
		t.Fatalf("CLUTTER_LOCALSTACK_ENDPOINT must be http://127.0.0.1:<port>, not %q", endpoint)
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(endpoint + "/_localstack/health")
	if err != nil {
		t.Fatalf("LocalStack does not answer at %s (npm run localstack -- start): %v", endpoint, err)
	}
	resp.Body.Close()
	tfBin, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("terraform is not on the PATH")
	}
	pluginCache := os.Getenv("TF_PLUGIN_CACHE_DIR")
	if pluginCache == "" {
		pluginCache = t.TempDir()
	}

	// The boundary the customer's stack would create, under the name the
	// deployer derives from the role's ARN.
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	unique := hex.EncodeToString(suffix)
	setupDir := t.TempDir()
	setup := fmt.Sprintf(`terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = %q
}

resource "aws_iam_policy" "boundary" {
  name   = "ClutterPermissionsBoundary-%s"
  path   = "/clutter/"
  policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [{ Effect = "Allow", Action = "*", Resource = "*" }]
  })
}

output "arn" {
  value = aws_iam_policy.boundary.arn
}
`, localstackRegion, unique)
	if err := os.WriteFile(filepath.Join(setupDir, "main.tf"), []byte(setup), 0o600); err != nil {
		t.Fatal(err)
	}
	boundary := newLocalstackRun(t, tfBin, setupDir, endpoint, pluginCache, nil)
	boundary.must("init", "-input=false", "-no-color")
	boundary.must("apply", "-auto-approve", "-input=false", "-no-color")
	t.Cleanup(func() {
		if out, err := boundary.terraform("destroy", "-auto-approve", "-input=false", "-no-color"); err != nil {
			t.Errorf("destroying the boundary: %v\n%s", err, out)
		}
	})
	boundaryARN := strings.Trim(strings.TrimSpace(boundary.must("output", "-raw", "arn")), `"`)
	t.Logf("boundary %s", boundaryARN)

	fixtures := []struct {
		name   string
		layout generic.DiagramLayout
	}{
		{"all-five-cors-on", allFiveDiagram(true)},
		{"all-five-cors-off", allFiveDiagram(false)},
		// The golden "defaults" fixture. The golden "hostile-free-text" one is
		// an escaping check, not a diagram AWS could build (its key pair does
		// not exist), so it is not applied.
		{"defaults", generic.DiagramLayout{Nodes: []generic.DiagramNode{
			node("a", "API Gateway", map[string]any{"resource_name": "api"}),
			node("b", "Lambda", map[string]any{"resource_name": "fn"}),
			node("c", "DynamoDB", map[string]any{"resource_name": "table"}),
			node("d", "S3", map[string]any{"resource_name": "bucket"}),
			node("e", "EC2", map[string]any{"resource_name": "vm"}),
		}}},
	}
	codeFile := regexp.MustCompile(`filename\s+=\s+"([^"]+\.zip)"`)
	keyName := regexp.MustCompile(`(?m)^\s*key_name = "[^"]*"\n`)
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			tf := mustGenerate(t, f.layout)
			// A key pair is the user's own and does not exist in LocalStack;
			// the instance is applied without it.
			tf.ResourcesTF = keyName.ReplaceAllString(tf.ResourcesTF, "")
			dir := t.TempDir()
			writeLikeTheWriter(t, dir, tf)
			// The runner downloads each function's code beside the files.
			for _, m := range codeFile.FindAllStringSubmatch(tf.ResourcesTF, -1) {
				writeCodeZip(t, filepath.Join(dir, m[1]))
			}
			run := newLocalstackRun(t, tfBin, dir, endpoint, pluginCache, map[string]string{
				"aws_region":               localstackRegion,
				"permissions_boundary_arn": boundaryARN,
			})
			run.must("init", "-input=false", "-no-color")
			destroyed := false
			t.Cleanup(func() {
				if destroyed {
					return
				}
				if out, err := run.terraform("destroy", "-auto-approve", "-input=false", "-no-color"); err != nil {
					t.Errorf("cleanup destroy: %v\n%s", err, out)
				}
			})
			run.must("apply", "-auto-approve", "-input=false", "-no-color")

			resources := run.managed()
			if len(resources) == 0 {
				t.Fatal("the apply created no AWS resource")
			}
			t.Logf("applied %d resources: %s", len(resources), strings.Join(resources, " "))
			for addr, action := range run.drift() {
				if action == "delete" {
					t.Errorf("%s does not exist after apply", addr)
				}
			}
			// A plan right after apply should change nothing. Where LocalStack
			// stores less than AWS, it shows here; that is logged, not failed.
			if out, err := run.terraform("plan", "-detailed-exitcode", "-input=false", "-no-color", "-lock=false"); err != nil {
				t.Logf("a plan after apply is not empty (LocalStack does not store every attribute):\n%s", planSummary(out))
			}

			state, err := os.ReadFile(filepath.Join(dir, "terraform.tfstate"))
			if err != nil {
				t.Fatal(err)
			}
			run.must("destroy", "-auto-approve", "-input=false", "-no-color")
			destroyed = true
			if left := run.managed(); len(left) > 0 {
				t.Fatalf("the state still holds %v after destroy", left)
			}
			// Read the state from before the destroy again: every resource
			// in it must now be gone from LocalStack.
			if err := os.WriteFile(filepath.Join(dir, "terraform.tfstate"), state, 0o600); err != nil {
				t.Fatal(err)
			}
			drift := run.drift()
			var still []string
			for _, addr := range resources {
				if drift[addr] != "delete" {
					still = append(still, addr)
				}
			}
			// The state put back holds resources that no longer exist; empty it
			// so the cleanup has nothing to destroy.
			if err := os.Remove(filepath.Join(dir, "terraform.tfstate")); err != nil {
				t.Fatal(err)
			}
			if len(still) > 0 {
				t.Errorf("still in LocalStack after destroy: %s", strings.Join(still, " "))
			}
		})
	}
}

// writeCodeZip writes a one-file Node.js function, as the user's upload would be.
func writeCodeZip(t *testing.T, path string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("index.js")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("exports.handler = async () => ({ statusCode: 200, body: \"ok\" });\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// planSummary keeps the lines of a plan that name a change.
func planSummary(out string) string {
	var keep []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, " will be ") || strings.Contains(l, " must be ") || strings.Contains(l, "forces replacement") ||
			strings.HasPrefix(strings.TrimSpace(l), "Plan:") || strings.HasPrefix(strings.TrimSpace(l), "~ ") ||
			strings.HasPrefix(strings.TrimSpace(l), "Error") {
			keep = append(keep, l)
		}
	}
	return strings.Join(keep, "\n")
}
