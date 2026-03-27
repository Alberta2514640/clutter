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

	actions := g.getActionsForRelationship(rel.SourceResource.Type, rel.TargetResource.Type)
	if len(actions) == 0 {
		return "", fmt.Errorf("no IAM actions defined for %s -> %s", rel.SourceResource.Type, rel.TargetResource.Type)
	}

	resourceArn := g.getResourceArn(rel.TargetResource)
	policyName := fmt.Sprintf("%s_to_%s", rel.SourceResource.Name, rel.TargetResource.Name)

	var sb strings.Builder

	// IAM Policy
	sb.WriteString(fmt.Sprintf(`resource "aws_iam_policy" "%s" {
  name = "%s-access"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
`, policyName, policyName))

	for i, action := range actions {
		if i < len(actions)-1 {
			sb.WriteString(fmt.Sprintf("        \"%s\",\n", action))
		} else {
			sb.WriteString(fmt.Sprintf("        \"%s\"\n", action))
		}
	}

	sb.WriteString(fmt.Sprintf(`      ]
      Resource = %s
    }]
  })
}

`, resourceArn))

	// Policy attachment to source role
	if rel.SourceResource.Type == internal.ResourceTypeLambda {
		sb.WriteString(fmt.Sprintf(`resource "aws_iam_role_policy_attachment" "%s" {
  role       = aws_iam_role.%s_role.name
  policy_arn = aws_iam_policy.%s.arn
}

`, policyName, rel.SourceResource.Name, policyName))
	}

	return sb.String(), nil
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

// getActionsForRelationship determines IAM actions based on source->target types
func (g *IAMPolicyGenerator) getActionsForRelationship(source, target internal.ResourceType) []string {
	switch {
	case source == internal.ResourceTypeLambda && target == internal.ResourceTypeDynamoDB:
		return []string{
			"dynamodb:GetItem",
			"dynamodb:PutItem",
			"dynamodb:UpdateItem",
			"dynamodb:DeleteItem",
			"dynamodb:Query",
			"dynamodb:Scan",
		}
	case source == internal.ResourceTypeLambda && target == internal.ResourceTypeS3:
		return []string{
			"s3:GetObject",
			"s3:PutObject",
			"s3:DeleteObject",
			"s3:ListBucket",
		}
	case source == internal.ResourceTypeAPIGateway && target == internal.ResourceTypeLambda:
		return []string{
			"lambda:InvokeFunction",
		}
	default:
		return nil
	}
}

// getResourceArn returns the terraform reference for the target resource ARN
func (g *IAMPolicyGenerator) getResourceArn(resource *internal.TerraformResource) string {
	switch resource.Type {
	case internal.ResourceTypeDynamoDB:
		return fmt.Sprintf("aws_dynamodb_table.%s.arn", resource.Name)
	case internal.ResourceTypeS3:
		return fmt.Sprintf("[aws_s3_bucket.%s.arn, \"${aws_s3_bucket.%s.arn}/*\"]", resource.Name, resource.Name)
	case internal.ResourceTypeLambda:
		return fmt.Sprintf("aws_lambda_function.%s.arn", resource.Name)
	case internal.ResourceTypeAPIGateway:
		return fmt.Sprintf("aws_apigatewayv2_api.%s.arn", resource.Name)
	case internal.ResourceTypeEC2:
		return fmt.Sprintf("aws_instance.%s.arn", resource.Name)
	default:
		return "\"*\""
	}
}
