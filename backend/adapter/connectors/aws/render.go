package aws

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
)

// TemplateSource loads a customer template by its bucket path, such as
// "templates/aws/ec2/main.tf.tmpl". *tftemplate.TemplateLoader is one: it
// reads the templates bucket in the cloud, or a folder on disk
// (tftemplate.NewTemplateLoaderFromDir) locally and in tests.
type TemplateSource interface {
	Load(ctx context.Context, templatePath string) (*tftemplate.Template, error)
}

//go:embed tmpl/*.tf.tmpl
var ownTemplates embed.FS

// sampleMessage is what a function answers until code is uploaded for it.
const sampleMessage = "Hello from Clutter. Upload your code to replace this."

// placeholderCode is the one file a function gets when Clutter has no sample
// code for its runtime and handler: it does not answer.
const placeholderCode = "Clutter has no sample code for this runtime and handler. Upload your code to run the function."

// The sample's answer names the tables and buckets the function's links hand
// it (functionEnv), read from its environment at run time, so the answer
// shows what the function received: {"message": "...", "table": "<name>"}, or
// "tables": [...] when it has several. The variables end in these suffixes
// (connectors.EnvName).
var (
	sampleTableSuffix  = "_" + strings.ToUpper(connectors.ValueTable)
	sampleBucketSuffix = "_" + strings.ToUpper(connectors.ValueBucket)
)

// sampleMessageJSON is sampleMessage as a JSON string. It holds no quote,
// backslash or dollar, so each sample can carry it as plain text.
var sampleMessageJSON = func() string {
	b, _ := json.Marshal(sampleMessage)
	if strings.ContainsAny(sampleMessage, `'"\$`) {
		panic("aws: the sample message holds a quote, a backslash or a dollar")
	}
	return string(b)
}()

// sampleHandler is a handler the Python and Node.js samples can serve: one
// file name and one function name, both plain identifiers.
var sampleHandler = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)$`)

// sampleFile is one file of a function's sample code.
type sampleFile struct{ Name, Content string }

// sampleCode is the code a function with no upload runs: a few lines that
// answer every request with status 200 and a JSON body (the shape API
// Gateway's proxy integration reads) holding sampleMessage and the names of
// its linked tables and buckets, for the deployment's runtime and the
// function's handler. A custom runtime (provided.*) runs its bootstrap file
// whatever the handler; it uses only sh, env and curl, which the
// provided.al2023 image holds (AWS's custom runtime tutorial uses the same).
// ok is false for a runtime or handler Clutter has no sample for.
func sampleCode(runtime, handler string) (files []sampleFile, ok bool) {
	switch {
	case strings.HasPrefix(runtime, "provided."):
		return []sampleFile{{Name: "bootstrap", Content: `#!/bin/sh
# Clutter's sample code. Upload your own code to replace it.
set -eu
api="http://${AWS_LAMBDA_RUNTIME_API}/2018-06-01/runtime/invocation"
# The answer names the tables and buckets this function's links hand it.
q='\"'
tables=""
ntables=0
buckets=""
nbuckets=0
set -f
for pair in $(env); do
  value="${pair#*=}"
  case "$value" in
    "" | *[!A-Za-z0-9._-]*) continue ;;
  esac
  case "${pair%%=*}" in
    CLUTTER_*` + sampleTableSuffix + `)
      tables="$tables${tables:+,}$q$value$q"
      ntables=$((ntables + 1))
      ;;
    CLUTTER_*` + sampleBucketSuffix + `)
      buckets="$buckets${buckets:+,}$q$value$q"
      nbuckets=$((nbuckets + 1))
      ;;
  esac
