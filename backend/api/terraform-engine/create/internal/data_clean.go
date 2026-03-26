package internal

import (
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// HasDataChanged checks if the diagram data has changed in a way that's relevant
// to the terraform engine (ignores name-only changes)
func HasDataChanged(oldData, newData generic.DiagramRecord) bool {
	// Check if name has changed, ignore if only name changed
	if oldData.Name != newData.Name {
		return false
	}

	// TODO: Add more checks for position, measured, etc. if needed
	return true
}

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
