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

// ResourceTemplate represents a template from the resource table.
type ResourceTemplate struct {
	ResourceID         string                 `json:"resourceId"`
	Platform           string                 `json:"platform"`
	Type               string                 `json:"type"`
	Version            string                 `json:"version"` // Changed to string to match VARCHAR(20)
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
	resourceType := request.QueryStringParameters["resource"]
	if resourceType == "" {
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

	// 3. Query template from 'resource' table
	queryResource := `
		SELECT resource_id, platform, resource_type, resource_version, variables, snippet
		FROM resource
		WHERE LOWER(resource_type) = LOWER($1)
		LIMIT 1
	`

	var template ResourceTemplate
	var variablesJSON []byte

	err = conn.QueryRow(ctx, queryResource, resourceType).Scan(
		&template.ResourceID,
		&template.Platform,
		&template.Type,
		&template.Version,
		&variablesJSON,
		&template.Snippet,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generic.Response(http.StatusNotFound, generic.Json{
				"message": "template not found for resource type: " + resourceType,
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch template",
			"error":   err.Error(),
		})
	}

	// 4. Query allowed connections from 'resource_connections' table
	// We want target_resource_type where source is current resource
	queryConnections := `
		SELECT target_resource_type
		FROM resource_connections
		WHERE LOWER(source_resource_type) = LOWER($1)
	`
	rows, err := conn.Query(ctx, queryConnections, template.Type)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch resource connections",
			"error":   err.Error(),
		})
	}
	defer rows.Close()

	var connections []string
	for rows.Next() {
		var connType string
		if err := rows.Scan(&connType); err != nil {
			continue // Skip bad rows but try to return what we have
		}
		connections = append(connections, connType)
	}
	template.AllowedConnections = connections

	// 5. Unmarshal JSON columns
	if len(variablesJSON) > 0 {
		if err := json.Unmarshal(variablesJSON, &template.Variables); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to parse variables",
				"error":   err.Error(),
			})
		}
	}

	// 6. Return template response
	return generic.Response(http.StatusOK, generic.Json{
		"message": "template retrieved successfully",
		"data":    template,
	})
}
