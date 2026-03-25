package resources

import (
	"context"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator/template"
)

type EC2Generator struct {
	templateLoader *template.TemplateLoader
	ctx            context.Context
}

func NewEC2Generator(ctx context.Context, loader *template.TemplateLoader) *EC2Generator {
	return &EC2Generator{
		templateLoader: loader,
		ctx:            ctx,
	}
}

func (g *EC2Generator) GetRequiredVariables() []string {
	return []string{"resource_name"}
}

func (g *EC2Generator) ValidateVariables(variables map[string]interface{}) error {
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

func (g *EC2Generator) Generate(node generic.DiagramNode, resourceName string) (string, error) {
	if err := g.ValidateVariables(node.Variables); err != nil {
		return "", err
	}

	templatePath := template.GetTemplatePath("ec2", "main.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load ec2 template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
		"AMI":          generic.GetString(node.Variables, "ami", "ami-0c55b159cbfafe1f0"),
		"InstanceType": generic.GetString(node.Variables, "instance_type", "t3.micro"),
		"KeyName":      generic.GetString(node.Variables, "key_name", ""),
	}

	return template.Render(tmpl, vars)
}

func (g *EC2Generator) GetOutputs(resourceName string) string {
	templatePath := template.GetTemplatePath("ec2", "outputs.tf.tmpl")
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
