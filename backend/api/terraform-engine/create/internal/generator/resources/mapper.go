package resources

import (
	"fmt"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"
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
	case "api-gateway":
		return internal.ResourceTypeAPIGateway, nil
	default:
		return "", fmt.Errorf("unknown resource type: %s", label)
	}
}

// GetGenerator returns the appropriate generator for a resource type
func GetGenerator(resourceType internal.ResourceType) (internal.ResourceGenerator, error) {
	switch resourceType {
	case internal.ResourceTypeLambda:
		return NewLambdaGenerator(), nil
	// case internal.ResourceTypeDynamoDB:
	// 	return NewDynamoDBGenerator(), nil
	case internal.ResourceTypeS3:
		return NewS3Generator(), nil
	// case internal.ResourceTypeAPIGateway:
	// 	return NewAPIGatewayGenerator(), nil
	default:
		return nil, fmt.Errorf("no generator for resource type: %s", resourceType)
	}
}
