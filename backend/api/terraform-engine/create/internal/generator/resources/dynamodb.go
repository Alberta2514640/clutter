package resources

import (
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

type DynamoDBGenerator struct{}

func NewDynamoDBGenerator() *DynamoDBGenerator {
	return &DynamoDBGenerator{}
}

func (g *DynamoDBGenerator) GetRequiredVariables() []string {
	return []string{"table_name"}
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
	return "", nil
	
}

func (g *DynamoDBGenerator) GetOutputs(resourceName string) string {
	return ""
}