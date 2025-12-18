// Package main implements the terraform-template GET Lambda function for retrieving
// Terraform templates from PostgreSQL based on resource type.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

// ResourceTemplate represents a template from the resources table.
// Matches: PK=resourceID, Platform, Type (SK), Version, Variables, Snippet, Allowed_connections
type ResourceTemplate struct {
	ResourceID         string                 `json:"resourceId"`
	Platform           string                 `json:"platform"`
	Type               string                 `json:"type"`
	Version            float64                `json:"version"`
	Variables          map[string]interface{} `json:"variables"`
	Snippet            string                 `json:"snippet"`
	AllowedConnections []string               `json:"allowedConnections"`
}

func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()

	// 1. Extract resource query parameter
	resource := request.QueryStringParameters["resource"]
	if resource == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "missing required query parameter: resource",
		})
	}

	// 2. Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	// 3. Query template from resources table
	// Columns: resource_id (PK), platform, type (SK), version, variables, snippet, allowed_connections
	query := `
		SELECT resource_id, platform, type, version, variables, snippet, allowed_connections
		FROM resources
		WHERE LOWER(type) = LOWER($1)
		LIMIT 1
	`

	var template ResourceTemplate
	var variablesJSON, connectionsJSON []byte

	err = conn.QueryRow(ctx, query, resource).Scan(
		&template.ResourceID,
		&template.Platform,
		&template.Type,
		&template.Version,
		&variablesJSON,
		&template.Snippet,
		&connectionsJSON,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generic.Response(http.StatusNotFound, generic.Json{
				"message": "template not found for resource: " + resource,
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch template",
			"error":   err.Error(),
		})
	}

	// 4. Unmarshal JSON columns
	if len(variablesJSON) > 0 {
		if err := json.Unmarshal(variablesJSON, &template.Variables); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to parse variables",
				"error":   err.Error(),
			})
		}
	}
	if len(connectionsJSON) > 0 {
		if err := json.Unmarshal(connectionsJSON, &template.AllowedConnections); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to parse allowed_connections",
				"error":   err.Error(),
			})
		}
	}

	// 5. Return template response with full resource data
	return generic.Response(http.StatusOK, generic.Json{
		"message": "template retrieved successfully",
		"data":    template,
	})
}
