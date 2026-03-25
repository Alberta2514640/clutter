package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type SupportedResource struct {
	ID          string          `json:"id"`
	Label       string          `json:"label"`
	DisplayName string          `json:"displayName"`
	Description *string         `json:"description,omitempty"`
	Variables   json.RawMessage `json:"variables"`
	CreatedAt   time.Time       `json:"createdAt"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT id, label, display_name, description, variables, created_at
		FROM supported_resources
		ORDER BY display_name ASC
	`)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch supported resources",
			"error":   err.Error(),
		})
	}
	defer rows.Close()

	var resources []SupportedResource
	for rows.Next() {
		var r SupportedResource
		var rawVariables []byte
		if err := rows.Scan(&r.ID, &r.Label, &r.DisplayName, &r.Description, &rawVariables, &r.CreatedAt); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to scan resource row",
				"error":   err.Error(),
			})
		}
		r.Variables = json.RawMessage(rawVariables)
		resources = append(resources, r)
	}

	if resources == nil {
		resources = []SupportedResource{}
	}

	if err := rows.Err(); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to read resource data",
			"error":   err.Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "supported resources retrieved successfully",
		"data":    resources,
	})
}
