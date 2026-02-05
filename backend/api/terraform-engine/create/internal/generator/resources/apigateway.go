package resources

import (
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

type APIGatewayGenerator struct{}

func NewAPIGatewayGenerator() *APIGatewayGenerator {
	return &APIGatewayGenerator{}
}

func (g *APIGatewayGenerator) GetRequiredVariables() []string {
	return []string{"api_name"}
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
	return "", nil
	
}

func (g *APIGatewayGenerator) GetOutputs(resourceName string) string {
	return ""
}