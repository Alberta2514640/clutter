package aws

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
)

// testSSHKey is a throwaway public key; its private half was never kept.
const testSSHKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAaLzQkB8bkh7I4/uWuVxHsQDSjL4RIq6y+JnLJP5Nn"

// resourceBlock returns one rendered resource block, or "".
func resourceBlock(all, typ, addr string) string {
	return regexp.MustCompile(`(?s)resource "` + regexp.QuoteMeta(typ) + `" "` + regexp.QuoteMeta(addr) + `" \{.*?\n\}`).FindString(all)
}

// TestPrivateServersReachTheInternet: a network splits its range into a
// private and a public subnet; with a private VM and private_internet on (the
// default) it gets a NAT gateway with its address in the public subnet, and
// the private subnet routes through it. Off, or with no private VM, it has
// none.
func TestPrivateServersReachTheInternet(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	p := d.Prefix
	files := renderFor(t, c, fullGraph(), d)
	all := joined(files)
	for _, want := range []string{
		`cidr_block              = "10.1.0.0/25"`,
		`cidr_block              = "10.1.0.128/25"`,
		`resource "aws_eip" "` + p + `_net1_nat"`,
		`resource "aws_nat_gateway" "` + p + `_net1_nat"`,
		`subnet_id     = aws_subnet.` + p + `_net1_public_subnet.id`,
		`nat_gateway_id = aws_nat_gateway.` + p + `_net1_nat.id`,
		`subnet_id      = aws_subnet.` + p + `_net1_subnet.id`,
		// The public VM in the public subnet, the private one in the private.
		`subnet_id      = aws_subnet.` + p + `_net1_public_subnet.id`,
	} {
		if !strings.Contains(all, want) {
			t.Errorf("rendered Terraform lacks %q", want)
		}
	}
	override := string(files["override.tf"])
	web, worker := resourceBlock(override, "aws_instance", p+"_web"), resourceBlock(override, "aws_instance", p+"_worker")
	if !strings.Contains(web, "_net1_public_subnet.id") || !strings.Contains(worker, "aws_subnet."+p+"_net1_subnet.id") {
		t.Errorf("the VMs are not in their subnets:\n%s\n%s", web, worker)
	}

	g := fullGraph()
	g.Blocks["net1"].Fields["private_internet"] = false
	if all := joined(renderFor(t, c, g, d)); strings.Contains(all, "aws_nat_gateway") || strings.Contains(all, "aws_eip") {
		t.Error("a network with private_internet off has a NAT gateway")
	}
	g = fullGraph()
	g.Blocks["worker"].Fields["public"] = true
	if all := joined(renderFor(t, c, g, d)); strings.Contains(all, "aws_nat_gateway") {
		t.Error("a network with no private VM has a NAT gateway")
	}
	// A /28 cannot split into two subnets.
	g = fullGraph()
	g.Blocks["net1"].Fields["cidr"] = "10.1.0.0/28"
	if !contract.HasProblem(c.Validate(g, d), "net1", connectors.CodeInvalidValue) {
		t.Error("a /28 network was accepted")
	}
}

// TestCORSAnswersOnEveryRoute: an API with cors on answers OPTIONS on each
// route's resource (not only the root), its deployment redeploys when that
// answer changes, and A's root answer and unattached validator are gone.
func TestCORSAnswersOnEveryRoute(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	p := d.Prefix
	files := renderFor(t, c, fullGraph(), d)
	all := joined(files)
	iam, override := string(files["iam.tf"]), string(files["override.tf"])

	on, off := p+"_api_to_fn", p+"_api2_to_fn" // api has cors on, api2 off
	for _, want := range []string{
		`resource "aws_api_gateway_method" "` + on + `_options"`,
		`resource_id   = aws_api_gateway_resource.` + on + `.id`,
		`resource "aws_api_gateway_integration_response" "` + on + `_options"`,
		`"method.response.header.Access-Control-Allow-Origin"  = "'*'"`,
	} {
		if !strings.Contains(iam, want) {
			t.Errorf("iam.tf lacks %q", want)
		}
	}
	if strings.Contains(iam, off+"_options") {
		t.Error("an API with cors off answers OPTIONS")
	}
	if strings.Contains(all, "root_resource_id\n  http_method   = \"OPTIONS\"") || strings.Contains(all, p+"_api_options") {
		t.Error("A's root-only CORS answer is still rendered")
	}
	if strings.Contains(all, "aws_api_gateway_request_validator") {
		t.Error("a request validator that no method uses is rendered")
	}
	dep := resourceBlock(override, "aws_api_gateway_deployment", p+"_api_deployment")
	// A renamed function changes its route's path and target in place, with
	// the same ids, so both are in the trigger or the stage keeps the old route.
	for _, want := range []string{
		"aws_api_gateway_method." + on + "_options.id", "aws_api_gateway_integration_response." + on + "_options.id",
		"aws_api_gateway_resource." + on + ".path_part", "aws_api_gateway_integration." + on + ".uri",
	} {
		if !strings.Contains(dep, want) {
			t.Errorf("the deployment's trigger lacks %s:\n%s", want, dep)
		}
	}
	if dep2 := resourceBlock(override, "aws_api_gateway_deployment", p+"_api2_deployment"); strings.Contains(dep2, "_options") {
		t.Errorf("the cors-off API's trigger names a CORS answer:\n%s", dep2)
	}
}

