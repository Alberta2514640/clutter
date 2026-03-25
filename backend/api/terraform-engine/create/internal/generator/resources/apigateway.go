package resources

import (
	"context"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator/template"
)

type APIGatewayGenerator struct {
	templateLoader *template.TemplateLoader
	ctx            context.Context
}

func NewAPIGatewayGenerator(ctx context.Context, loader *template.TemplateLoader) *APIGatewayGenerator {
	return &APIGatewayGenerator{
		templateLoader: loader,
		ctx:            ctx,
	}
}

func (g *APIGatewayGenerator) GetRequiredVariables() []string {
	return []string{"resource_name"}
}

func (g *APIGatewayGenerator) ValidateVariables(variables map[string]interface{}) error {
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

func (g *APIGatewayGenerator) Generate(node generic.DiagramNode, resourceName string) (string, error) {
	if err := g.ValidateVariables(node.Variables); err != nil {
		return "", err
	}

	templatePath := template.GetTemplatePath("apigateway", "main.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load apigateway template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
		"APIName":      generic.GetString(node.Variables, "resource_name", ""),
		"Description":  generic.GetString(node.Variables, "description", ""),
		"StageName":    generic.GetString(node.Variables, "stage_name", "v1"),
		"EnableCORS":   generic.GetBool(node.Variables, "enable_cors", true),
		"HTTPMethods":  generic.GetString(node.Variables, "http_methods", "POST"),
	}

	return template.Render(tmpl, vars)
}

func (g *APIGatewayGenerator) GetOutputs(resourceName string) string {
	templatePath := template.GetTemplatePath("apigateway", "outputs.tf.tmpl")
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