done
set +f
body="{${q}message${q}:${q}` + sampleMessage + `${q}"
if [ "$ntables" -eq 1 ]; then body="$body,${q}table${q}:$tables"; fi
if [ "$ntables" -gt 1 ]; then body="$body,${q}tables${q}:[$tables]"; fi
if [ "$nbuckets" -eq 1 ]; then body="$body,${q}bucket${q}:$buckets"; fi
if [ "$nbuckets" -gt 1 ]; then body="$body,${q}buckets${q}:[$buckets]"; fi
answer="{\"statusCode\":200,\"headers\":{\"Content-Type\":\"application/json\"},\"body\":\"$body}\"}"
while true; do
  headers="$(mktemp)"
  curl -sS -o /dev/null -D "$headers" "$api/next"
  id=$(grep -Fi Lambda-Runtime-Aws-Request-Id "$headers" | tr -d '[:space:]' | cut -d: -f2)
  rm -f "$headers"
  curl -sS -o /dev/null -X POST "$api/$id/response" -d "$answer"
done
`}}, true
	case strings.HasPrefix(runtime, "python"):
		m := sampleHandler.FindStringSubmatch(handler)
		if m == nil {
			return nil, false
		}
		return []sampleFile{{Name: m[1] + ".py", Content: `# Clutter's sample code. Upload your own code to replace it.
import json as _clutter_json
import os as _clutter_os
import re as _clutter_re


def ` + m[2] + `(event, context):
    # The answer names the tables and buckets this function's links hand it.
    body = {"message": ` + sampleMessageJSON + `}
    for kind, suffix in (("table", "` + sampleTableSuffix + `"), ("bucket", "` + sampleBucketSuffix + `")):
        names = sorted(v for k, v in _clutter_os.environ.items() if k.startswith("CLUTTER_") and k.endswith(suffix) and _clutter_re.fullmatch(r"[A-Za-z0-9._-]+", v))
        if len(names) == 1:
            body[kind] = names[0]
        elif names:
            body[kind + "s"] = names
    return {"statusCode": 200, "headers": {"Content-Type": "application/json"}, "body": _clutter_json.dumps(body)}
`}}, true
	case strings.HasPrefix(runtime, "nodejs"):
		m := sampleHandler.FindStringSubmatch(handler)
		if m == nil {
			return nil, false
		}
		return []sampleFile{{Name: m[1] + ".mjs", Content: `// Clutter's sample code. Upload your own code to replace it.
