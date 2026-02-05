package resources

import (
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

type S3Generator struct{}

func NewS3Generator() *S3Generator {
	return &S3Generator{}
}

func (g *S3Generator) GetRequiredVariables() []string {
	return []string{"bucket_name"}
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
	return "", nil
	
}

func (g *S3Generator) GetOutputs(resourceName string) string {
	return ""
}
