package generator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator/iam"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator/writer"
)

// templateDir is the repo's copy of the templates the platform uploads to S3.
const templateDir = "../../../../../infra/modules/s3/uploads/aws"

// diskGenerator renders the repo's template files; no test here reaches S3.
func diskGenerator(t *testing.T, dir string) *TerraformGenerator {
	t.Helper()
	loader, err := tftemplate.NewTemplateLoaderFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &TerraformGenerator{iamGenerator: iam.NewIAMPolicyGenerator(), templateLoader: loader}
}

// hostile holds what a canvas value must never smuggle into generated code: a
// quote that ends the string, an interpolation, a directive, and a line break.
const hostile = "x\" }\nresource \"aws_iam_user\" \"evil\" { name = \"${data.aws_caller_identity.me.arn}%{ if true }y%{ endif }"

func allFiveDiagram(enableCORS bool) generic.DiagramLayout {
	return generic.DiagramLayout{
		Nodes: []generic.DiagramNode{
			node("api", "API Gateway", map[string]any{
				"resource_name": "orders-api",
				"description":   "Orders API",
				"stage_name":    "v1",
				"enable_cors":   enableCORS,
				"http_methods":  "GET, POST,DELETE",
			}),
			node("fn", "Lambda", map[string]any{
				"resource_name":         "orders-handler",
				"handler":               "index.handler",
				"runtime":               "nodejs22.x",
				"architecture":          "arm64",
				"timeout":               float64(30),
				"memory_size":           float64(256),
				"s3_key":                "org/project/diagram/terraform/code/orders-handler/bootstrap.zip",
				"environment_variables": map[string]any{"TABLE_NAME": "orders", "LOG_LEVEL": "info"},
			}),
			node("table", "DynamoDB", map[string]any{
				"resource_name": "orders",
				"billing_mode":  "PAY_PER_REQUEST",
				"hash_key":      "orderId",
				"hash_key_type": "S",
				"enable_ttl":    true,
				"ttl_attribute": "expires_at",
			}),
			node("bucket", "S3", map[string]any{
				"resource_name":     "order_files",
				"enable_versioning": true,
			}),
			node("vm", "EC2 Container", map[string]any{
				"resource_name": "worker",
				"instance_type": "t3.micro",
				"key_name":      "ops-key",
			}),
		},
		Edges: []generic.DiagramEdge{
			{ID: "e1", Source: "api", Target: "fn"},
			{ID: "e2", Source: "fn", Target: "table"},
			{ID: "e3", Source: "fn", Target: "bucket"},
		},
	}
}

func generate(t *testing.T, gen *TerraformGenerator, layout generic.DiagramLayout) (*internal.GeneratedTerraform, []internal.GenerationError) {
	t.Helper()
	return gen.Generate(context.Background(), "diagram", layout)
}

func mustGenerate(t *testing.T, layout generic.DiagramLayout) *internal.GeneratedTerraform {
	t.Helper()
	tf, errs := generate(t, diskGenerator(t, templateDir), layout)
	if len(errs) > 0 {
		t.Fatalf("errors: %+v", errs)
	}
	return tf
}

func TestAValidDiagramWithEveryResourceTypeRenders(t *testing.T) {
	tf := mustGenerate(t, allFiveDiagram(true))
	for _, want := range []string{
		`resource "aws_api_gateway_rest_api" "orders_api"`,
		`resource "aws_api_gateway_method" "orders_api_options"`,
		`"'GET,POST,DELETE,OPTIONS'"`,
		`resource "aws_lambda_function" "orders_handler"`,
		`runtime       = "nodejs22.x"`,
		`filename         = "code/orders-handler/bootstrap.zip"`,
		`TABLE_NAME = "orders"`,
		`resource "aws_dynamodb_table" "orders"`,
		`attribute_name = "expires_at"`,
		`resource "aws_s3_bucket" "order_files"`,
		`bucket = "order-files-${random_id.order_files_suffix.hex}"`,
		`resource "aws_instance" "worker"`,
		`key_name = "ops-key"`,
	} {
		if !strings.Contains(tf.ResourcesTF, want) {
			t.Errorf("resources lack %q", want)
		}
	}
	for _, want := range []string{"orders_api_invoke_url", "orders_handler_arn", "orders_arn", "order_files_bucket_arn", "worker_id"} {
		if !strings.Contains(tf.OutputsTF, `output "`+want+`"`) {
			t.Errorf("outputs lack %q", want)
		}
	}
}