export const ` + m[2] + ` = async () => {
  // The answer names the tables and buckets this function's links hand it.
  const body = { message: ` + sampleMessageJSON + ` };
  for (const [kind, suffix] of [["table", "` + sampleTableSuffix + `"], ["bucket", "` + sampleBucketSuffix + `"]]) {
    const names = Object.entries(process.env)
      .filter(([k, v]) => k.startsWith("CLUTTER_") && k.endsWith(suffix) && /^[A-Za-z0-9._-]+$/.test(v ?? ""))
      .map(([, v]) => v)
      .sort();
    if (names.length === 1) body[kind] = names[0];
    else if (names.length > 1) body[kind + "s"] = names;
  }
  return { statusCode: 200, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) };
};
`}}, true
	}
	return nil, false
}

// functionSample is a function's code when nothing is uploaded for it: the
// sample, or the one file that says there is none. answers is true for a
// sample that answers requests.
func (m *model) functionSample(b *block) (files []sampleFile, answers bool) {
	if files, ok := sampleCode(m.runtime, m.handler(b)); ok {
		return files, true
	}
	return []sampleFile{{Name: "welcome.txt", Content: placeholderCode}}, false
}

// kindTemplates maps a block kind to its template folder under templates/aws.
// vm, bucket, function, table and http_api are A's templates, unchanged;
// network, firewall and volume exist for B only.
var kindTemplates = map[connectors.BlockKind]string{
	connectors.KindVM:       "ec2",
	connectors.KindNetwork:  "network",
	connectors.KindFirewall: "firewall",
	connectors.KindVolume:   "volume",
	connectors.KindBucket:   "s3",
	connectors.KindFunction: "lambda",
	connectors.KindTable:    "dynamodb",
	connectors.KindHTTPAPI:  "apigateway",
}

// renderOrder puts foundations first, so the rendered file reads top-down.
var renderOrder = []connectors.BlockKind{
	connectors.KindNetwork, connectors.KindFirewall, connectors.KindVM, connectors.KindVolume,
	connectors.KindBucket, connectors.KindTable, connectors.KindFunction, connectors.KindHTTPAPI,
}

// execute renders one template. missingkey=error makes a template that reads a
// variable B does not set fail here, not render "<no value>" into Terraform.
func execute(name, content string, vars any) (string, error) {
	t, err := template.New(name).Option("missingkey=error").Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return buf.String(), nil
}

func executeOwn(name string, vars any) (string, error) {
	content, err := ownTemplates.ReadFile("tmpl/" + name)
	if err != nil {
		return "", err
	}
	return execute(name, string(content), vars)
}

// boundaryEnv and userBoundaryEnv set the permissions_boundary_arn and
// user_permissions_boundary_arn variables provider.tf.tmpl declares. Every
// run's environment holds both, a destroy included.
const (
	boundaryEnv     = "TF_VAR_permissions_boundary_arn"
	userBoundaryEnv = "TF_VAR_user_permissions_boundary_arn"
)

// A role the Clutter CloudFormation stack did not create is refused
// (generic.ParseStackRoleARN, which A's account link uses too): the role's
// account and UniqueId also name the stack's permissions boundaries.
var errNotStackRole = &connectors.UserError{Message: "This role was not made by Clutter's CloudFormation stack. Link the account again from the AWS Account page."}

// boundaryARN is the permissions boundary the connection's stack created
// beside its role. The role may create a role only with it, so every role a
// run creates names it. A role the stack did not create has none.
func boundaryARN(roleARN string) (string, error) {
	account, unique, ok := generic.ParseStackRoleARN(roleARN)
	if !ok {
		return "", errNotStackRole
	}
	return "arn:aws:iam::" + account + ":policy/clutter/ClutterPermissionsBoundary-" + unique, nil
}

// userBoundaryARN is the narrower boundary the stack created for users: the
// role may create a user (a scoped key, cross.tf.tmpl) only with it.
func userBoundaryARN(roleARN string) (string, error) {
	account, unique, ok := generic.ParseStackRoleARN(roleARN)
	if !ok {
		return "", errNotStackRole
	}
	return "arn:aws:iam::" + account + ":policy/clutter/ClutterUserBoundary-" + unique, nil
}

// providerFile is the configuration every run needs, a destroy included.
func providerFile(d connectors.Deployment) ([]byte, error) {
	s, err := executeOwn("provider.tf.tmpl", map[string]any{
		"Region":       hclsafe.Escape(d.Connection.Region),
		"DeploymentID": hclsafe.Escape(d.ID),
	})
	return []byte(s), err
}

// render turns a model with no problems into the run's Terraform files.
func (m *model) render(ctx context.Context, src TemplateSource) (map[string][]byte, error) {
	if n := len(m.problems) + len(m.crossProblems); n > 0 {
		return nil, fmt.Errorf("the diagram has %d problems for this deployment", n)
	}
	provider, err := providerFile(m.d)
	if err != nil {
		return nil, err
	}
	var main, outputs strings.Builder
	main.Write(provider)
	for _, kind := range renderOrder {
		for _, id := range m.order {
			b := m.blocks[id]
			if b.Kind != kind {
				continue
			}
			vars, err := m.vars(b)
			if err != nil {
				return nil, err
			}
			for _, file := range []string{"main.tf.tmpl", "outputs.tf.tmpl"} {
				path := tftemplate.GetTemplatePath(kindTemplates[kind], file)
				t, err := src.Load(ctx, path)
				if err != nil {
					return nil, fmt.Errorf("load %s: %w", path, err)
				}
				data := vars
				if file == "outputs.tf.tmpl" {
					data = map[string]any{"ResourceName": b.Addr}
				}
				out, err := execute(path, t.Content, data)
				if err != nil {
					return nil, err
				}
				w := &main
				if file == "outputs.tf.tmpl" {
					w = &outputs
				}
				fmt.Fprintf(w, "\n# Block %s (%s)\n%s\n", b.Part, kind, strings.TrimRight(out, "\n"))
			}
		}
	}

	iam, err := executeOwn("iam.tf.tmpl", m.iamVars())
	if err != nil {
		return nil, err
	}
	ov, err := m.overrideVars()
	if err != nil {
		return nil, err
	}
	override, err := executeOwn("override.tf.tmpl", ov)
	if err != nil {
		return nil, err
	}
	dv, err := m.defaultsVars()
	if err != nil {
		return nil, err
	}
	defaults, err := executeOwn("defaults.tf.tmpl", dv)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{
		"main.tf":     []byte(main.String()),
		"outputs.tf":  []byte(strings.TrimLeft(outputs.String(), "\n")),
		"iam.tf":      []byte(iam),
		"override.tf": []byte(override),
		"defaults.tf": []byte(defaults),
	}
	// An override file cannot add a block, so the sample code's zips are in
	// their own primary file. A function with uploaded code needs none.
	var samples []sampleVars
	for _, id := range m.order {
		b := m.blocks[id]
		if b.Kind != connectors.KindFunction || b.Code != nil {
			continue
		}
		files, _ := m.functionSample(b)
		v := sampleVars{Addr: b.Addr, Zip: b.Addr + ".zip"}
		for _, f := range files {
			v.Files = append(v.Files, sampleFile{Name: hclsafe.Escape(f.Name), Content: hclsafe.Escape(f.Content)})
		}
		samples = append(samples, v)
	}
	if len(samples) > 0 {
		code, err := executeOwn("code.tf.tmpl", map[string]any{"Functions": samples})
		if err != nil {
			return nil, err
		}
		files["code.tf"] = []byte(code)
	}
	if m.cross != nil && len(m.cross.Exports) > 0 {
		cross, err := executeOwn("cross.tf.tmpl", m.crossVars())
		if err != nil {
			return nil, err
		}
		files["cross.tf"] = []byte(cross)
	}
	return files, nil
}

// scopedKeyVars is one permission edge to another cloud: an IAM user that
// can only read Target, and its one access key.
type scopedKeyVars struct {
	// Addr is the Terraform address of the user, its policy and its key; Name
	// is the user's name in IAM. Both are the deployment prefix and the edge
	// key, so a deployment's keys never collide with another's.
	Addr, Name, Key, Target string
	Bucket                  bool
}

// addressVars is one address edge to another cloud.
type addressVars struct{ Key, Target string }

func (m *model) crossVars() map[string]any {
	var keys []scopedKeyVars
	var addresses []addressVars
	for _, e := range m.cross.Exports {
		t := m.blocks[e.Target]
		if e.Class == connectors.EdgeAddress {
			addresses = append(addresses, addressVars{Key: e.Key, Target: t.Addr})
			continue
		}
		keys = append(keys, scopedKeyVars{
			Addr:   m.d.Prefix + "_" + e.Key,
			Name:   m.d.Prefix + "-" + e.Key,
			Key:    e.Key,
			Target: t.Addr,
			Bucket: t.Kind == connectors.KindBucket,
		})
	}
	return map[string]any{
		"Keys":      keys,
		"Addresses": addresses,
		"Region":    hclsafe.Escape(m.region),
	}
}

// vars is one block's variable map for its template. Every value is either
// built by B from checked parts (addresses, names, numbers, allow-listed
// values) or escaped for an HCL string.
func (m *model) vars(b *block) (map[string]any, error) {
	switch b.Kind {
	case connectors.KindVM:
		return map[string]any{
			"ResourceName": b.Addr,
			"InstanceType": m.vmInstanceType(b),
			// A key pair the block names is set in override.tf, as a
			// reference, so the instance waits for it.
			"KeyName": "",
		}, nil
	case connectors.KindNetwork:
		cidr, err := networkCIDR(b.str(fieldCIDR))
		if err != nil {
			return nil, err
		}
		private, public := networkHalves(cidr)
		hasPublic, _ := m.networkVMs(b.ID)
		nat := m.networkNAT(b.ID)
		return map[string]any{
			"ResourceName": b.Addr,
			"NetworkName":  m.physical(b, maxName),
			"CIDR":         cidr.String(),
			"PrivateCIDR":  private.String(),
			"PublicCIDR":   public.String(),
			// The NAT gateway reaches the internet through the internet gateway.
			"Public": hasPublic || nat,
			"NAT":    nat,
		}, nil
	case connectors.KindFirewall:
		rules, err := firewallRules(b.str(fieldIngress), b.str(fieldSourceCIDR))
		if err != nil {
			return nil, err
		}
		net, err := m.firewallNetwork(b.ID)
		if err != nil {
			return nil, err
		}
		vpc := ""
		if net != "" {
			vpc = m.blocks[net].Addr
		}
		return map[string]any{
			"ResourceName":    b.Addr,
			"GroupName":       m.physical(b, maxName),
			"VPCResourceName": vpc,
			"Ingress":         rules,
		}, nil
	case connectors.KindVolume:
		size, _ := b.integer(fieldSizeGB)
		return map[string]any{
			"ResourceName":         b.Addr,
			"VolumeName":           m.physical(b, maxName),
			"InstanceResourceName": m.blocks[m.volumeVM[b.ID]].Addr,
			"SizeGB":               size,
			"VolumeType":           "gp3",
			"DeviceName":           m.deviceName(b.ID),
		}, nil
	case connectors.KindBucket:
		return map[string]any{
			"ResourceName":      b.Addr,
			"BucketName":        m.bucketName(b),
			"EnableVersioning":  b.boolean(fieldVersioning),
			"BlockPublicAccess": true,
		}, nil
	case connectors.KindTable:
		return map[string]any{
			"ResourceName": b.Addr,
			"TableName":    m.physical(b, maxName),
			"BillingMode":  "PAY_PER_REQUEST",
			"HashKey":      b.str(fieldKey),
			"HashKeyType":  keyTypes[b.str(fieldKeyType)],
			"EnableTTL":    false,
			"TTLAttribute": "ttl",
		}, nil
	case connectors.KindFunction:
		memory, _ := b.integer(fieldMemoryMB)
		timeout, _ := b.integer(fieldTimeoutS)
		return map[string]any{
			"ResourceName": b.Addr,
			// Also names the function's role: "<FunctionName>-lambda-role".
			"FunctionName":         m.physical(b, maxFunctionName),
			"Handler":              hclsafe.Escape(m.handler(b)),
			"Timeout":              timeout,
			"Runtime":              m.runtime,
			"Architecture":         m.architecture,
			"MemorySize":           memory,
			"Filename":             b.Addr + ".zip",
			"EnvironmentVariables": m.functionEnv(b.ID),
		}, nil
	case connectors.KindHTTPAPI:
		return map[string]any{
			"ResourceName": b.Addr,
			"APIName":      m.physical(b, maxName),
			"Description":  hclsafe.Escape(b.str(fieldDescription)),
			"StageName":    apiStage,
			// A's template answers CORS only at the API's root, where B has no
			// route; iam.tf answers it on each route instead.
			"EnableCORS": false,
			// AWS deploys an API only when it has a method; B's every API has a
			// route (CodeAPIWithoutRoute).
			"HasMethods": len(m.apiRoutes[b.ID]) > 0,
		}, nil
	}
	return nil, fmt.Errorf("no AWS template for %q blocks", b.Kind)
}

// readVars is one "reads" edge: Role is the reading role's Terraform address
// (a function's role, or a VM's instance role).
type readVars struct{ Addr, Name, Role, Target string }

type routeVars struct {
	Addr, API, Function, PathPart, StatementID string
	// CORS is the API's cors field: the route answers a browser's preflight.
	CORS bool
}

// peerVars is the security group that lets the VMs that get Target's address
// reach it. VPC is the network's Terraform address, or "" for the account's
// default VPC.
type peerVars struct {
	Addr, Name, VPC, Target string
	Sources                 []peerSource
}

type peerSource struct{ Addr, Part string }

// peerGroup is the Terraform address of a VM's peer security group.
func peerGroup(vm *block) string { return vm.Addr + "_peers" }

func (m *model) iamVars() map[string]any {
	var functions []string
	var bucketReads, tableReads []readVars
	var routes []routeVars
	var peers []peerVars
	for _, id := range m.order {
		if b := m.blocks[id]; b.Kind == connectors.KindFunction {
			functions = append(functions, b.Addr)
		}
	}
	add := func(reader *block, role string, target *block) {
		v := readVars{
			Addr:   reader.Addr + "_reads_" + target.Part,
			Name:   m.d.Prefix + "-read-" + target.Part,
			Role:   role,
			Target: target.Addr,
		}
		if target.Kind == connectors.KindBucket {
			bucketReads = append(bucketReads, v)
		} else {
			tableReads = append(tableReads, v)
		}
	}
	for _, r := range m.reads {
		fn := m.blocks[r[0]]
		add(fn, "aws_iam_role."+fn.Addr+"_role", m.blocks[r[1]])
	}
	// A VM reads through its instance role, which A's ec2 template creates.
	for _, r := range m.vmReads {
		vm := m.blocks[r[0]]
		add(vm, "aws_iam_role."+vm.Addr+"_ssm_role", m.blocks[r[1]])
	}
	for _, id := range m.order {
		api := m.blocks[id]
		for _, fnID := range m.apiRoutes[id] {
			fn := m.blocks[fnID]
			routes = append(routes, routeVars{
				Addr:     api.Addr + "_to_" + fn.Part,
				API:      api.Addr,
				Function: fn.Addr,
				// A block name is lowercase letters, digits, hyphens and
				// underscores, which a path part takes as is.
				PathPart: fn.Name,
				// Unique per API, so two APIs on one function do not collide.
				StatementID: "AllowInvokeFrom-" + api.Addr,
				CORS:        api.boolean(fieldCORS),
			})
		}
	}
	for _, id := range m.order {
		vm := m.blocks[id]
		sources := m.peersOf(id)
		if vm.Kind != connectors.KindVM || len(sources) == 0 {
			continue
		}
		p := peerVars{
			Addr: peerGroup(vm),
			// A block name holds no dot, so this name is never another block's.
			Name:   m.physical(vm, maxName-len(".peers")) + ".peers",
			Target: vm.Addr,
		}
		if net, ok := m.vmNetwork[id]; ok {
			p.VPC = m.blocks[net].Addr
		}
		for _, s := range sources {
			p.Sources = append(p.Sources, peerSource{Addr: m.blocks[s].Addr, Part: m.blocks[s].Part})
		}
		peers = append(peers, p)
	}
	return map[string]any{
		"Functions":   functions,
		"BucketReads": bucketReads,
		"TableReads":  tableReads,
		"Routes":      routes,
		"Peers":       peers,
	}
}

// Stage throttling for every HTTP API: 100 requests a second on average and
// bursts of 200, well under the account's regional limit (10,000 and 5,000),
// so one API cannot use it all, and a loop or a flood of calls costs at most
// that much.
const (
	apiRateLimit  = 100
	apiBurstLimit = 200
	apiStage      = "v1"
)

type logGroupVars struct{ Addr, Name string }

type keyPairVars struct{ Addr, Name, Key string }

// defaultsVars feeds defaults.tf: each function's log group, each bucket's
// TLS-only policy, each API's throttling and each VM's key pair.
func (m *model) defaultsVars() (map[string]any, error) {
	var functions []logGroupVars
	var buckets, apis []string
	var keys []keyPairVars
	for _, id := range m.order {
		b := m.blocks[id]
		switch b.Kind {
		case connectors.KindFunction:
			functions = append(functions, logGroupVars{Addr: b.Addr, Name: m.physical(b, maxFunctionName)})
		case connectors.KindBucket:
			buckets = append(buckets, b.Addr)
		case connectors.KindHTTPAPI:
			apis = append(apis, b.Addr)
		case connectors.KindVM:
			key, err := connectors.SSHPublicKey(b.fields)
			if err != nil {
				return nil, err
			}
			if key != "" {
				keys = append(keys, keyPairVars{Addr: b.Addr, Name: m.physical(b, maxName), Key: hclsafe.Escape(key)})
			}
		}
	}
	return map[string]any{
		"Functions":  functions,
		"Buckets":    buckets,
		"APIs":       apis,
		"KeyPairs":   keys,
		"RateLimit":  apiRateLimit,
		"BurstLimit": apiBurstLimit,
	}, nil
}

type instanceVars struct {
	// Subnet is the subnet's whole Terraform address after "aws_subnet.".
	Addr, Subnet   string
	SecurityGroups []string
	Public         bool
	// KeyPair is the address of the VM's aws_key_pair, or "" for none.
	KeyPair string
	// Name is the instance's Name tag: the block's name, as an HCL string.
	Name string
	// UserData is the user_data_base64 argument (model.userData): an HCL
	// string of base64, or an expression that builds the startup wrapper, or
	// "" for none.
	UserData string
	// Triggers are the references UserData reads. When one of them is
	// unknown at plan time (its resource is being replaced), the provider
	// plans an update and only finds the replacement during apply, which
	// Terraform refuses as an inconsistent final plan. Listing them in
	// replace_triggered_by puts that replacement in the plan.
	Triggers []string
}

// functionVars is one function's code: the zip code.tf builds from its sample
// code, or, when Hash is set, the uploaded zip the Terraform runner stages as
// Zip.
type functionVars struct{ Addr, Zip, Hash string }

// sampleVars is one function's sample code as code.tf zips it: each file's
// name and content already escaped for an HCL string.
type sampleVars struct {
	Addr, Zip string
	Files     []sampleFile
}

// handler is a function's entry point: its own field, or the runtime's
// default when the field is empty.
func (m *model) handler(b *block) string {
	if h := b.str(connectors.FieldHandler); h != "" {
		return h
	}
	return generic.LambdaRuntimes[m.runtime]
}

// archives are the uploaded zips a run stages, one per function that has one.
func (m *model) archives() []terraform.Archive {
	var out []terraform.Archive
	for _, id := range m.order {
		b := m.blocks[id]
		if b.Kind == connectors.KindFunction && b.Code != nil {
			out = append(out, terraform.Archive{ID: b.Addr, Key: b.Code.Key, SHA256: b.Code.SHA256})
		}
	}
	return out
}

type apiVars struct {
	Addr   string
	Routes []string
	CORS   bool
}

type bucketVars struct {
	Addr         string
	ForceDestroy bool
}

func (m *model) overrideVars() (map[string]any, error) {
	var instances []instanceVars
	var functions []functionVars
	var apis []apiVars
	var buckets []bucketVars
	var tables []string
	for _, id := range m.order {
		b := m.blocks[id]
		switch b.Kind {
		case connectors.KindVM:
			v := instanceVars{
				Addr:   b.Addr,
				Public: b.boolean(fieldPublic),
				Subnet: m.vmSubnet(id),
				Name:   hclsafe.Escape(m.physical(b, maxName)),
			}
			for _, fw := range m.vmFirewalls[id] {
				v.SecurityGroups = append(v.SecurityGroups, m.blocks[fw].Addr)
			}
			if len(m.peersOf(id)) > 0 {
				v.SecurityGroups = append(v.SecurityGroups, peerGroup(b))
			}
			key, err := connectors.SSHPublicKey(b.fields)
			if err != nil {
				return nil, err
			}
			if key != "" {
				v.KeyPair = b.Addr
			}
			// The script reaches HCL only as base64; values reach it only
			// checked and escaped (connectors.StartupHCL).
			ud, err := m.userData(b)
			if err != nil {
				return nil, err
			}
			v.UserData = ud
			env, err := m.vmEnv(id)
			if err != nil {
				return nil, err
			}
			for _, e := range env {
				if e.Ref != "" {
					v.Triggers = append(v.Triggers, e.Ref)
				}
			}
			instances = append(instances, v)
		case connectors.KindFunction:
			f := functionVars{Addr: b.Addr, Zip: b.Addr + ".zip"}
			if b.Code != nil {
				hash, err := b.Code.Base64SHA256()
				if err != nil {
					return nil, err
				}
				f.Zip, f.Hash = terraform.ArchiveName(b.Addr), hash
			}
			functions = append(functions, f)
		case connectors.KindHTTPAPI:
			v := apiVars{Addr: b.Addr, CORS: b.boolean(fieldCORS)}
			for _, fnID := range m.apiRoutes[id] {
				v.Routes = append(v.Routes, b.Addr+"_to_"+m.blocks[fnID].Part)
			}
			apis = append(apis, v)
		case connectors.KindBucket:
			buckets = append(buckets, bucketVars{Addr: b.Addr, ForceDestroy: b.boolean(fieldForceDestroy)})
		case connectors.KindTable:
			tables = append(tables, b.Addr)
		}
	}
	return map[string]any{
		"Instances": instances,
		"Functions": functions,
		"APIs":      apis,
		"Buckets":   buckets,
		"Tables":    tables,
	}, nil
}

// resultOutput is one output of results.tf: a connectors.ResultOutputName and
// a Terraform expression built from checked parts.
type resultOutput struct{ Name, Value string }

// resultOutputs are what an apply tells the user about each block
// (connectors.ResultNames): a VM's private address and, when it is public, its
// public one, and with an endpoint that it is simulated; a bucket's and a
// table's name; an HTTP API's address, on its
// first route, which answers. endpoint is the connector's endpoint (LocalStack
// on a laptop, "" on AWS): there the API's address is the endpoint's own form,
// since AWS's execute-api name does not reach LocalStack.
func (m *model) resultOutputs(endpoint string) []resultOutput {
	var out []resultOutput
	add := func(id, name, value string) {
		if key, ok := connectors.ResultOutputName(id, name); ok {
			out = append(out, resultOutput{Name: key, Value: value})
		}
	}
	for _, id := range m.order {
		b := m.blocks[id]
		switch b.Kind {
		case connectors.KindVM:
			add(id, connectors.ResultPrivateAddress, "aws_instance."+b.Addr+".private_ip")
			if b.boolean(fieldPublic) {
				add(id, connectors.ResultPublicAddress, "aws_instance."+b.Addr+".public_ip")
			}
			// LocalStack records an instance but never boots it.
			if endpoint != "" {
				add(id, connectors.ResultSimulated, `"`+connectors.ResultSimulatedEmulator+`"`)
			}
		case connectors.KindFunction:
			if b.Code == nil {
				if _, answers := m.functionSample(b); answers {
					add(id, connectors.ResultCode, `"`+connectors.ResultCodeSample+`"`)
				}
			}
		case connectors.KindBucket:
			add(id, connectors.ResultBucket, "aws_s3_bucket."+b.Addr+".bucket")
		case connectors.KindTable:
			add(id, connectors.ResultTable, "aws_dynamodb_table."+b.Addr+".name")
		case connectors.KindHTTPAPI:
			routes := m.apiRoutes[id]
			if len(routes) == 0 {
				continue // no route, so no stage and no address (CodeAPIWithoutRoute)
			}
			path := hclsafe.Escape(m.blocks[routes[0]].Name)
			if endpoint != "" {
				add(id, connectors.ResultURL, `"`+hclsafe.Escape(endpoint)+"/restapis/${aws_api_gateway_rest_api."+b.Addr+".id}/"+apiStage+"/_user_request_/"+path+`"`)
			} else {
				add(id, connectors.ResultURL, `"${aws_api_gateway_stage.`+b.Addr+"_stage[0].invoke_url}/"+path+`"`)
			}
		}
	}
	return out
}

// primaryAddresses maps each block's main Terraform resource to its node id,
// so an apply can report what it built.
func (m *model) primaryAddresses() map[string]string {
	types := map[connectors.BlockKind]string{
		connectors.KindVM:       "aws_instance",
		connectors.KindNetwork:  "aws_vpc",
		connectors.KindFirewall: "aws_security_group",
		connectors.KindVolume:   "aws_ebs_volume",
		connectors.KindBucket:   "aws_s3_bucket",
		connectors.KindFunction: "aws_lambda_function",
		connectors.KindTable:    "aws_dynamodb_table",
		connectors.KindHTTPAPI:  "aws_api_gateway_rest_api",
	}
	out := map[string]string{}
	for _, id := range m.order {
		b := m.blocks[id]
		out[types[b.Kind]+"."+b.Addr] = id
	}
	return out
}
