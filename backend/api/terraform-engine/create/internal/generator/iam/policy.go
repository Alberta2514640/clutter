package iam

import (
	"fmt"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"
)

type IAMPolicyGenerator struct{}

func NewIAMPolicyGenerator() *IAMPolicyGenerator {
	return &IAMPolicyGenerator{}
}

// GenerateFromRelationships creates IAM policies based on edges
func (g *IAMPolicyGenerator) GenerateFromRelationships(relationships []*internal.IAMRelationship) (string, error) {
	if len(relationships) == 0 {
		return "", nil
	}

	var sb strings.Builder

	for _, rel := range relationships {
		policy, err := g.generatePolicy(rel)
		if err != nil {
			continue // Skip unsupported relationships
		}
		sb.WriteString(policy)
	}

	return sb.String(), nil
}

func (g *IAMPolicyGenerator) generatePolicy(rel *internal.IAMRelationship) (string, error) {
	// API Gateway → Lambda uses aws_lambda_permission, not an IAM policy
	if rel.SourceResource.Type == internal.ResourceTypeAPIGateway && rel.TargetResource.Type == internal.ResourceTypeLambda {
		return g.generateLambdaPermission(rel), nil
	}

	// Only a Lambda has a role (aws_iam_role.<name>_role, from the lambda
	// template) for the edge to grant to.
	if rel.SourceResource.Type != internal.ResourceTypeLambda {
		return "", fmt.Errorf("no IAM actions defined for %s -> %s", rel.SourceResource.Type, rel.TargetResource.Type)
	}
	statements := g.readStatements(rel.TargetResource)
	if statements == "" {
		return "", fmt.Errorf("no IAM actions defined for %s -> %s", rel.SourceResource.Type, rel.TargetResource.Type)
	}

	policyName := fmt.Sprintf("%s_to_%s", rel.SourceResource.Name, rel.TargetResource.Name)

	// An inline policy on the function's own role. The customer role may not
	// create managed policies (client_side_terraform_deployer_role.yaml), and
	// an inline policy is deleted with its role.
	return fmt.Sprintf(`resource "aws_iam_role_policy" "%s" {
  name = "%s-access"
  role = aws_iam_role.%s_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
%s    ]
  })
}

`, policyName, policyName, rel.SourceResource.Name, statements), nil
}

func (g *IAMPolicyGenerator) generateLambdaPermission(rel *internal.IAMRelationship) string {
	permName := fmt.Sprintf("%s_invoke_%s", rel.SourceResource.Name, rel.TargetResource.Name)
	return fmt.Sprintf(`resource "aws_lambda_permission" "%s" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.%s.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.%s.execution_arn}/*/*"
}

`, permName, rel.TargetResource.Name, rel.SourceResource.Name)
}

// readStatements returns the policy statements for a Lambda that reads the
// target, or "" when an edge to that target grants nothing. An edge means
// "reads": it grants no write action. The lists match B's AWS connector
// (backend/adapter/connectors/aws/tmpl/iam.tf.tmpl).
func (g *IAMPolicyGenerator) readStatements(target *internal.TerraformResource) string {
	switch target.Type {
	case internal.ResourceTypeDynamoDB:
		return fmt.Sprintf(`      {
        Effect = "Allow"
        Action = [
          "dynamodb:BatchGetItem",
          "dynamodb:ConditionCheckItem",
          "dynamodb:DescribeTable",
          "dynamodb:GetItem",
          "dynamodb:Query",
          "dynamodb:Scan",
        ]
        Resource = [
          aws_dynamodb_table.%s.arn,
          "${aws_dynamodb_table.%s.arn}/index/*",
        ]
      },
`, target.Name, target.Name)
	case internal.ResourceTypeS3:
		return fmt.Sprintf(`      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = aws_s3_bucket.%s.arn
      },
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject"]
        Resource = "${aws_s3_bucket.%s.arn}/*"
      },
`, target.Name, target.Name)
	default:
		return ""
	}
}
