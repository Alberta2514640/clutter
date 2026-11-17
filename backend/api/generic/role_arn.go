package generic

import "regexp"

// stackRoleARN is the role Clutter's CloudFormation stack creates in a
// customer's account (backend/infra/modules/s3/uploads/client_side_terraform_deployer_role.yaml):
// RoleName AllowClutterToDeployTerraformRole-${UniqueId}, with no path, and a
// UniqueId of 8 lowercase letters and digits (RandomID(8) in
// cloudformation/stack-url-generator). This is the one owner of the shape. A
// copies it where Go cannot be read: the API Gateway model
// organization-submit-account-role-arn.json, the runner's entrypoint.sh, and
// the website's awsRoleArnPattern (frontend organization/rules.ts).
var stackRoleARN = regexp.MustCompile(`^arn:aws:iam::([0-9]{12}):role/AllowClutterToDeployTerraformRole-([a-z0-9]{8})$`)

// StackRoleARNShape describes the accepted ARN for an error message.
const StackRoleARNShape = "arn:aws:iam::<12-digit account id>:role/AllowClutterToDeployTerraformRole-<8 lowercase letters or digits>"

// ParseStackRoleARN returns the account id and UniqueId of a role Clutter's
// stack created. ok is false for any other ARN, including one with a path,
// another partition or trailing text.
func ParseStackRoleARN(arn string) (accountID, uniqueID string, ok bool) {
	m := stackRoleARN.FindStringSubmatch(arn)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}
