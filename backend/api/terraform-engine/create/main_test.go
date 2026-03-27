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

// getEnvOrDefault returns the environment variable value or the default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestHandler(t *testing.T) {
	os.Setenv("TEMPLATE_BUCKET_NAME", getEnvOrDefault("TEMPLATE_BUCKET_NAME", "clutter-templates-us-west-2-b35a3c5c"))
	os.Setenv("S3_BUCKET_NAME", getEnvOrDefault("S3_BUCKET_NAME", "clutter-us-west-2-b35a3c5c"))

	if os.Getenv("PSQL_CONNECTION_STRING") == "" {
		t.Skip("Skipping TestHandler: PSQL_CONNECTION_STRING not set")
	}

	// Variables are nested inside data.variables to match the frontend shape
	payload := map[string]interface{}{
		"type":   "UPDATE",
		"table":  "diagrams",
		"schema": "public",
		"old_record": map[string]interface{}{
			"id":   "test-diagram-123",
			"name": "Old Name",
		},
		"record": map[string]interface{}{
			"id":         "test-diagram-123",
			"name":       "Test Architecture",
			"project_id": "13b9254f-9728-4f37-ab0e-e89a5ada3a18",
			"data": map[string]interface{}{
				"nodes": []map[string]interface{}{
					{
						"id":   "node-apigw-1",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "API-Gateway",
							"variables": map[string]interface{}{
								"resource_name": "my_api",
								"http_methods":  "GET,POST,DELETE",
							},
						},
					},
					{
						"id":   "node-lambda-1",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "Lambda",
							"variables": map[string]interface{}{
								"resource_name": "my_handler_one",
								"timeout":       30,
								"handler":       "bootstrap",
							},
						},
					},
					{
						"id":   "node-lambda-2",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "Lambda",
							"variables": map[string]interface{}{
								"resource_name": "my_handler_two",
								"timeout":       30,
								"handler":       "bootstrap",
							},
						},
					},
					{
						"id":   "node-lambda-3",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "Lambda",
							"variables": map[string]interface{}{
								"resource_name": "my_handler_three",
								"timeout":       30,
								"handler":       "bootstrap",
							},
						},
					},
					{
						"id":   "node-lambda-4",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "Lambda",
							"variables": map[string]interface{}{
								"resource_name": "my_handler_four",
								"timeout":       30,
								"handler":       "bootstrap",
							},
						},
					},
					{
						"id":   "node-dynamodb-1",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "DynamoDB",
							"variables": map[string]interface{}{
								"resource_name": "my_users_table",
								"hash_key":      "userId",
							},
						},
					},
					{
						"id":   "node-s3-1",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "S3",
							"variables": map[string]interface{}{
								"resource_name": "my_assets_bucket",
							},
						},
					},
					{
						"id":   "node-ec2-1",
						"type": "awsNode",
						"data": map[string]interface{}{
							"label": "EC2",
							"variables": map[string]interface{}{
								"resource_name": "my_server",
							},
						},
					},
				},
				"edges": []map[string]interface{}{
					{"id": "e1", "source": "node-apigw-1", "target": "node-lambda-1"},
					{"id": "e2", "source": "node-apigw-1", "target": "node-lambda-2"},
					{"id": "e3", "source": "node-apigw-1", "target": "node-lambda-3"},
					{"id": "e4", "source": "node-apigw-1", "target": "node-lambda-4"},
					{"id": "e5", "source": "node-lambda-1", "target": "node-dynamodb-1"},
					{"id": "e6", "source": "node-lambda-2", "target": "node-dynamodb-1"},
					{"id": "e7", "source": "node-lambda-3", "target": "node-dynamodb-1"},
					{"id": "e8", "source": "node-lambda-4", "target": "node-dynamodb-1"},
					{"id": "e9", "source": "node-lambda-1", "target": "node-s3-1"},
					{"id": "e10", "source": "node-lambda-2", "target": "node-s3-1"},
					{"id": "e11", "source": "node-lambda-3", "target": "node-s3-1"},
					{"id": "e12", "source": "node-lambda-4", "target": "node-s3-1"},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)

	request := events.APIGatewayProxyRequest{
		Body: string(body),
	}

	response, err := handler(context.Background(), request)
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}

	fmt.Printf("Status: %d\n", response.StatusCode)
	fmt.Printf("Body: %s\n", response.Body)

	if response.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d: %s", response.StatusCode, response.Body)
	}
}

