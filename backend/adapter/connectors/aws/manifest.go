package aws

import (
	"sort"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// Block field names. AWS offers every standard block as-is, so each field is
// a portable one connectors owns; StandardField panics on a name it does not
// have, so a typo fails at start-up and in every test.
var (
	fieldSize        = connectors.StandardField(connectors.KindVM, "size").Name
	fieldPublic      = connectors.StandardField(connectors.KindVM, "public").Name
	fieldCIDR        = connectors.StandardField(connectors.KindNetwork, "cidr").Name
	fieldIngress     = connectors.StandardField(connectors.KindFirewall, "ingress").Name
	fieldSourceCIDR  = connectors.StandardField(connectors.KindFirewall, "source_cidr").Name
	fieldSizeGB      = connectors.StandardField(connectors.KindVolume, "size_gb").Name
	fieldVersioning  = connectors.StandardField(connectors.KindBucket, "versioning").Name
	fieldMemoryMB    = connectors.StandardField(connectors.KindFunction, "memory_mb").Name
	fieldTimeoutS    = connectors.StandardField(connectors.KindFunction, "timeout_s").Name
	fieldKey         = connectors.StandardField(connectors.KindTable, "key").Name
	fieldKeyType     = connectors.StandardField(connectors.KindTable, "key_type").Name
	fieldDescription = connectors.StandardField(connectors.KindHTTPAPI, "description").Name
	fieldCORS        = connectors.StandardField(connectors.KindHTTPAPI, "cors").Name
)

// Deployment setting names: what is AWS's own, so it lives on the deployment.
const (
	settingInstanceType = "instance_type"
	settingRuntime      = "function_runtime"
	settingArchitecture = "function_architecture"
)

// sizeTypes maps the portable VM size to an instance type. Every type is
// x86_64, because A's EC2 template pins an x86_64 Amazon Linux image.
var sizeTypes = map[string]string{
	"small":  "t3.micro",
	"medium": "t3.small",
	"large":  "t3.medium",
}

// instanceTypes are the instance types a deployment may choose instead of the
// size mapping. x86_64 only, for the same reason as sizeTypes.
var instanceTypes = []string{"t3.micro", "t3.small", "t3.medium", "t3.large", "m5.large", "m5.xlarge", "c5.large"}

// keyTypes maps the portable table key type to DynamoDB's attribute type.
var keyTypes = map[string]string{"string": "S", "number": "N", "binary": "B"}

// regions returns the AWS regions A supports (generic owns the list), sorted.
func regions() []string {
	out := make([]string, 0, len(generic.SupportedRegions))
	for r := range generic.SupportedRegions {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// runtimes returns the Lambda runtimes A supports (generic owns the list), sorted.
func runtimes() []string {
	out := generic.LambdaRuntimeNames()
	sort.Strings(out)
	return out
}

func manifest() connectors.Manifest {
	return connectors.Manifest{
		Connector: ID,
		Title:     "Amazon Web Services",
		Version:   "1",
		Regions:   regions(),
		// The logo is a neutral tile with the name: AWS's trademark guidelines
		// license no logo for this use (frontend/clutter/public/clouds/README.md).
		Display: connectors.Display{ShortName: "AWS", Logo: "logo"},
		// Icons are AWS Architecture Icons. AWS publishes no security group
		// icon, so the firewall keeps its neutral icon.
		Blocks: []connectors.BlockSpec{
			connectors.StandardBlock(connectors.KindVM, "VM").WithDisplay("EC2", "ec2"),
			connectors.StandardBlock(connectors.KindNetwork, "Network").WithDisplay("VPC", "vpc"),
			connectors.StandardBlock(connectors.KindFirewall, "Firewall").WithDisplay("Security group", ""),
			connectors.StandardBlock(connectors.KindVolume, "Volume").WithDisplay("EBS", "ebs"),
			connectors.StandardBlock(connectors.KindBucket, "Bucket").WithDisplay("S3", "s3"),
			connectors.StandardBlock(connectors.KindFunction, "Function").WithDisplay("Lambda", "lambda"),
			connectors.StandardBlock(connectors.KindTable, "Table").WithDisplay("DynamoDB", "dynamodb"),
			connectors.StandardBlock(connectors.KindHTTPAPI, "HTTP API").WithDisplay("API Gateway", "api-gateway"),
		},
		Edges: []connectors.EdgeSpec{
			{Source: connectors.KindVM, Target: connectors.KindNetwork, Meaning: "the VM runs in the network"},
			{Source: connectors.KindVolume, Target: connectors.KindVM, Meaning: "the volume is attached to the VM"},
			{Source: connectors.KindFirewall, Target: connectors.KindVM, Meaning: "the firewall rules apply to the VM"},
			{Source: connectors.KindFunction, Target: connectors.KindBucket, Meaning: "the function can read the bucket"},
			{Source: connectors.KindFunction, Target: connectors.KindTable, Meaning: "the function can read the table"},
			{Source: connectors.KindHTTPAPI, Target: connectors.KindFunction, Meaning: "the HTTP API sends requests to the function"},
			// A VM's edges hand it values at first boot (connectors.StartupHCL);
			// across clouds they are a split deployment's address and
			// permission edges (connectors.CrossClass).
			{Source: connectors.KindVM, Target: connectors.KindVM, Meaning: "the VM gets the other VM's address when it first starts"},
			{Source: connectors.KindVM, Target: connectors.KindBucket, Meaning: "the VM can read the bucket"},
			{Source: connectors.KindVM, Target: connectors.KindTable, Meaning: "the VM can read the table"},
		},
		Settings: []connectors.FieldSpec{
			{Name: settingInstanceType, Title: "Instance type for every VM (instead of its size)", Type: connectors.FieldEnum, Allowed: instanceTypes},
			{Name: settingRuntime, Title: "Function runtime", Type: connectors.FieldEnum, Default: "provided.al2023", Allowed: runtimes()},
			{Name: settingArchitecture, Title: "Function processor", Type: connectors.FieldEnum, Default: "arm64", Allowed: []string{"arm64", "x86_64"}},
		},
	}
}