func TestEveryAPIIsThrottled(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	defaults := string(renderFor(t, c, fullGraph(), d)["defaults.tf"])
	for _, api := range []string{d.Prefix + "_api", d.Prefix + "_api2"} {
		b := resourceBlock(defaults, "aws_api_gateway_method_settings", api+"_throttle")
		for _, want := range []string{
			"stage_name  = aws_api_gateway_stage." + api + "_stage[0].stage_name",
			`method_path = "*/*"`,
			"throttling_rate_limit  = 100",
			"throttling_burst_limit = 200",
		} {
			if !strings.Contains(b, want) {
				t.Errorf("%s throttling lacks %q:\n%s", api, want, b)
			}
		}
	}
}

// TestFunctionsGetTheNamesTheyRead: a function that reads a bucket and a
// table gets their names and region as environment variables, as a VM does.
func TestFunctionsGetTheNamesTheyRead(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	fn := resourceBlock(string(renderFor(t, c, fullGraph(), d)["main.tf"]), "aws_lambda_function", d.Prefix+"_fn")
	for _, want := range []string{
		`CLUTTER_UPLOADS_BUCKET = "${aws_s3_bucket.` + d.Prefix + `_bucket.bucket}"`,
		`CLUTTER_UPLOADS_REGION = "ca-central-1"`,
		`CLUTTER_ORDERS_TABLE = "${aws_dynamodb_table.` + d.Prefix + `_table.name}"`,
		`CLUTTER_ORDERS_REGION = "ca-central-1"`,
	} {
		if !strings.Contains(fn, want) {
			t.Errorf("the function lacks %q:\n%s", want, fn)
		}
	}
}

// TestVMToVMOpensAPath: the VM that gets another's address may reach it,
// through a security group on the target that lets in the source's private
// address; VMs in different networks are refused.
func TestVMToVMOpensAPath(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	p := d.Prefix
	g := fullGraph()
	g.Edges = append(g.Edges, connectors.Edge{Source: "web", Target: "worker"})
	files := renderFor(t, c, g, d)
	iam, override := string(files["iam.tf"]), string(files["override.tf"])
	group := p + "_worker_peers"
	for _, want := range []string{
		`resource "aws_security_group" "` + group + `"`,
		`name        = "` + p + `-worker.peers"`,
		`vpc_id      = aws_vpc.` + p + `_net1.id`,
		`cidr_ipv4         = "${aws_instance.` + p + `_web.private_ip}/32"`,
		`resource "aws_vpc_security_group_egress_rule" "` + group + `_out_all"`,
	} {
		if !strings.Contains(iam, want) {
			t.Errorf("iam.tf lacks %q", want)
		}
	}
	worker := resourceBlock(override, "aws_instance", p+"_worker")
	if !strings.Contains(worker, "aws_security_group."+p+"_fw.id, aws_security_group."+group+".id") {
		t.Errorf("the target VM is not in its peer group:\n%s", worker)
	}
	if strings.Contains(resourceBlock(override, "aws_instance", p+"_web"), "_peers") {
		t.Error("the source VM got a peer group")
	}

	g = fullGraph()
	addNetwork(&g, "net2")
	for i, e := range g.Edges {
		if e.Source == "worker" && e.Target == "net1" {
			g.Edges[i].Target = "net2"
		}
	}
	g.Edges = append(g.Edges, connectors.Edge{Source: "web", Target: "worker"})
	found := false
	for _, pr := range c.Validate(g, d) {
		if pr.Code == connectors.CodePeerUnreachable && pr.EdgeSource == "web" && pr.EdgeTarget == "worker" {
			found = true
		}
	}
	if !found {
		t.Error("VMs in different networks were accepted as peers")
	}
}