func TestEveryResourceTypeRendersWithOnlyAName(t *testing.T) {
	for _, label := range []string{"API Gateway", "Lambda", "DynamoDB", "S3", "EC2"} {
		t.Run(label, func(t *testing.T) {
			tf := mustGenerate(t, generic.DiagramLayout{Nodes: []generic.DiagramNode{
				node("a", label, map[string]any{"resource_name": "thing"}),
			}})
			if tf.ResourcesTF == "" || tf.OutputsTF == "" {
				t.Fatalf("empty render: %+v", tf)
			}
		})
	}
}

func TestAnAPIGatewayWithCORSOffNamesNoOptionsResource(t *testing.T) {
	tf := mustGenerate(t, allFiveDiagram(false))
	if strings.Contains(tf.ResourcesTF, "_options") {
		t.Fatalf("CORS off still names an OPTIONS resource:\n%s", tf.ResourcesTF)
	}
	if !strings.Contains(tf.ResourcesTF, `resource "aws_api_gateway_deployment" "orders_api_deployment"`) {
		t.Fatal("no deployment rendered")
	}
}

func TestOutOfListValuesAndBadNamesAreRefused(t *testing.T) {
	tests := []struct {
		name, label, field string
		value              any
	}{
		{"name with a quote", "S3", "resource_name", `files"`},
		{"name with an interpolation", "Lambda", "resource_name", "${var.x}"},
		{"name with a directive", "DynamoDB", "resource_name", "%{if true}x"},
		{"name with a newline", "EC2", "resource_name", "web\nserver"},
		{"name starting with a digit", "API Gateway", "resource_name", "1api"},
		{"name in capitals", "S3", "resource_name", "Files"},
		{"empty name", "S3", "resource_name", ""},
		{"long name", "S3", "resource_name", strings.Repeat("a", hclsafe.MaxNameLength+1)},
		{"runtime off the list", "Lambda", "runtime", "nodejs24"},
		{"runtime the pinned provider refuses", "Lambda", "runtime", "nodejs24.x"},
		{"runtime with a quote", "Lambda", "runtime", `provided.al2023"`},
		{"architecture off the list", "Lambda", "architecture", "arm64 "},
		{"environment name with a quote", "Lambda", "environment_variables", map[string]any{`A" = "x`: "v"}},
		{"environment name with an interpolation", "Lambda", "environment_variables", map[string]any{"${x}": "v"}},
		{"code folder with an interpolation", "Lambda", "s3_key", "o/p/d/terraform/code/${x}/bootstrap.zip"},
		{"code folder with a quote", "Lambda", "s3_key", `o/p/d/terraform/code/a"b/bootstrap.zip`},
		{"billing mode off the list", "DynamoDB", "billing_mode", "ON_DEMAND"},
		{"provisioned billing, which the template gives no capacity", "DynamoDB", "billing_mode", "PROVISIONED"},
		{"key type off the list", "DynamoDB", "hash_key_type", "S\"\n"},
		{"method off the list", "API Gateway", "http_methods", "GET,ANY"},
		{"method with a quote", "API Gateway", "http_methods", "GET'\"}"},
		{"method with an interpolation", "API Gateway", "http_methods", "${x}"},
		{"instance type off the list", "EC2", "instance_type", "t4g.micro"},
		{"instance type with a newline", "EC2", "instance_type", "t3.micro\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]any{"resource_name": "ok", tt.field: tt.value}
			tf, errs := generate(t, diskGenerator(t, templateDir), generic.DiagramLayout{Nodes: []generic.DiagramNode{
				node("a", tt.label, vars),
			}})
			if len(errs) != 1 {
				t.Fatalf("errors: %+v (resources %q)", errs, tf.ResourcesTF)
			}
			if errs[0].ErrorType != "invalid_config" {
				t.Fatalf("error type %q: %s", errs[0].ErrorType, errs[0].Message)
			}
			named := tt.field
			if tt.field == "s3_key" {
				named = "uploaded code"
			}
			if !strings.Contains(errs[0].Message, named) {
				t.Fatalf("message does not name %q: %q", named, errs[0].Message)
			}
			if tf.ResourcesTF != "" {
				t.Fatalf("a refused node rendered: %q", tf.ResourcesTF)
			}
		})
	}
}

