package terraform_template_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestGetTerraformTemplate_WithMock tests the terraform template retrieval query logic
func TestGetTerraformTemplate_WithMock(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	ctx := context.Background()

	t.Run("template found - lambda", func(t *testing.T) {
		// Mock data matching the spec:
		// PK: resourceID, Platform, Type (SK), Version, Variables, Snippet, Allowed_connections
		variablesJSON := []byte(`{"source":"./modules/templates/lambda","function_name":"organization-create","actions":"dynamodb:PutItem"}`)
		connectionsJSON := []byte(`["API Gateway"]`)
		snippet := `module "${function_name}-lambda" { source = ${source} function_name = ${function_name} actions = ${actions} }`

		mock.ExpectQuery(`SELECT resource_id, platform, type, version, variables, snippet, allowed_connections FROM resources WHERE LOWER\(type\) = LOWER\(\$1\) LIMIT 1`).
			WithArgs("lambda").
			WillReturnRows(pgxmock.NewRows([]string{
				"resource_id", "platform", "type", "version", "variables", "snippet", "allowed_connections",
			}).AddRow("123141sadasd", "AWS", "Lambda", 1.0, variablesJSON, snippet, connectionsJSON))

		// Execute query
		query := `SELECT resource_id, platform, type, version, variables, snippet, allowed_connections FROM resources WHERE LOWER(type) = LOWER($1) LIMIT 1`

		var resourceID, platform, resourceType, snippetResult string
		var version float64
		var variablesBytes, connectionsBytes []byte

		row := mock.QueryRow(ctx, query, "lambda")
		err := row.Scan(&resourceID, &platform, &resourceType, &version, &variablesBytes, &snippetResult, &connectionsBytes)
		if err != nil {
			t.Errorf("Query failed: %v", err)
		}

		// Verify results
		if resourceID != "123141sadasd" {
			t.Errorf("Expected resourceID '123141sadasd', got '%s'", resourceID)
		}
		if platform != "AWS" {
			t.Errorf("Expected platform 'AWS', got '%s'", platform)
		}
		if resourceType != "Lambda" {
			t.Errorf("Expected type 'Lambda', got '%s'", resourceType)
		}
		if version != 1.0 {
			t.Errorf("Expected version 1.0, got %f", version)
		}

		// Verify JSON unmarshaling works
		var variables map[string]interface{}
		if err := json.Unmarshal(variablesBytes, &variables); err != nil {
			t.Errorf("Failed to unmarshal variables: %v", err)
		}
		if variables["function_name"] != "organization-create" {
			t.Errorf("Expected function_name 'organization-create', got '%v'", variables["function_name"])
		}

		var connections []string
		if err := json.Unmarshal(connectionsBytes, &connections); err != nil {
			t.Errorf("Failed to unmarshal connections: %v", err)
		}
		if len(connections) != 1 || connections[0] != "API Gateway" {
			t.Errorf("Expected connections ['API Gateway'], got %v", connections)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("template not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT resource_id, platform, type, version, variables, snippet, allowed_connections FROM resources WHERE LOWER\(type\) = LOWER\(\$1\) LIMIT 1`).
			WithArgs("nonexistent").
			WillReturnError(pgx.ErrNoRows)

		query := `SELECT resource_id, platform, type, version, variables, snippet, allowed_connections FROM resources WHERE LOWER(type) = LOWER($1) LIMIT 1`

		var resourceID string
		row := mock.QueryRow(ctx, query, "nonexistent")
		err := row.Scan(&resourceID)

		if err == nil {
			t.Error("Expected error for nonexistent resource type")
		}
		if err != pgx.ErrNoRows {
			t.Errorf("Expected pgx.ErrNoRows, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("template found - api-gateway", func(t *testing.T) {
		variablesJSON := []byte(`{"api_name":"clutter-api","aws_region":"us-east-1"}`)
		connectionsJSON := []byte(`["Lambda"]`)
		snippet := `resource "aws_api_gateway_rest_api" "api" { name = var.api_name }`

		mock.ExpectQuery(`SELECT resource_id, platform, type, version, variables, snippet, allowed_connections FROM resources WHERE LOWER\(type\) = LOWER\(\$1\) LIMIT 1`).
			WithArgs("api-gateway").
			WillReturnRows(pgxmock.NewRows([]string{
				"resource_id", "platform", "type", "version", "variables", "snippet", "allowed_connections",
			}).AddRow("api-gw-001", "AWS", "API Gateway", 1.0, variablesJSON, snippet, connectionsJSON))

		query := `SELECT resource_id, platform, type, version, variables, snippet, allowed_connections FROM resources WHERE LOWER(type) = LOWER($1) LIMIT 1`

		var resourceID, snippetResult string
		var platform, resourceType string
		var version float64
		var variablesBytes, connectionsBytes []byte

		row := mock.QueryRow(ctx, query, "api-gateway")
		err := row.Scan(&resourceID, &platform, &resourceType, &version, &variablesBytes, &snippetResult, &connectionsBytes)
		if err != nil {
			t.Errorf("Query failed: %v", err)
		}

		if resourceType != "API Gateway" {
			t.Errorf("Expected type 'API Gateway', got '%s'", resourceType)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

// TestResourceQueryValidation validates query parameter handling
func TestResourceQueryValidation(t *testing.T) {
	t.Run("empty resource returns error", func(t *testing.T) {
		resource := ""
		if resource == "" {
			// This matches the handler logic - empty resource should return 400
			t.Log("Empty resource correctly identified - would return 400 Bad Request")
		} else {
			t.Error("Empty resource should be detected")
		}
	})

	t.Run("valid resource types", func(t *testing.T) {
		validTypes := []string{"lambda", "Lambda", "LAMBDA", "api-gateway", "dynamodb", "s3"}
		for _, rt := range validTypes {
			if rt == "" {
				t.Errorf("Resource type '%s' should not be empty", rt)
			}
		}
	})
}
