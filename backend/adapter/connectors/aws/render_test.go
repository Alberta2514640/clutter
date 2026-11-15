package aws

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
)

// TestEveryRoleCarriesTheBoundary: the customer role may create a role only
// under /clutter/, with the stack's permissions boundary and the tag
// ManagedBy = Clutter, and pass only roles under /clutter/. So every role and
// instance profile a run renders has that path, every role the boundary and
// the tag, and the boundary's variable is declared for every run, a destroy
// included.
func TestEveryRoleCarriesTheBoundary(t *testing.T) {
	c, _, _ := newTestConnector(t)
	all := joined(renderFor(t, c, fullGraph(), testDeployment("ca-central-1")))
	block := func(kind string) []string {
		return regexp.MustCompile(`(?s)resource "`+kind+`" "[^"]+" \{.*?\n\}`).FindAllString(all, -1)
	}
	roles, profiles := block("aws_iam_role"), block("aws_iam_instance_profile")
	if len(roles) < 3 || len(profiles) < 2 {
		t.Fatalf("only %d roles and %d instance profiles rendered; the pattern no longer matches the output", len(roles), len(profiles))
	}
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
	for _, p := range profiles {
		if !strings.Contains(p, `path = "/clutter/"`) {
			t.Errorf("an instance profile is not under /clutter/:\n%s", p)
		}
	}
	provider, err := providerFile(testDeployment("ca-central-1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`variable "permissions_boundary_arn" {`, `variable "user_permissions_boundary_arn" {`, `ManagedBy         = "Clutter"`} {
		if !strings.Contains(string(provider), want) {
			t.Errorf("the provider file lacks %q", want)
		}
	}
}

// TestBoundaryMatchesTheRoleStack keeps boundaryARN and the CloudFormation
// stack in step: the stack names the boundary after the same UniqueId as the
// role, under /clutter/, and the role may create or change a role only with it.
func TestBoundaryMatchesTheRoleStack(t *testing.T) {
	got, err := boundaryARN(goodRole)
	if want := "arn:aws:iam::123456789012:policy/clutter/ClutterPermissionsBoundary-ab12cd34"; err != nil || got != want {
		t.Fatalf("boundaryARN(%s) = %q, %v; want %q", goodRole, got, err, want)
	}
	got, err = userBoundaryARN(goodRole)
	if want := "arn:aws:iam::123456789012:policy/clutter/ClutterUserBoundary-ab12cd34"; err != nil || got != want {
		t.Fatalf("userBoundaryARN(%s) = %q, %v; want %q", goodRole, got, err, want)
	}
	if _, err := userBoundaryARN("arn:aws:iam::123456789012:role/SomeOtherRole"); err == nil {
		t.Error("userBoundaryARN accepted a role the stack did not create")
	}
	for _, role := range []string{
		"arn:aws:iam::123456789012:role/SomeOtherRole",
		"arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-abc",
		"arn:aws:iam::123456789012:role/path/AllowClutterToDeployTerraformRole-ab12cd34",
		"arn:aws-us-gov:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd34",
		goodRole + "\n",
	} {
		var userErr *connectors.UserError
		if _, err := boundaryARN(role); !errors.As(err, &userErr) {
			t.Errorf("boundaryARN(%q) = %v, want a user error", role, err)
		}
	}

	raw, err := os.ReadFile(filepath.Join(backendDir(), "infra", "modules", "s3", "uploads", "client_side_terraform_deployer_role.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ReplaceAll(string(raw), "\r\n", "\n")
	for _, want := range []string{
		`RoleName: !Sub "AllowClutterToDeployTerraformRole-${UniqueId}"`,
		`ManagedPolicyName: !Sub "ClutterPermissionsBoundary-${UniqueId}"`,
		`ManagedPolicyName: !Sub "ClutterUserBoundary-${UniqueId}"`,
		"iam:PermissionsBoundary: !Ref UserPermissionsBoundary",
		// The users' boundary reaches only buckets and tables named with a
		// deployment prefix (connectors.PrefixFor).
		`:s3:::c????????-*/*"`,
		`:table/c????????-*"`,
		"Path: /clutter/",
		"iam:PermissionsBoundary: !Ref PermissionsBoundary",
		"- iam:DeleteRolePermissionsBoundary\n                - iam:DeleteUserPermissionsBoundary",
		"aws:RequestTag/ManagedBy: Clutter",
		`:role/clutter/*-lambda-role"`,
		`:role/clutter/*-ssm-role"`,
		`:instance-profile/clutter/*-ssm-profile"`,
		`:user/clutter/*"`,
		"aws:ResourceTag/ManagedBy: Clutter",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the role template lacks %q", want)
		}
	}
	// Every role, instance profile or user the stack's role may touch is
	// under /clutter/, so it never reaches one the account's owner made.
	for _, m := range regexp.MustCompile(`:(role|instance-profile|user)/([^"]*)"`).FindAllStringSubmatch(body, -1) {
		if m[1] == "role" && (m[2] == "TerraformDeployerFargateTaskRole" || m[2] == "${AdapterRunnerRoleName}") {
			continue // the trusted principals in Clutter's own account
		}
		if !strings.HasPrefix(m[2], "clutter/") {
			t.Errorf("the role template names %s/%s outside /clutter/", m[1], m[2])
		}
	}
	for _, refused := range []string{"iam:UpdateAssumeRolePolicy", "iam:PutRolePermissionsBoundary", "iam:PutUserPermissionsBoundary",
		"iam:CreateLoginProfile", "iam:AddUserToGroup", "iam:AttachUserPolicy", "iam:UpdateAccessKey"} {
		if strings.Contains(body, refused) {
			t.Errorf("the role template allows %s", refused)
		}
	}
}

// TestRendersEveryTemplateFromDisk renders A's five template folders and B's
// three from the repo, with missingkey=error, through the diagram that uses
// every block and edge. A change to A's templates that B's variable maps no
// longer satisfy fails here, in CI.
func TestRendersEveryTemplateFromDisk(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	all := joined(renderFor(t, c, fullGraph(), d))
	p := d.Prefix

	for _, want := range []string{
		// One primary resource per block kind, addressed by prefix and node id.
		`resource "aws_vpc" "` + p + `_net1"`,
		`resource "aws_security_group" "` + p + `_fw"`,
		`resource "aws_instance" "` + p + `_web"`,
		`resource "aws_ebs_volume" "` + p + `_dndnode-3-`,
		`resource "aws_s3_bucket" "` + p + `_bucket"`,
		`resource "aws_lambda_function" "` + p + `_fn"`,
		`resource "aws_dynamodb_table" "` + p + `_table"`,
		`resource "aws_api_gateway_rest_api" "` + p + `_api"`,
		// Overrides: the VM in its subnet and security group, public.
		`subnet_id                   = aws_subnet.` + p + `_net1_subnet.id`,
		`vpc_security_group_ids      = [aws_security_group.` + p + `_fw.id]`,
		`associate_public_ip_address = true`,
		// A public VM gives its network an internet gateway.
		`resource "aws_internet_gateway" "` + p + `_net1_igw"`,
		// The firewall is in the VMs' network, with the parsed rules.
		`vpc_id      = aws_vpc.` + p + `_net1.id`,
		`from_port         = 1000`,
		`cidr_ipv4         = "203.0.113.0/24"`,
		// Two volumes on one VM get two devices.
		`device_name = "/dev/sdf"`,
		`device_name = "/dev/sdg"`,
		// Fields and settings reach the templates.
		`instance_type        = "t3.small"`,
		`runtime       = "python3.12"`,
		`handler       = "lambda_function.lambda_handler"`,
		`memory_size      = 256`,
		`hash_key     = "order_id"`,
		`type = "N"`,
		`status = "Enabled"`,
		`region = "ca-central-1"`,
		`ClutterDeployment = "` + d.ID + `"`,
	} {
		if !strings.Contains(all, want) {
			t.Errorf("rendered Terraform lacks %q", want)
		}
	}
	if strings.Contains(all, "<no value>") {
		t.Error("a template read a variable B does not set")
	}
}

// TestEdgesBecomeWorkingIAMAndRoutes covers A's four IAM defects: B's HTTP API
// to function edge routes (resource, method, integration, permission), each
// API has its own statement id, every function can write logs, and "reads"
// grants read actions only.
func TestEdgesBecomeWorkingIAMAndRoutes(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	files := renderFor(t, c, fullGraph(), d)
	iam, override := string(files["iam.tf"]), string(files["override.tf"])
	p := d.Prefix

	for _, api := range []string{p + "_api", p + "_api2"} {
		route := api + "_to_fn"
		for _, want := range []string{
			`resource "aws_api_gateway_resource" "` + route + `"`,
			`resource "aws_api_gateway_method" "` + route + `"`,
			`resource "aws_api_gateway_integration" "` + route + `"`,
			`uri                     = aws_lambda_function.` + p + `_fn.invoke_arn`,
			`resource "aws_lambda_permission" "` + route + `"`,
			`statement_id  = "AllowInvokeFrom-` + api + `"`,
		} {
			if !strings.Contains(iam, want) {
				t.Errorf("iam.tf lacks %q", want)
			}
		}
		if !strings.Contains(override, `resource "aws_api_gateway_deployment" "`+api+`_deployment"`) ||
			!strings.Contains(override, `aws_api_gateway_integration.`+route+`.id`) {
			t.Errorf("the %s deployment does not wait for its route", api)
		}
	}
	if n := strings.Count(iam, "statement_id"); n != 2 {
		t.Errorf("%d statement ids, want 2", n)
	}
	// Logs go only to the function's own group, which defaults.tf creates
	// with a retention; the role cannot create a group or write elsewhere.
	if !strings.Contains(iam, `resource "aws_iam_role_policy" "`+p+`_fn_logs"`) ||
		!strings.Contains(iam, `Resource = "${aws_cloudwatch_log_group.`+p+`_fn.arn}:*"`) ||
		strings.Contains(iam, "logs:CreateLogGroup") || strings.Contains(iam, "AWSLambdaBasicExecutionRole") {
		t.Error("the function's log permission is not scoped to its own log group")
	}
	for _, write := range []string{"s3:PutObject", "s3:DeleteObject", "s3:*", "dynamodb:PutItem", "dynamodb:UpdateItem", "dynamodb:DeleteItem", "dynamodb:*"} {
		if strings.Contains(iam, write) {
			t.Errorf("a reads edge grants %s", write)
		}
	}
	for _, read := range []string{"s3:GetObject", "s3:ListBucket", "dynamodb:GetItem", "dynamodb:Query"} {
		if !strings.Contains(iam, read) {
			t.Errorf("a reads edge does not grant %s", read)
		}
	}
	if !strings.Contains(string(files["code.tf"]), `data "archive_file" "`+p+`_fn_code"`) ||
		!strings.Contains(override, `source_code_hash = data.archive_file.`+p+`_fn_code.output_base64sha256`) {
		t.Error("the function has no placeholder code")
	}
}

var (
	addressPattern = regexp.MustCompile(`(?m)^(?:resource|data) "([^"]+)" "([^"]+)"`)
	cloudName      = regexp.MustCompile(`(?m)^\s*(name|bucket|function_name|Name|statement_id)\s*=\s*"([^"]*)"`)
)

// fixedNames are names A's templates hard-code. None is a name in the
// account: the SSM parameter is AWS's own, which the ec2 template reads.
var fixedNames = map[string]bool{
	"/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64": true,
}

// TestTwoDeploymentsShareNoName renders one diagram for two deployments in one
// account: every Terraform address and every cloud-side name carries the
// deployment's prefix, and the two share none.
func TestTwoDeploymentsShareNoName(t *testing.T) {
	c, _, _ := newTestConnector(t)
	seen := map[string]string{}
	for _, d := range []connectors.Deployment{testDeployment("ca-central-1"), testDeployment("ca-central-1")} {
		all := joined(renderFor(t, c, fullGraph(), d))
		var names []string
		for _, m := range addressPattern.FindAllStringSubmatch(all, -1) {
			if !strings.HasPrefix(m[2], d.Prefix+"_") {
				t.Errorf("address %s.%s lacks the prefix %s", m[1], m[2], d.Prefix)
			}
			names = append(names, m[1]+"."+m[2])
		}
		for _, m := range cloudName.FindAllStringSubmatch(all, -1) {
			// order_id is the table's key attribute, not a name in the account.
			if fixedNames[m[2]] || m[2] == "order_id" {
				continue
			}
			if !strings.Contains(m[2], d.Prefix) {
				t.Errorf("cloud name %s = %q lacks the prefix %s", m[1], m[2], d.Prefix)
			}
			names = append(names, m[2])
		}
		if len(names) < 30 {
			t.Fatalf("only %d names found; the patterns no longer match the output", len(names))
		}
		for _, n := range names {
			if other, ok := seen[n]; ok && other != d.ID {
				t.Errorf("two deployments share %q", n)
			}
			seen[n] = d.ID
		}
	}
}

func TestNamesKeepAWSLimits(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	long := strings.Repeat("a", 64)
	longID := strings.Repeat("N", 64)
	g := connectors.Graph{Blocks: map[string]connectors.Block{
		longID: {ID: longID, Kind: connectors.KindFunction, Name: long, Fields: map[string]any{}},
		"vm":   {ID: "vm", Kind: connectors.KindVM, Name: strings.Repeat("v", 64), Fields: map[string]any{}},
		"b":    {ID: "b", Kind: connectors.KindBucket, Name: "a_" + strings.Repeat("b", 62), Fields: map[string]any{}},
	}}
	all := joined(renderFor(t, c, g, d))

	for _, m := range cloudName.FindAllStringSubmatch(all, -1) {
		if fixedNames[m[2]] {
			continue
		}
		limit := 64
		switch m[1] {
		case "function_name":
			limit = maxFunctionName
		case "bucket":
			limit = 63
		}
		v := strings.Replace(m[2], "${random_id."+d.Prefix+"_b_suffix.hex}", "12345678", 1)
		if len(v) > limit {
			t.Errorf("%s %q is %d characters, over %d", m[1], v, len(v), limit)
		}
	}
	for _, m := range addressPattern.FindAllStringSubmatch(all, -1) {
		if m[1] == "aws_instance" && len(m[2]+"-ssm-role") > 64 {
			t.Errorf("the VM's IAM role name %s-ssm-role is over 64 characters", m[2])
		}
	}
	if !regexp.MustCompile(`bucket = "` + d.Prefix + `-a-b+-[0-9a-f]{8}-\$\{`).MatchString(all) {
		t.Error("the bucket name is not lowercase letters, digits and hyphens with a hash")
	}
}

func TestExclusionsLeaveBlocksAndTheirWiringOut(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	d.Exclusions = []string{"bucket", "table", "fn", "api", "api2"}
	all := joined(renderFor(t, c, fullGraph(), d))
	for _, gone := range []string{"aws_lambda_function", "aws_s3_bucket", "aws_dynamodb_table", "aws_api_gateway", `"aws_iam_role_policy"`, "archive_file"} {
		if strings.Contains(all, gone) {
			t.Errorf("an excluded block left %s behind", gone)
		}
	}
	if !strings.Contains(all, `resource "aws_instance" "`+d.Prefix+`_web"`) {
		t.Error("the included VM is missing")
	}

	// Leaving out what a block depends on is a problem, not a broken render.
	cases := map[string]struct {
		exclude []string
		code    string
		node    string
	}{
		"API without its function": {[]string{"fn"}, CodeAPIWithoutRoute, "api"},
		"volume without its VM":    {[]string{"web"}, CodeVolumeNeedsVM, "logs"},
	}
	for name, tc := range cases {
		d := testDeployment("ca-central-1")
		d.Exclusions = tc.exclude
		if !contract.HasProblem(c.Validate(fullGraph(), d), tc.node, tc.code) {
			t.Errorf("%s: no %s problem on %s", name, tc.code, tc.node)
		}
	}
}

func TestValidateRefusesWhatAWSCannotBuild(t *testing.T) {
	c, _, _ := newTestConnector(t)
	type edit func(g *connectors.Graph, d *connectors.Deployment)
	field := func(id, name string, v any) edit {
		return func(g *connectors.Graph, _ *connectors.Deployment) { g.Blocks[id].Fields[name] = v }
	}
	cases := map[string]struct {
		edit edit
		code string
		node string
	}{
		"network range too big":         {field("net1", "cidr", "10.0.0.0/8"), connectors.CodeInvalidValue, "net1"},
		"network range IPv6":            {field("net1", "cidr", "fd00::/64"), connectors.CodeInvalidValue, "net1"},
		"network range off its address": {field("net1", "cidr", "10.0.0.5/24"), connectors.CodeInvalidValue, "net1"},
		"port out of range":             {field("fw", "ingress", "tcp/70000"), connectors.CodeInvalidValue, "fw"},
		"unknown protocol":              {field("fw", "ingress", "icmp/0"), connectors.CodeInvalidValue, "fw"},
		"bad source":                    {field("fw", "source_cidr", "anywhere"), connectors.CodeInvalidValue, "fw"},
		"source off its address":        {field("fw", "source_cidr", "203.0.113.5/24"), connectors.CodeInvalidValue, "fw"},
		"volume too big":                {field("logs", "size_gb", float64(20000)), connectors.CodeInvalidValue, "logs"},
		"memory too small":              {field("fn", "memory_mb", float64(64)), connectors.CodeInvalidValue, "fn"},
		"table key with a quote":        {field("table", "key", `id"`), connectors.CodeInvalidValue, "table"},
		"duplicate name": {func(g *connectors.Graph, _ *connectors.Deployment) {
			b := g.Blocks["worker"]
			b.Name = "web"
			g.Blocks["worker"] = b
		}, connectors.CodeDuplicateName, "worker"},
		"vm on two networks": {func(g *connectors.Graph, _ *connectors.Deployment) {
			addNetwork(g, "net2")
			g.Edges = append(g.Edges, connectors.Edge{Source: "web", Target: "net2"})
		}, CodeVMTwoNetworks, "web"},
		"firewall across networks": {func(g *connectors.Graph, _ *connectors.Deployment) {
			addNetwork(g, "net2")
			for i, e := range g.Edges {
				if e.Source == "worker" {
					g.Edges[i].Target = "net2"
				}
			}
		}, CodeFirewallNetworks, "fw"},
		"API with no function": {func(g *connectors.Graph, _ *connectors.Deployment) {
			g.Blocks["api3"] = connectors.Block{ID: "api3", Kind: connectors.KindHTTPAPI, Name: "lonely", Fields: map[string]any{}}
		}, CodeAPIWithoutRoute, "api3"},
		"region not offered": {func(_ *connectors.Graph, d *connectors.Deployment) { d.Connection.Region = "mars-north-1" }, connectors.CodeDeploymentInvalid, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			g, d := fullGraph(), testDeployment("ca-central-1")
			tc.edit(&g, &d)
			if ps := c.Validate(g, d); !contract.HasProblem(ps, tc.node, tc.code) {
				t.Fatalf("no %s problem on %q: %+v", tc.code, tc.node, ps)
			}
		})
	}
}

// TestHostileTextNeverBecomesTerraform puts text that would break out of an
// HCL string, interpolate, or name a refused feature into every place a user
// can type, and checks it is refused or arrives as plain text.
func TestHostileTextNeverBecomesTerraform(t *testing.T) {
	c, _, _ := newTestConnector(t)
	breakout := "x\" }\nresource \"aws_iam_user\" \"pwn\" { name = \"${file(\"/etc/passwd\")}\" %{ if true }"

	g, d := fullGraph(), testDeployment("ca-central-1")
	g.Blocks["api"].Fields["description"] = breakout
	all := joined(renderFor(t, c, g, d))
	if strings.Contains(all, `"aws_iam_user"`) && !strings.Contains(all, `\"aws_iam_user\"`) {
		t.Fatal("the description broke out of its string")
	}
	if regexp.MustCompile(`(^|[^$])\$\{file|(^|[^%])%\{ if`).MatchString(all) {
		t.Fatal("an interpolation or directive reached the Terraform unescaped")
	}
	if !strings.Contains(all, `$${file(\"/etc/passwd\")}`) {
		t.Fatal("the description was not escaped as expected")
	}

	// A description that names a refused feature is plain text inside a
	// string, so the runner's check, which reads blocks, accepts it.
	g, d = fullGraph(), testDeployment("ca-central-1")
	g.Blocks["api"].Fields["description"] = `runs a local-exec step, module x, provisioner "local-exec" {}`
	if all := joined(renderFor(t, c, g, d)); !strings.Contains(all, "module x") {
		t.Fatal("the description did not arrive as text")
	}

	refused := map[string]func(g *connectors.Graph, d *connectors.Deployment){
		"name with a quote": func(g *connectors.Graph, _ *connectors.Deployment) {
			b := g.Blocks["fn"]
			b.Name = `a"b`
			g.Blocks["fn"] = b
		},
		"name with interpolation": func(g *connectors.Graph, _ *connectors.Deployment) {
			b := g.Blocks["bucket"]
			b.Name = "${x}"
			g.Blocks["bucket"] = b
		},
		"ingress with a quote": func(g *connectors.Graph, _ *connectors.Deployment) { g.Blocks["fw"].Fields["ingress"] = `tcp/22"` },
		"source with a newline": func(g *connectors.Graph, _ *connectors.Deployment) {
			g.Blocks["fw"].Fields["source_cidr"] = "0.0.0.0/0\n}"
		},
		"cidr with interpolation":   func(g *connectors.Graph, _ *connectors.Deployment) { g.Blocks["net1"].Fields["cidr"] = "${x}" },
		"instance type not allowed": func(_ *connectors.Graph, d *connectors.Deployment) { d.Settings["instance_type"] = `t3.micro"` },
		"runtime not allowed":       func(_ *connectors.Graph, d *connectors.Deployment) { d.Settings["function_runtime"] = "bash" },
		"region with a quote":       func(_ *connectors.Graph, d *connectors.Deployment) { d.Connection.Region = `ca-central-1"` },
	}
	for name, edit := range refused {
		t.Run(name, func(t *testing.T) {
			g, d := fullGraph(), testDeployment("ca-central-1")
			edit(&g, &d)
			// B's core runs both checks before any render.
			if ps := capability.CheckConnector(c, g, d).Problems; len(ps) == 0 {
				t.Fatal("accepted")
			}
		})
	}

	// A node id that translate would refuse still cannot reach an address.
	g, d = fullGraph(), testDeployment("ca-central-1")
	hostileID := `x" { }`
	b := g.Blocks["bucket"]
	delete(g.Blocks, "bucket")
	b.ID = hostileID
	g.Blocks[hostileID] = b
	for i, e := range g.Edges {
		if e.Target == "bucket" {
			g.Edges[i].Target = hostileID
		}
	}
	all = joined(renderFor(t, c, g, d))
	if strings.Contains(all, hostileID) {
		t.Fatal("a node id reached the Terraform as is")
	}
}

func addNetwork(g *connectors.Graph, id string) {
	g.Blocks[id] = connectors.Block{ID: id, Kind: connectors.KindNetwork, Name: id, Fields: map[string]any{"cidr": "10.2.0.0/24"}}
}
