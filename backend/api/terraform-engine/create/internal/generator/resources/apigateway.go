package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
)

type APIGatewayGenerator struct {
	templateLoader *tftemplate.TemplateLoader
	ctx            context.Context
}

func NewAPIGatewayGenerator(ctx context.Context, loader *tftemplate.TemplateLoader) *APIGatewayGenerator {
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
	if err := checkResourceName(variables); err != nil {
		return err
	}
	_, err := apiHTTPMethods(variables)
	return err
}

// apiGatewayHTTPMethods are the methods the http_methods list may name. The
// template adds OPTIONS itself.
var apiGatewayHTTPMethods = hclsafe.NewAllowList("GET", "POST", "PUT", "PATCH", "DELETE")

// apiHTTPMethods returns the comma-separated http_methods list with spaces and
// empty entries removed, or an error if it names a method outside
// apiGatewayHTTPMethods.
func apiHTTPMethods(vars map[string]interface{}) (string, error) {
	var methods []string
	for _, m := range strings.Split(generic.GetString(vars, "http_methods", "POST"), ",") {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if !apiGatewayHTTPMethods.Contains(m) {
			return "", invalidVariable("http_methods may list only: %s", apiGatewayHTTPMethods)
		}
		methods = append(methods, m)
	}
	return strings.Join(methods, ","), nil
}

func (g *APIGatewayGenerator) Generate(node generic.DiagramNode, resourceName string) (string, error) {
	if err := g.ValidateVariables(node.Variables); err != nil {
		return "", err
	}
	httpMethods, err := apiHTTPMethods(node.Variables)
	if err != nil {
		return "", err
	}

	templatePath := tftemplate.GetTemplatePath("apigateway", "main.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load apigateway template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
		"APIName":      generic.GetString(node.Variables, "resource_name", ""),
		"Description":  hclsafe.Escape(generic.GetString(node.Variables, "description", "")),
		"StageName":    hclsafe.Escape(generic.GetString(node.Variables, "stage_name", "v1")),
		"EnableCORS":   generic.GetBool(node.Variables, "enable_cors", true),
		// A renders no route for an edge, so CORS's OPTIONS is the API's only method.
		"HasMethods":  generic.GetBool(node.Variables, "enable_cors", true),
		"HTTPMethods": httpMethods,
	}

	return tftemplate.Render(tmpl, vars)
}

func (g *APIGatewayGenerator) GetOutputs(resourceName string) (string, error) {
	templatePath := tftemplate.GetTemplatePath("apigateway", "outputs.tf.tmpl")
	tmpl, err := g.templateLoader.Load(g.ctx, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load apigateway outputs template: %w", err)
	}

	vars := map[string]interface{}{
		"ResourceName": resourceName,
	}

	return tftemplate.Render(tmpl, vars)
}
