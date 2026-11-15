package aws

import (
	"bytes"
	"context"
	"embed"
	"fmt"
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

// placeholderCode is Clutter's placeholder code, which a function gets until
// code is uploaded for it. It is the same text as
// backend/infra/modules/s3/uploads/bootstrap.zip.
const placeholderCode = "Welcome to your Lambda! You can upload your own code and take it from there."

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
	files := map[string][]byte{
		"main.tf":     []byte(main.String()),
		"outputs.tf":  []byte(strings.TrimLeft(outputs.String(), "\n")),
		"iam.tf":      []byte(iam),
		"override.tf": []byte(override),
	}
	// An override file cannot add a block, so the placeholder zips are in
	// their own primary file. A function with uploaded code needs none.
	var placeholders []functionVars
	fns, _ := ov["Functions"].([]functionVars)
	for _, f := range fns {
		if f.Hash == "" {
			placeholders = append(placeholders, f)
		}
	}
	if len(placeholders) > 0 {
		code, err := executeOwn("code.tf.tmpl", map[string]any{
			"Functions":       placeholders,
			"PlaceholderText": hclsafe.Escape(placeholderCode),
		})
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
			// B links no key pair: access is through SSM, as in A.
			"KeyName": "",
		}, nil
	case connectors.KindNetwork:
		cidr, err := networkCIDR(b.str(fieldCIDR))
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"ResourceName": b.Addr,
			"NetworkName":  m.physical(b, maxName),
			"CIDR":         cidr.String(),
			"Public":       m.networkPublic(b.ID),
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
			"EnvironmentVariables": map[string]string{},
		}, nil
	case connectors.KindHTTPAPI:
		return map[string]any{
			"ResourceName": b.Addr,
			"APIName":      m.physical(b, maxName),
			"Description":  hclsafe.Escape(b.str(fieldDescription)),
			"StageName":    "v1",
			"EnableCORS":   b.boolean(fieldCORS),
			// AWS deploys an API only when it has a method: the CORS answer or a route.
			"HasMethods": b.boolean(fieldCORS) || len(m.apiRoutes[b.ID]) > 0,
			// Only the CORS answer reads this; each route takes every method.
			"HTTPMethods": "GET,POST,PUT,PATCH,DELETE",
		}, nil
	}
	return nil, fmt.Errorf("no AWS template for %q blocks", b.Kind)
}

// readVars is one "reads" edge: Role is the reading role's Terraform address
// (a function's role, or a VM's instance role).
type readVars struct{ Addr, Name, Role, Target string }

type routeVars struct{ Addr, API, Function, PathPart, StatementID string }

func (m *model) iamVars() map[string]any {
	var functions []string
	var bucketReads, tableReads []readVars
	var routes []routeVars
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
			})
		}
	}
	return map[string]any{
		"Functions":   functions,
		"BucketReads": bucketReads,
		"TableReads":  tableReads,
		"Routes":      routes,
	}
}

type instanceVars struct {
	Addr, Subnet   string
	SecurityGroups []string
	Public         bool
	// UserData is the user_data_base64 argument (model.userData): an HCL
	// string of base64, or an expression that builds the startup wrapper, or
	// "" for none.
	UserData string
}

// functionVars is one function's code: the zip code.tf builds from the
// placeholder text, or, when Hash is set, the uploaded zip the Terraform
// runner stages as Zip.
type functionVars struct{ Addr, Zip, Hash string }

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
}

func (m *model) overrideVars() (map[string]any, error) {
	var instances []instanceVars
	var functions []functionVars
	var apis []apiVars
	for _, id := range m.order {
		b := m.blocks[id]
		switch b.Kind {
		case connectors.KindVM:
			v := instanceVars{Addr: b.Addr, Public: b.boolean(fieldPublic)}
			if net, ok := m.vmNetwork[id]; ok {
				v.Subnet = m.blocks[net].Addr
			}
			for _, fw := range m.vmFirewalls[id] {
				v.SecurityGroups = append(v.SecurityGroups, m.blocks[fw].Addr)
			}
			// The script reaches HCL only as base64; values reach it only
			// checked and escaped (connectors.StartupHCL).
			ud, err := m.userData(b)
			if err != nil {
				return nil, err
			}
			v.UserData = ud
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
			v := apiVars{Addr: b.Addr}
			for _, fnID := range m.apiRoutes[id] {
				v.Routes = append(v.Routes, b.Addr+"_to_"+m.blocks[fnID].Part)
			}
			apis = append(apis, v)
		}
	}
	return map[string]any{
		"Instances": instances,
		"Functions": functions,
		"APIs":      apis,
	}, nil
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