// TestSafeDefaults: an encrypted root disk and IMDSv2 on every VM, the
// block's name as its Name tag, a log group kept 30 days that the function
// logs to, point-in-time recovery on every table, and a TLS-only policy on
// every bucket.
func TestSafeDefaults(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	p := d.Prefix
	files := renderFor(t, c, fullGraph(), d)
	override, defaults := string(files["override.tf"]), string(files["defaults.tf"])

	web := resourceBlock(override, "aws_instance", p+"_web")
	for _, want := range []string{
		"root_block_device {\n    encrypted   = true",
		`http_tokens   = "required"`,
		`Name      = "` + p + `-web"`,
		`ManagedBy = "Ansible"`,
	} {
		if !strings.Contains(web, want) {
			t.Errorf("the VM lacks %q:\n%s", want, web)
		}
	}
	for _, want := range []string{
		`name              = "/aws/lambda/` + p + `-api_handler"`,
		"retention_in_days = 30",
		`resource "aws_s3_bucket_policy" "` + p + `_bucket_tls_only"`,
		`Bool = { "aws:SecureTransport" = "false" }`,
	} {
		if !strings.Contains(defaults, want) {
			t.Errorf("defaults.tf lacks %q", want)
		}
	}
	for _, want := range []string{
		"log_group  = aws_cloudwatch_log_group." + p + "_fn.name",
		"point_in_time_recovery {\n    enabled = true",
		`resource "aws_s3_bucket" "` + p + `_bucket" {` + "\n  # The block's",
		"force_destroy = false",
	} {
		if !strings.Contains(override, want) {
			t.Errorf("override.tf lacks %q", want)
		}
	}
	g := fullGraph()
	g.Blocks["bucket"].Fields["force_destroy"] = true
	if !strings.Contains(string(renderFor(t, c, g, d)["override.tf"]), "force_destroy = true") {
		t.Error("the bucket's force_destroy field does not reach the bucket")
	}
}

// TestSSHKeyBecomesAKeyPair: a VM's public key becomes a key pair the
// instance names; a bad key is a problem on the field.
func TestSSHKeyBecomesAKeyPair(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	p := d.Prefix
	g := fullGraph()
	g.Blocks["web"].Fields[connectors.FieldSSHKey] = testSSHKey + " me@laptop\n"
	files := renderFor(t, c, g, d)
	kp := resourceBlock(string(files["defaults.tf"]), "aws_key_pair", p+"_web")
	if !strings.Contains(kp, `key_name   = "`+p+`-web"`) || !strings.Contains(kp, `public_key = sensitive("`+testSSHKey+`")`) || strings.Contains(kp, "laptop") {
		t.Errorf("key pair:\n%s", kp)
	}
	if !strings.Contains(resourceBlock(string(files["override.tf"]), "aws_instance", p+"_web"), "key_name                    = aws_key_pair."+p+"_web.key_name") {
		t.Error("the instance does not name its key pair")
	}
	if strings.Contains(resourceBlock(string(files["override.tf"]), "aws_instance", p+"_worker"), "key_name") {
		t.Error("a VM with no key names a key pair")
	}
	g.Blocks["web"].Fields[connectors.FieldSSHKey] = "ssh-ed25519 ${file(\"/etc/passwd\")}"
	if !contract.HasProblem(c.Validate(g, d), "web", connectors.CodeInvalidValue) {
		t.Error("a bad key was accepted")
	}
}

// TestTheRoleAllowsTheSafeDefaults: the customer role allows what the NAT
// gateway, key pairs, log groups, point-in-time recovery, bucket policies
// and force_destroy need, and a bucket policy or an object delete only on
// B's buckets.
func TestTheRoleAllowsTheSafeDefaults(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(backendDir(), "infra", "modules", "s3", "uploads", "client_side_terraform_deployer_role.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ReplaceAll(string(raw), "\r\n", "\n")
	for _, want := range []string{
		"ec2:AllocateAddress", "ec2:ReleaseAddress", "ec2:CreateNatGateway", "ec2:DeleteNatGateway", "ec2:DescribeNatGateways",
		"ec2:ImportKeyPair", "ec2:DeleteKeyPair", "ec2:ModifyInstanceMetadataOptions",
		"logs:CreateLogGroup", "logs:DeleteLogGroup", "logs:PutRetentionPolicy", "logs:TagResource",
		"dynamodb:UpdateContinuousBackups", "s3:PutBucketPolicy", "s3:DeleteBucketPolicy", "s3:DeleteObjectVersion",
	} {
		if !strings.Contains(body, "- "+want+"\n") && !strings.Contains(body, "Action: "+want+"\n") {
			t.Errorf("the role does not allow %s", want)
		}
	}
	for _, sid := range []string{"S3DeploymentBucketPolicies", "S3EmptyDeploymentBuckets"} {
		stmt := regexp.MustCompile(`(?s)- Sid: ` + sid + `\n.*?\n\n`).FindString(body)
		if !strings.Contains(stmt, `:s3:::c????????-*`) || strings.Contains(stmt, `:s3:::*"`) {
			t.Errorf("%s reaches buckets other than B's:\n%s", sid, stmt)
		}
	}
}

// B offers no runtime AWS deprecated, though A keeps it.
func TestNoDeprecatedRuntime(t *testing.T) {
	for _, s := range manifest().Settings {
		if s.Name == settingRuntime && slices.Contains(s.Allowed, "provided.al2") {
			t.Fatalf("B offers provided.al2: %v", s.Allowed)
		}
	}
}
