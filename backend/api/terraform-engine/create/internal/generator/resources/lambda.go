package resources

import (
	"context"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator/template"
)

type LambdaGenerator struct {
	templateLoader *template.TemplateLoader
	ctx            context.Context
}

func NewLambdaGenerator(ctx context.Context, loader *template.TemplateLoader) *LambdaGenerator {
	return &LambdaGenerator{
		templateLoader: loader,
		ctx:            ctx,
	}
}

func (g *LambdaGenerator) GetRequiredVariables() []string {
	return []string{"resource_name"}
}

func (g *LambdaGenerator) ValidateVariables(variables map[string]interface{}) error {
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

func (g *LambdaGenerator) Generate(node generic.DiagramNode, resourceName string) (string, error) {
	if err := g.ValidateVariables(node.Variables); err != nil {
		return "", err
	}

	// Load main template from S3
	templatePath := template.GetTemplatePath("lambda", "main.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load lambda template: %w", err)
	}

	// If user has uploaded code, reference it by its S3 key path (downloaded
	// by the entrypoint before the client role is assumed). Otherwise fall back
	// to the default bootstrap.zip.
	// s3Key := generic.GetString(node.Variables, "s3_key", "")
	// var filename string
	// if s3Key != "" {
	// 	filename = s3Key
	// } else {
	// 	filename = "bootstrap.zip"
	// }

	vars := map[string]interface{}{
		"ResourceName": resourceName,
		"FunctionName": generic.GetString(node.Variables, "resource_name", ""),
		"Handler":      generic.GetString(node.Variables, "handler", "main"),
		"Timeout":      generic.GetInt(node.Variables, "timeout", 3),
		"Runtime":      generic.GetString(node.Variables, "runtime", "provided.al2023"),
		"Architecture": generic.GetString(node.Variables, "architecture", "arm64"),
		"MemorySize":   generic.GetInt(node.Variables, "memory_size", 128),
		"Filename":     "bootstrap.zip",
	}

	// Add environment variables if present
	envVars := generic.GetStringMap(node.Variables, "environment_variables")
	if len(envVars) > 0 {
		vars["EnvironmentVariables"] = envVars
	}

	// Render template
	return template.Render(tmpl, vars)
}

func (g *LambdaGenerator) GetOutputs(resourceName string) string {
	// Load outputs template from S3
	templatePath := template.GetTemplatePath("lambda", "outputs.tf.tmpl")
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
