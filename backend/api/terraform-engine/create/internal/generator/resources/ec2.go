package resources

import (
	"context"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
)

type EC2Generator struct {
	templateLoader *tftemplate.TemplateLoader
	ctx            context.Context
}

func NewEC2Generator(ctx context.Context, loader *tftemplate.TemplateLoader) *EC2Generator {
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
	if err := checkResourceName(variables); err != nil {
		return err
	}
	if v, ok := variables["instance_type"].(string); ok && !ec2InstanceTypes.Contains(v) {
		return invalidVariable("instance_type must be a supported x86 type such as t3.micro: t2, t3 or t3a from nano to 2xlarge, or m5, m6i, m7i, c5, c6i, c7i, r5, r6i or r7i from large to 4xlarge")
	}
	return nil
}

// ec2InstanceTypes are the instance types a diagram may choose. The template
// boots an x86_64 Amazon Linux image, so the list holds x86 types only.
var ec2InstanceTypes = hclsafe.NewAllowList(append(
	instanceTypes([]string{"t2", "t3", "t3a"}, "nano", "micro", "small", "medium", "large", "xlarge", "2xlarge"),
	instanceTypes([]string{"m5", "m6i", "m7i", "c5", "c6i", "c7i", "r5", "r6i", "r7i"}, "large", "xlarge", "2xlarge", "4xlarge")...,
)...)

// instanceTypes returns every family.size pair; each pair in the lists above
// exists in AWS.
func instanceTypes(families []string, sizes ...string) []string {
	var types []string
	for _, family := range families {
		for _, size := range sizes {
			types = append(types, family+"."+size)
		}
	}
	return types
}

func (g *EC2Generator) Generate(node generic.DiagramNode, resourceName string) (string, error) {
	if err := g.ValidateVariables(node.Variables); err != nil {
		return "", err
	}

	templatePath := tftemplate.GetTemplatePath("ec2", "main.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load ec2 template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
		"InstanceType": generic.GetString(node.Variables, "instance_type", "t3.micro"),
		"KeyName":      hclsafe.Escape(generic.GetString(node.Variables, "key_name", "")),
	}

	return tftemplate.Render(tmpl, vars)
}

func (g *EC2Generator) GetOutputs(resourceName string) (string, error) {
	templatePath := tftemplate.GetTemplatePath("ec2", "outputs.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load ec2 outputs template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
	}

	return tftemplate.Render(tmpl, vars)
}
