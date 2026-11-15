package resources

import (
	"context"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
)

type DynamoDBGenerator struct {
	templateLoader *tftemplate.TemplateLoader
	ctx            context.Context
}

func NewDynamoDBGenerator(ctx context.Context, loader *tftemplate.TemplateLoader) *DynamoDBGenerator {
	return &DynamoDBGenerator{
		templateLoader: loader,
		ctx:            ctx,
	}
}

func (g *DynamoDBGenerator) GetRequiredVariables() []string {
	return []string{"resource_name"}
}

func (g *DynamoDBGenerator) ValidateVariables(variables map[string]interface{}) error {
	required := g.GetRequiredVariables()
	var missing []string

	for _, varName := range required {
		if _, exists := variables[varName]; !exists {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required variables: %v", missing)
	}
	if err := checkResourceName(variables); err != nil {
		return err
	}
	if err := checkOneOf(variables, "billing_mode", dynamoDBBillingModes); err != nil {
		return err
	}
	return checkOneOf(variables, "hash_key_type", dynamoDBKeyTypes)
}

// dynamoDBBillingModes holds only on-demand billing: the template sets no read
// or write capacity, which PROVISIONED requires, so apply would fail.
var dynamoDBBillingModes = hclsafe.NewAllowList("PAY_PER_REQUEST")

// dynamoDBKeyTypes are the key attribute types: string, number and binary.
var dynamoDBKeyTypes = hclsafe.NewAllowList("S", "N", "B")

func (g *DynamoDBGenerator) Generate(node generic.DiagramNode, resourceName string) (string, error) {
	if err := g.ValidateVariables(node.Variables); err != nil {
		return "", err
	}

	templatePath := tftemplate.GetTemplatePath("dynamodb", "main.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load dynamodb template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
		"TableName":    generic.GetString(node.Variables, "resource_name", ""),
		"BillingMode":  generic.GetString(node.Variables, "billing_mode", "PAY_PER_REQUEST"),
		"HashKey":      hclsafe.Escape(generic.GetString(node.Variables, "hash_key", "id")),
		"HashKeyType":  generic.GetString(node.Variables, "hash_key_type", "S"),
		"EnableTTL":    generic.GetBool(node.Variables, "enable_ttl", false),
		"TTLAttribute": hclsafe.Escape(generic.GetString(node.Variables, "ttl_attribute", "ttl")),
	}

	return tftemplate.Render(tmpl, vars)
}

func (g *DynamoDBGenerator) GetOutputs(resourceName string) (string, error) {
	templatePath := tftemplate.GetTemplatePath("dynamodb", "outputs.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load dynamodb outputs template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
	}

	return tftemplate.Render(tmpl, vars)
}
