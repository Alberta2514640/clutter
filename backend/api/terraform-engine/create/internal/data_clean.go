package internal

import (
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// SanitizeNodes returns nodes with only the fields relevant to terraform.
func SanitizeNodes(nodes []generic.DiagramNode) []generic.DiagramNode {
	sanitized := make([]generic.DiagramNode, len(nodes))
	for i, node := range nodes {
		variables, _ := node.Data["variables"].(map[string]any)
		sanitized[i] = generic.DiagramNode{
			ID:        node.ID,
			Type:      node.Type,
			Data:      node.Data,
			Variables: variables,
		}
	}
	return sanitized
}
