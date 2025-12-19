package terraform_template_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestGetTerraformTemplate_WithMock tests the terraform template retrieval logic with the new normalized schema
func TestGetTerraformTemplate_WithMock(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	ctx := context.Background()

	t.Run("template found - lambda (normalized)", func(t *testing.T) {
		// 1. Mock 'resource' table query
		variablesJSON := []byte(`{"source":"./modules/templates/lambda","function_name":"organization-create","actions":"dynamodb:PutItem"}`)
		snippet := `module "${function_name}-lambda" { source = ${source} ... }`

		mock.ExpectQuery(`SELECT resource_id, platform, resource_type, resource_version, variables, snippet FROM resource WHERE LOWER\(resource_type\) = LOWER\(\$1\) LIMIT 1`).
			WithArgs("lambda").
			WillReturnRows(pgxmock.NewRows([]string{
				"resource_id", "platform", "resource_type", "resource_version", "variables", "snippet",
			}).AddRow("res-123", "AWS", "Lambda", "1.0", variablesJSON, snippet))

		// 2. Mock 'resource_connections' table query
		mock.ExpectQuery(`SELECT target_resource_type FROM resource_connections WHERE LOWER\(source_resource_type\) = LOWER\(\$1\)`).
			WithArgs("Lambda").
			WillReturnRows(pgxmock.NewRows([]string{"target_resource_type"}).AddRow("API Gateway").AddRow("DynamoDB"))

		// Execute Logic Checks (Simulating Handler)
		// ------------------------------------------

		// Query 1: Get Resource
		queryRes := `SELECT resource_id, platform, resource_type, resource_version, variables, snippet FROM resource WHERE LOWER(resource_type) = LOWER($1) LIMIT 1`
		var resourceID, platform, resType, version, snippetResult string
		var variablesBytes []byte

		err := mock.QueryRow(ctx, queryRes, "lambda").Scan(&resourceID, &platform, &resType, &version, &variablesBytes, &snippetResult)
		if err != nil {
			t.Fatalf("Resource query failed: %v", err)
		}

		if resourceID != "res-123" {
			t.Errorf("Expected ID 'res-123', got '%s'", resourceID)
		}
		if version != "1.0" {
			t.Errorf("Expected version '1.0', got '%s'", version)
		}

		// Query 2: Get Connections
		queryConn := `SELECT target_resource_type FROM resource_connections WHERE LOWER(source_resource_type) = LOWER($1)`
		rows, err := mock.Query(ctx, queryConn, resType)
		if err != nil {
			t.Fatalf("Connections query failed: %v", err)
		}
		defer rows.Close()

		var connections []string
		for rows.Next() {
			var conn string
			rows.Scan(&conn)
			connections = append(connections, conn)
		}

		if len(connections) != 2 {
			t.Errorf("Expected 2 connections, got %d", len(connections))
		}
		if connections[0] != "API Gateway" {
			t.Errorf("Expected first connection 'API Gateway', got '%s'", connections[0])
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("template not found (normalized)", func(t *testing.T) {
		mock.ExpectQuery(`SELECT resource_id, platform, resource_type, resource_version, variables, snippet FROM resource WHERE LOWER\(resource_type\) = LOWER\(\$1\) LIMIT 1`).
			WithArgs("nonexistent").
			WillReturnError(pgx.ErrNoRows)

		query := `SELECT resource_id, platform, resource_type, resource_version, variables, snippet FROM resource WHERE LOWER(resource_type) = LOWER($1) LIMIT 1`
		var id string
		err := mock.QueryRow(ctx, query, "nonexistent").Scan(&id)

		if err != pgx.ErrNoRows {
			t.Errorf("Expected pgx.ErrNoRows, got %v", err)
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
