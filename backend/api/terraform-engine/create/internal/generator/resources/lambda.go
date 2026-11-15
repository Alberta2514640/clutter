package resources

import (
	"context"
	"fmt"
	"path"
	"regexp"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
)

type LambdaGenerator struct {
	templateLoader *tftemplate.TemplateLoader
	ctx            context.Context
}

func NewLambdaGenerator(ctx context.Context, loader *tftemplate.TemplateLoader) *LambdaGenerator {
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
	if err := checkResourceName(variables); err != nil {
		return err
	}
	if err := checkOneOf(variables, "runtime", lambdaRuntimes); err != nil {
		return err
	}
	if err := checkOneOf(variables, "architecture", lambdaArchitectures); err != nil {
		return err
	}
	if folder := lambdaCodeFolder(variables); folder != "" {
		if err := hclsafe.CheckName(folder); err != nil {
			return invalidVariable("the uploaded code's folder name %v; upload the code again", err)
		}
	}
	for key := range generic.GetStringMap(variables, "environment_variables") {
		if !lambdaEnvKeyPattern.MatchString(key) {
			return invalidVariable("environment_variables names must start with a letter and use only letters, digits and underscores")
		}
	}
	return nil
}

// lambdaRuntimes are the Lambda runtimes a diagram may choose; generic owns the list.
var lambdaRuntimes = hclsafe.NewAllowList(generic.LambdaRuntimeNames()...)

var lambdaArchitectures = hclsafe.NewAllowList("arm64", "x86_64")

// lambdaEnvKeyPattern is AWS's rule for a Lambda environment variable name. A
// name that passes is also a bare HCL object key, which is how the template
// writes it.
var lambdaEnvKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// lambdaCodeFolder returns the folder of the uploaded code's S3 key, or "" when
// no code was uploaded. Code upload names that folder after the Lambda.
func lambdaCodeFolder(vars map[string]interface{}) string {
	s3Key := generic.GetString(vars, "s3_key", "")
	if s3Key == "" {
		return ""
	}
	return path.Base(path.Dir(s3Key))
}

func (g *LambdaGenerator) Generate(node generic.DiagramNode, resourceName string) (string, error) {
	if err := g.ValidateVariables(node.Variables); err != nil {
		return "", err
	}

	// Load main template from S3
	templatePath := tftemplate.GetTemplatePath("lambda", "main.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load lambda template: %w", err)
	}

	// If user has uploaded code, reference it by its S3 key path (downloaded
	// by the entrypoint before the client role is assumed). Otherwise fall back
	// to the default bootstrap.zip.
	filename := "bootstrap.zip"
	if folder := lambdaCodeFolder(node.Variables); folder != "" {
		// Use the folder name from the S3 key so it matches what the sync downloaded.
		// e.g. ".../terraform/code/my-lambda/bootstrap.zip" -> "code/my-lambda/bootstrap.zip"
		filename = "code/" + folder + "/bootstrap.zip"
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
		"FunctionName": generic.GetString(node.Variables, "resource_name", ""),
		"Handler":      hclsafe.Escape(generic.GetString(node.Variables, "handler", "main")),
		"Timeout":      generic.GetInt(node.Variables, "timeout", 3),
		"Runtime":      generic.GetString(node.Variables, "runtime", "provided.al2023"),
		"Architecture": generic.GetString(node.Variables, "architecture", "arm64"),
		"MemorySize":   generic.GetInt(node.Variables, "memory_size", 128),
		"Filename":     filename,
	}

	// Add environment variables if present. Their names are checked in
	// ValidateVariables; their values are free text.
	envVars := generic.GetStringMap(node.Variables, "environment_variables")
	if len(envVars) > 0 {
		for key, value := range envVars {
			envVars[key] = hclsafe.Escape(value)
		}
		vars["EnvironmentVariables"] = envVars
	}

	// Render template
	return tftemplate.Render(tmpl, vars)
}

func (g *LambdaGenerator) GetOutputs(resourceName string) (string, error) {
	// Load outputs template from S3
	templatePath := tftemplate.GetTemplatePath("lambda", "outputs.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load lambda outputs template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
	}

	return tftemplate.Render(tmpl, vars)
}
