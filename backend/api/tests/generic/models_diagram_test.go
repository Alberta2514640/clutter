package generic_test

import (
	"encoding/json"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

func TestDiagramLayout_Marshal(t *testing.T) {
	// 1. Setup Mock Data (matching user provided example)
	mockNodes := []generic.DiagramNode{
		{
			ID:   "912e7d9c-62ca-4416-af4c-da5826671e7d",
			Type: "awsService",
			Position: generic.Position{
				X: 400,
				Y: 280,
			},
			Data: map[string]interface{}{
				"label": "Lambda",
				"img":   "λ",
			},
		},
		{
			ID:   "3f9f1679-d083-4bdc-83de-aa72e5c9b148",
			Type: "awsService",
			Position: generic.Position{
				X: 880,
				Y: 280,
			},
			Data: map[string]interface{}{
				"label": "DynamoDB",
				"img":   "DB",
			},
		},
	}

	mockEdges := []generic.DiagramEdge{
		{
			ID:     "xy-edge__912e7d9c-62ca-4416-af4c-da5826671e7d-3f9f1679-d083-4bdc-83de-aa72e5c9b148",
			Source: "912e7d9c-62ca-4416-af4c-da5826671e7d",
			Target: "3f9f1679-d083-4bdc-83de-aa72e5c9b148",
		},
	}

	layout := generic.DiagramLayout{
		Nodes: mockNodes,
		Edges: mockEdges,
	}

	// 2. Marshal
	bytes, err := json.Marshal(layout)
	if err != nil {
		t.Fatalf("Failed to marshal layout: %v", err)
	}

	// 3. Unmarshal back to check integrity
	var result generic.DiagramLayout
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal layout: %v", err)
	}

	// 4. Verify
	if len(result.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(result.Nodes))
	}
	if len(result.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(result.Edges))
	}
	if result.Nodes[0].ID != "912e7d9c-62ca-4416-af4c-da5826671e7d" {
		t.Errorf("Expected first node ID to match")
	}
	if result.Edges[0].Source != "912e7d9c-62ca-4416-af4c-da5826671e7d" {
		t.Errorf("Expected edge source to match")
	}
}