// TestGenerateOnly tests just the terraform generation without S3 upload
func TestGenerateOnly(t *testing.T) {
	os.Setenv("TEMPLATE_BUCKET_NAME", getEnvOrDefault("TEMPLATE_BUCKET_NAME", "clutter-templates-us-west-2-b35a3c5c"))

	ctx := context.Background()

	gen, err := generator.NewTerraformGenerator(ctx, os.Getenv("TEMPLATE_BUCKET_NAME"))
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Variables nested inside Data["variables"] to match the frontend shape
	// SanitizeNodes extracts them into node.Variables before generation
	layout := generic.DiagramLayout{
		Nodes: []generic.DiagramNode{
			{
				ID:   "node-apigw-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "API-Gateway",
					"variables": map[string]interface{}{
						"resource_name": "my_api",
						"http_methods":  "GET,POST,DELETE",
					},
				},
			},
			{
				ID:   "node-lambda-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "Lambda",
					"variables": map[string]interface{}{
						"resource_name": "my_handler_one",
						"timeout":       30,
						"handler":       "bootstrap",
					},
				},
			},
			{
				ID:   "node-lambda-2",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "Lambda",
					"variables": map[string]interface{}{
						"resource_name": "my_handler_two",
						"timeout":       30,
						"handler":       "bootstrap",
					},
				},
			},
			{
				ID:   "node-lambda-3",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "Lambda",
					"variables": map[string]interface{}{
						"resource_name": "my_handler_three",
						"timeout":       30,
						"handler":       "bootstrap",
					},
				},
			},
			{
				ID:   "node-lambda-4",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "Lambda",
					"variables": map[string]interface{}{
						"resource_name": "my_handler_four",
						"timeout":       30,
						"handler":       "bootstrap",
					},
				},
			},
			{
				ID:   "node-dynamodb-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "DynamoDB",
					"variables": map[string]interface{}{
						"resource_name": "my_users_table",
						"hash_key":      "userId",
					},
				},
			},
			{
				ID:   "node-s3-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "S3",
					"variables": map[string]interface{}{
						"resource_name": "my_assets_bucket",
					},
				},
			},
			{
				ID:   "node-ec2-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "EC2",
					"variables": map[string]interface{}{
						"resource_name": "my_server",
					},
				},
			},
		},
		Edges: []generic.DiagramEdge{
			{ID: "e1", Source: "node-apigw-1", Target: "node-lambda-1"},
			{ID: "e2", Source: "node-apigw-1", Target: "node-lambda-2"},
			{ID: "e3", Source: "node-apigw-1", Target: "node-lambda-3"},
			{ID: "e4", Source: "node-apigw-1", Target: "node-lambda-4"},
			{ID: "e5", Source: "node-lambda-1", Target: "node-dynamodb-1"},
			{ID: "e6", Source: "node-lambda-2", Target: "node-dynamodb-1"},
			{ID: "e7", Source: "node-lambda-3", Target: "node-dynamodb-1"},
			{ID: "e8", Source: "node-lambda-4", Target: "node-dynamodb-1"},
			{ID: "e9", Source: "node-lambda-1", Target: "node-s3-1"},
			{ID: "e10", Source: "node-lambda-2", Target: "node-s3-1"},
			{ID: "e11", Source: "node-lambda-3", Target: "node-s3-1"},
			{ID: "e12", Source: "node-lambda-4", Target: "node-s3-1"},
		},
	}

	layout.Nodes = internal.SanitizeNodes(layout.Nodes)

	tf, errors := gen.Generate(ctx, "test-diagram-123", layout)

	if len(errors) > 0 {
		t.Log("Generation errors:")
		for _, e := range errors {
			t.Logf("  [%s] %s: %s", e.ErrorType, e.NodeLabel, e.Message)
		}
		t.FailNow()
	}

	fmt.Println("\n========== main.tf ==========")
	fmt.Println(tf.MainTF)
	fmt.Println("\n========== resources.tf ==========")
	fmt.Println(tf.ResourcesTF)
	fmt.Println("\n========== iam.tf ==========")
	fmt.Println(tf.IAMTF)
	fmt.Println("\n========== outputs.tf ==========")
	fmt.Println(tf.OutputsTF)
}
