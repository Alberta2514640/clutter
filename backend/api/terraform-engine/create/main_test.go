package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator"
	"github.com/aws/aws-lambda-go/events"
)

func TestHandler(t *testing.T) {
	// Set required env vars
	os.Setenv("TEMPLATE_BUCKET_NAME", "clutter-templates-us-west-2-b35a3c5c")
	os.Setenv("S3_BUCKET_NAME", "clutter-us-west-2-b35a3c5c")

	// Mock Supabase webhook payload
	payload := map[string]interface{}{
		"type":   "UPDATE",
		"table":  "diagrams",
		"schema": "public",
		"old_record": map[string]interface{}{
			"id":   "test-diagram-123",
			"name": "Old Name",
		},
		"record": map[string]interface{}{
			"id":   "test-diagram-123",
			"name": "Test Architecture",
			"data": map[string]interface{}{
				"nodes": []map[string]interface{}{
					{
						"id":   "node-lambda-1",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "Lambda",
						},
						"variables": map[string]interface{}{
							"resource_name": "my-test-function",
							"timeout":       30,
							"handler":       "bootstrap",
						},
					},
					{
						"id":   "node-s3-1",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "S3",
						},
						"variables": map[string]interface{}{
							"resource_name": "my-test-bucket",
						},
					},
				},
				"edges": []map[string]interface{}{
					{
						"id":     "edge-1",
						"source": "node-lambda-1",
						"target": "node-s3-1",
						"label":  "writes to",
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)

	// Create mock API Gateway request
	request := events.APIGatewayProxyRequest{
		Body: string(body),
	}

	// Call the actual handler
	response, err := handler(context.Background(), request)
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}

	fmt.Printf("Status: %d\n", response.StatusCode)
	fmt.Printf("Body: %s\n", response.Body)
}

// TestGenerateOnly tests just the terraform generation (no S3 upload)
func TestGenerateOnly(t *testing.T) {
	os.Setenv("TEMPLATE_BUCKET_NAME", "clutter-templates-us-west-2-b35a3c5c")

	ctx := context.Background()

	// Create generator
	gen, err := generator.NewTerraformGenerator(ctx, os.Getenv("TEMPLATE_BUCKET_NAME"))
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Mock diagram layout
	layout := generic.DiagramLayout{
		Nodes: []generic.DiagramNode{
			{
				ID:   "node-lambda-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "Lambda",
				},
				Variables: map[string]interface{}{
					"resource_name": "my-test-function",
					"timeout":       30,
					"handler":       "bootstrap",
				},
			},
			{
				ID:   "node-s3-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "S3",
				},
				Variables: map[string]interface{}{
					"resource_name": "my-test-bucket",
				},
			},
		},
		Edges: []generic.DiagramEdge{
			{
				ID:     "edge-1",
				Source: "node-lambda-1",
				Target: "node-s3-1",
				Label:  "writes to",
			},
		},
	}

	// Sanitize nodes
	layout.Nodes = internal.SanitizeNodes(layout.Nodes)

	// Generate terraform
	tf, errors := gen.Generate(ctx, "test-diagram-123", layout)

	if len(errors) > 0 {
		t.Log("Generation errors:")
		for _, e := range errors {
			t.Logf("  [%s] %s: %s", e.ErrorType, e.NodeLabel, e.Message)
		}
		t.FailNow()
	}

	// Print generated terraform
	fmt.Println("\n========== main.tf ==========")
	fmt.Println(tf.MainTF)
	fmt.Println("\n========== resources.tf ==========")
	fmt.Println(tf.ResourcesTF)
	fmt.Println("\n========== iam.tf ==========")
	fmt.Println(tf.IAMTF)
	fmt.Println("\n========== outputs.tf ==========")
	fmt.Println(tf.OutputsTF)
}
