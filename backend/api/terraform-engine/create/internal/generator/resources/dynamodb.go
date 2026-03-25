package resources

import (
	"context"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator/template"
)

type DynamoDBGenerator struct {
	templateLoader *template.TemplateLoader
	ctx            context.Context
}

func NewDynamoDBGenerator(ctx context.Context, loader *template.TemplateLoader) *DynamoDBGenerator {
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
	return nil
}

func (g *DynamoDBGenerator) Generate(node generic.DiagramNode, resourceName string) (string, error) {
	if err := g.ValidateVariables(node.Variables); err != nil {
		return "", err
	}

	templatePath := template.GetTemplatePath("dynamodb", "main.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load dynamodb template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
		"TableName":    generic.GetString(node.Variables, "resource_name", ""),
		"BillingMode":  generic.GetString(node.Variables, "billing_mode", "PAY_PER_REQUEST"),
		"HashKey":      generic.GetString(node.Variables, "hash_key", "id"),
		"HashKeyType":  generic.GetString(node.Variables, "hash_key_type", "S"),
		"EnableTTL":    generic.GetBool(node.Variables, "enable_ttl", false),
		"TTLAttribute": generic.GetString(node.Variables, "ttl_attribute", "ttl"),
	}

	return template.Render(tmpl, vars)
}

func (g *DynamoDBGenerator) GetOutputs(resourceName string) string {
	templatePath := template.GetTemplatePath("dynamodb", "outputs.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return ""
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
	}

	output, err := template.Render(tmpl, vars)
	if err != nil {
		return ""
	}

	return output
}