func TestFreeTextIsEscaped(t *testing.T) {
	tests := []struct {
		label, field, prefix string
	}{
		{"API Gateway", "description", `description = "`},
		{"API Gateway", "stage_name", `stage_name    = "`},
		{"Lambda", "handler", `handler       = "`},
		{"DynamoDB", "hash_key", `hash_key     = "`},
		{"DynamoDB", "ttl_attribute", `attribute_name = "`},
		{"EC2", "key_name", `key_name = "`},
	}
	for _, tt := range tests {
		t.Run(tt.label+" "+tt.field, func(t *testing.T) {
			tf := mustGenerate(t, generic.DiagramLayout{Nodes: []generic.DiagramNode{
				node("a", tt.label, map[string]any{"resource_name": "ok", "enable_ttl": true, tt.field: hostile}),
			}})
			assertEscaped(t, tf.ResourcesTF, tt.prefix)
		})
	}
	t.Run("Lambda environment value", func(t *testing.T) {
		tf := mustGenerate(t, generic.DiagramLayout{Nodes: []generic.DiagramNode{
			node("a", "Lambda", map[string]any{"resource_name": "ok", "environment_variables": map[string]any{"SECRET": hostile}}),
		}})
		assertEscaped(t, tf.ResourcesTF, `SECRET = "`)
	})
}

func assertEscaped(t *testing.T, rendered, prefix string) {
	t.Helper()
	if !strings.Contains(rendered, prefix+hclsafe.Escape(hostile)+`"`) {
		t.Fatalf("no escaped value after %q in:\n%s", prefix, rendered)
	}
	for _, raw := range []string{`"evil"`, "\nresource \"aws_iam_user\"", ` ${data`, "y%{ endif"} {
		if strings.Contains(rendered, raw) {
			t.Fatalf("%q reached the render unescaped:\n%s", raw, rendered)
		}
	}
}

