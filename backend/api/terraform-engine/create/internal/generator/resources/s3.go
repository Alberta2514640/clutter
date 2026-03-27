package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator/template"
)

type S3Generator struct {
	templateLoader *template.TemplateLoader
	ctx            context.Context
}

func NewS3Generator(ctx context.Context, loader *template.TemplateLoader) *S3Generator {
	return &S3Generator{
		templateLoader: loader,
		ctx:            ctx,
	}
}

func (g *S3Generator) GetRequiredVariables() []string {
	return []string{"resource_name"}
}

func (g *S3Generator) ValidateVariables(variables map[string]interface{}) error {
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

func (g *S3Generator) Generate(node generic.DiagramNode, resourceName string) (string, error) {
	if err := g.ValidateVariables(node.Variables); err != nil {
		return "", err
	}

	// Load main template from S3
	templatePath := template.GetTemplatePath("s3", "main.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load s3 template: %w", err)
	}

	// S3 bucket names must be lowercase with hyphens, no underscores
	bucketName := strings.ReplaceAll(
		strings.ToLower(generic.GetString(node.Variables, "resource_name", "")),
		"_", "-",
	)

	// Map diagram variables to template variables
	vars := map[string]interface{}{
		"ResourceName":      resourceName,
		"BucketName":        bucketName,
		"EnableVersioning":  generic.GetBool(node.Variables, "enable_versioning", false),
		"BlockPublicAccess": generic.GetBool(node.Variables, "block_public_access", true),
	}

	// Render template
	return template.Render(tmpl, vars)
}

func (g *S3Generator) GetOutputs(resourceName string) string {
	// Load outputs template from S3
	templatePath := template.GetTemplatePath("s3", "outputs.tf.tmpl")
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
