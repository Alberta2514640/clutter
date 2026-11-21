package generic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStackRoleARN(t *testing.T) {
	account, unique, ok := ParseStackRoleARN("arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd34")
	if !ok || account != "123456789012" || unique != "ab12cd34" {
		t.Fatalf("ParseStackRoleARN(good) = %q, %q, %v", account, unique, ok)
	}
	for _, arn := range []string{
		"",
		"arn:aws:iam::123456789012:role/SomeOtherRole-ab12cd34",
		"arn:aws:iam::123456789012:role/ClutterDeployRole",
		"arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-abc",
		"arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-AB12CD34",
		"arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd345",
		"arn:aws:iam::123456789012:role/path/AllowClutterToDeployTerraformRole-ab12cd34",
		"arn:aws-us-gov:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd34",
		"arn:aws:iam::12345678901:role/AllowClutterToDeployTerraformRole-ab12cd34",
		"arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd34\n",
		" arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd34",
	} {
		if _, _, ok := ParseStackRoleARN(arn); ok {
			t.Errorf("ParseStackRoleARN(%q) accepted an ARN the stack does not create", arn)
		}
	}
}

// The role name and UniqueId length come from the stack template and the
// stack URL handler; this fails if either changes without the validator.
func TestStackRoleARNMatchesTheStack(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "infra", "modules", "s3", "uploads", "client_side_terraform_deployer_role.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `RoleName: !Sub "AllowClutterToDeployTerraformRole-${UniqueId}"`) {
		t.Error("the stack no longer names its role AllowClutterToDeployTerraformRole-${UniqueId}")
	}
	handler, err := os.ReadFile(filepath.Join("..", "cloudformation", "stack-url-generator", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(handler), "uniqueId := generic.RandomID(8)") {
		t.Error("the stack URL handler no longer makes an 8-character UniqueId")
	}
	if _, _, ok := ParseStackRoleARN("arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-" + RandomID(8)); !ok {
		t.Error("ParseStackRoleARN refused a UniqueId RandomID(8) makes")
	}
}