// templatesWithout copies the template folder, replacing one file's content
// (or leaving it out when content is empty).
func templatesWithout(t *testing.T, service, file, content string) string {
	t.Helper()
	dir := t.TempDir()
	err := filepath.WalkDir(templateDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if rel == filepath.Join(service, file) {
			if content == "" {
				return nil
			}
			body = []byte(content)
		}
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(rel)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, rel), body, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAFailedOutputTemplateFailsTheRun(t *testing.T) {
	cases := map[string]string{
		"missing":      "",
		"unparsable":   "output \"{{.ResourceName\" {}",
		"unrenderable": "output \"{{.ResourceName.Nope}}\" {}",
	}
	for _, service := range []string{"apigateway", "lambda", "dynamodb", "s3", "ec2"} {
		label := map[string]string{"apigateway": "API Gateway", "lambda": "Lambda", "dynamodb": "DynamoDB", "s3": "S3", "ec2": "EC2"}[service]
		for name, content := range cases {
			t.Run(service+" "+name, func(t *testing.T) {
				gen := diskGenerator(t, templatesWithout(t, service, "outputs.tf.tmpl", content))
				_, errs := generate(t, gen, generic.DiagramLayout{Nodes: []generic.DiagramNode{
					node("a", label, map[string]any{"resource_name": "ok"}),
				}})
				if len(errs) != 1 || errs[0].ErrorType != internal.ErrorTypeTemplate {
					t.Fatalf("errors: %+v", errs)
				}
				if !strings.Contains(errs[0].Message, "outputs") {
					t.Fatalf("message: %q", errs[0].Message)
				}
			})
		}
	}
}

// TestRenderedTerraformValidates runs terraform validate on rendered diagrams.
// It runs only when CLUTTER_TF_VALIDATE is set and terraform is on the PATH,
// because init downloads the AWS provider. It never plans or applies.
func TestRenderedTerraformValidates(t *testing.T) {
	if os.Getenv("CLUTTER_TF_VALIDATE") == "" {
		t.Skip("set CLUTTER_TF_VALIDATE=1 to run terraform validate on rendered diagrams")
	}
	tfBin, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform is not on the PATH")
	}
	// A plugin cache from the environment saves downloading the provider again.
	pluginCache := os.Getenv("TF_PLUGIN_CACHE_DIR")
	if pluginCache == "" {
		pluginCache = t.TempDir()
	}

	fixtures := map[string]generic.DiagramLayout{
		"all five, CORS on":  allFiveDiagram(true),
		"all five, CORS off": allFiveDiagram(false),
		"API Gateway alone, CORS off": {Nodes: []generic.DiagramNode{
			node("api", "API Gateway", map[string]any{"resource_name": "bare-api", "enable_cors": false}),
		}},
		"defaults": {Nodes: []generic.DiagramNode{
			node("a", "API Gateway", map[string]any{"resource_name": "api"}),
			node("b", "Lambda", map[string]any{"resource_name": "fn"}),
			node("c", "DynamoDB", map[string]any{"resource_name": "table"}),
			node("d", "S3", map[string]any{"resource_name": "bucket"}),
			node("e", "EC2", map[string]any{"resource_name": "vm"}),
		}},
		"hostile free text": {Nodes: []generic.DiagramNode{
			node("a", "API Gateway", map[string]any{"resource_name": "api", "description": hostile, "stage_name": hostile}),
			node("b", "Lambda", map[string]any{"resource_name": "fn", "handler": hostile, "environment_variables": map[string]any{"V": hostile}}),
			node("c", "DynamoDB", map[string]any{"resource_name": "table", "hash_key": hostile, "enable_ttl": true, "ttl_attribute": hostile}),
			node("e", "EC2", map[string]any{"resource_name": "vm", "key_name": hostile}),
		}},
	}
	// Every allowed enumerated value, so validate checks the allow-lists against
	// the provider the generated code pins.
	var everyValue []generic.DiagramNode
	for i, runtime := range []string{"nodejs22.x", "python3.13", "python3.12", "provided.al2023", "provided.al2"} {
		for _, arch := range []string{"arm64", "x86_64"} {
			everyValue = append(everyValue, node(runtime+arch, "Lambda", map[string]any{
				"resource_name": "fn-" + strings.ReplaceAll(arch, "_", "-") + "-" + string(rune('a'+i)), "runtime": runtime, "architecture": arch,
			}))
		}
	}
	for i, mode := range []string{"PAY_PER_REQUEST"} {
		for _, keyType := range []string{"S", "N", "B"} {
			everyValue = append(everyValue, node(mode+keyType, "DynamoDB", map[string]any{
				"resource_name": "t-" + strings.ToLower(keyType) + "-" + string(rune('a'+i)), "billing_mode": mode, "hash_key_type": keyType,
			}))
		}
	}
	fixtures["every allowed value"] = generic.DiagramLayout{Nodes: everyValue}

	for name, layout := range fixtures {
		t.Run(name, func(t *testing.T) {
			tf := mustGenerate(t, layout)
			dir := t.TempDir()
			writeLikeTheWriter(t, dir, tf)
			for _, args := range [][]string{
				{"init", "-backend=false", "-input=false", "-no-color"},
				{"validate", "-no-color"},
			} {
				cmd := exec.Command(tfBin, append([]string{"-chdir=" + dir}, args...)...)
				cmd.Env = append(os.Environ(), "TF_IN_AUTOMATION=1", "TF_PLUGIN_CACHE_DIR="+pluginCache)
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("terraform %s: %v\n%s", args[0], err, out)
				}
				if args[0] == "validate" {
					t.Logf("%s", strings.TrimSpace(string(out)))
				}
			}
		})
	}
}

// TestEdgesGrantReadsAsInlinePolicies keeps A's edges inside what the customer
// role allows: an inline policy on the function's role (the role may not
// create or attach managed policies) that grants read actions only.
func TestEdgesGrantReadsAsInlinePolicies(t *testing.T) {
	iamTF := mustGenerate(t, allFiveDiagram(true)).IAMTF
	for _, want := range []string{
		`resource "aws_iam_role_policy" "orders_handler_to_orders"`,
		`resource "aws_iam_role_policy" "orders_handler_to_order_files"`,
		`role = aws_iam_role.orders_handler_role.id`,
		"s3:GetObject", "s3:ListBucket", "dynamodb:GetItem", "dynamodb:Query",
	} {
		if !strings.Contains(iamTF, want) {
			t.Errorf("iam lacks %q", want)
		}
	}
	for _, refused := range []string{
		`"aws_iam_policy"`, "aws_iam_role_policy_attachment", "policy_arn",
		"s3:PutObject", "s3:DeleteObject", "s3:*",
		"dynamodb:PutItem", "dynamodb:UpdateItem", "dynamodb:DeleteItem", "dynamodb:BatchWriteItem", "dynamodb:*",
	} {
		if strings.Contains(iamTF, refused) {
			t.Errorf("iam holds %q", refused)
		}
	}
}

