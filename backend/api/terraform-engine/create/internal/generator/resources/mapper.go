package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator/template"
)

// ParseResourceType extracts ResourceType from node label
func ParseResourceType(label string) (internal.ResourceType, error) {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "lambda":
		return internal.ResourceTypeLambda, nil
	case "dynamodb":
		return internal.ResourceTypeDynamoDB, nil
	case "s3":
		return internal.ResourceTypeS3, nil
	case "api gateway", "api-gateway":
		return internal.ResourceTypeAPIGateway, nil
	case "ec2", "ec2 container":
		return internal.ResourceTypeEC2, nil
	default:
		return "", fmt.Errorf("unknown resource type: %s", label)
	}
}

// GetGenerator returns the appropriate generator for a resource type
func GetGenerator(ctx context.Context, resourceType internal.ResourceType, loader *template.TemplateLoader, templateBucket string) (internal.ResourceGenerator, error) {
	switch resourceType {
	case internal.ResourceTypeLambda:
		return NewLambdaGenerator(ctx, loader, templateBucket), nil
	case internal.ResourceTypeDynamoDB:
		return NewDynamoDBGenerator(ctx, loader), nil
	case internal.ResourceTypeS3:
		return NewS3Generator(ctx, loader), nil
	case internal.ResourceTypeAPIGateway:
		return NewAPIGatewayGenerator(ctx, loader), nil
	case internal.ResourceTypeEC2:
		return NewEC2Generator(ctx, loader), nil
	default:
		return nil, fmt.Errorf("no generator for resource type: %s", resourceType)
	}
}