// TestEveryRoleCarriesTheStackBoundary: every role and instance profile A
// renders is one the customer role may create (path, boundary, tag), the
// provider file declares the boundary's variable, and the deployer derives the
// ARN the stack gives the boundary from the role's ARN.
func TestEveryRoleCarriesTheStackBoundary(t *testing.T) {
	tf := mustGenerate(t, allFiveDiagram(true))
	// The lambda template is stored with CRLF line endings.
	resources := strings.ReplaceAll(tf.ResourcesTF, "\r\n", "\n")
	block := func(kind string) []string {
		return regexp.MustCompile(`(?s)resource "`+kind+`" "[^"]+" \{.*?\n\}`).FindAllString(resources, -1)
	}
	roles, profiles := block("aws_iam_role"), block("aws_iam_instance_profile")
	if len(roles) != 2 || len(profiles) != 1 {
		t.Fatalf("%d roles and %d instance profiles rendered, want the function's and the instance's roles and one profile", len(roles), len(profiles))
	}
	// The customer role may create, change or pass a role only under
	// /clutter/, with the boundary and the tag ManagedBy = Clutter.
	for _, r := range roles {
		for _, want := range []string{
			`path                 = "/clutter/"`,
			"permissions_boundary = var.permissions_boundary_arn",
			`ManagedBy = "Clutter"`,
		} {
			if !strings.Contains(r, want) {
				t.Errorf("a role lacks %q:\n%s", want, r)
			}
		}
	}
	if !strings.Contains(profiles[0], `path = "/clutter/"`) {
		t.Errorf("the instance profile is not under /clutter/:\n%s", profiles[0])
	}
	if !strings.Contains(writer.GenerateMainTF(), `variable "permissions_boundary_arn" {`) {
		t.Error("main.tf does not declare permissions_boundary_arn")
	}
	// The customer role changes or deletes only resources tagged ManagedBy,
	// and requires the tag in every create request: the provider's
	// default_tags put it on everything A's templates render.
	if !regexp.MustCompile(`(?s)default_tags \{\s*tags = \{\s*ManagedBy = "Clutter"`).MatchString(writer.GenerateMainTF()) {
		t.Error("main.tf's provider does not tag every resource ManagedBy = Clutter")
	}

	read := func(rel string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join("../../../../../infra", rel))
		if err != nil {
			t.Fatal(err)
		}
		return strings.ReplaceAll(string(raw), "\r\n", "\n")
	}
	stack := read("modules/s3/uploads/client_side_terraform_deployer_role.yaml")
	for _, want := range []string{
		`RoleName: !Sub "AllowClutterToDeployTerraformRole-${UniqueId}"`,
		`ManagedPolicyName: !Sub "ClutterPermissionsBoundary-${UniqueId}"`,
		"Path: /clutter/",
	} {
		if !strings.Contains(stack, want) {
			t.Errorf("the role stack lacks %q", want)
		}
	}
	deployer := read("modules/fargate/ecr-scripts/terraform-deployer/entrypoint.sh")
	for _, want := range []string{
		`:role/AllowClutterToDeployTerraformRole-([a-z0-9]{8})$'`,
		`:policy/clutter/ClutterPermissionsBoundary-${BASH_REMATCH[3]}"`,
		"export TF_VAR_permissions_boundary_arn",
	} {
		if !strings.Contains(deployer, want) {
			t.Errorf("the deployer entrypoint lacks %q", want)
		}
	}
}

// writeLikeTheWriter lays the files out as writer.WriteToS3 uploads them.
func writeLikeTheWriter(t *testing.T, dir string, tf *internal.GeneratedTerraform) {
	t.Helper()
	main := writer.GenerateMainTF()
	if tf.ResourcesTF != "" {
		main += "\n" + tf.ResourcesTF
	}
	if tf.IAMTF != "" {
		main += "\n" + tf.IAMTF
	}
	for file, content := range map[string]string{"main.tf": main, "outputs.tf": tf.OutputsTF} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
