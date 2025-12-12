// Package main implements the diagram GET Lambda function for retrieving diagrams from PostgreSQL
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

// DiagramData represents a diagram's data structure
type DiagramData struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Data           json.RawMessage `json:"data"`
	CreatedBy      string          `json:"createdBy"`
	CreatedAt      time.Time       `json:"createdAt"`
	LatestUpdateBy *string         `json:"latestUpdateBy,omitempty"`
	LatestUpdateAt *time.Time      `json:"latestUpdateAt,omitempty"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Extract userId from request
	userID, err := generic.GetUserIDFromRequest(request)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "UNAUTHORIZED",
				"message": err.Error(),
			},
		})
	}

	projectID := request.QueryStringParameters["projectId"]
	diagramID := request.QueryStringParameters["diagramId"]

	if projectID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": "Missing required query parameter: projectId",
			},
		})
	}

	// 2. Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to connect to database",
			},
		})
	}
	defer conn.Close(ctx)

	// 3. Authorization: Get project's organization and check membership
	orgID, err := generic.GetProjectOrganizationPSQL(ctx, conn, projectID)
	if err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    authErr.Code,
					"message": authErr.Message,
				},
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to fetch project",
			},
		})
	}

	// 4. Check organization membership
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userID, orgID); err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    authErr.Code,
					"message": authErr.Message,
				},
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check authorization",
			},
		})
	}

	// Case 1: Get Single Diagram
	if diagramID != "" {
		var diagram DiagramData

		query := `
			SELECT id, name, data, created_by, created_at, latest_update_by, latest_update_at
			FROM diagrams
			WHERE id = $1 AND project_id = $2
		`
		err := conn.QueryRow(ctx, query, diagramID, projectID).Scan(
			&diagram.ID,
			&diagram.Name,
			&diagram.Data,
			&diagram.CreatedBy,
			&diagram.CreatedAt,
			&diagram.LatestUpdateBy,
			&diagram.LatestUpdateAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return generic.Response(http.StatusNotFound, generic.Json{
					"success": false,
					"error": generic.Json{
						"code":    "NOT_FOUND",
						"message": "Diagram not found",
					},
				})
			}
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    "INTERNAL_ERROR",
					"message": "Failed to fetch diagram",
				},
			})
		}

		return generic.Response(http.StatusOK, generic.Json{
			"success": true,
			"data":    diagram,
		})
	}

	// Case 2: List Diagrams for Project
	query := `
		SELECT id, name, data, created_by, created_at, latest_update_by, latest_update_at
		FROM diagrams
		WHERE project_id = $1
		ORDER BY created_at DESC
	`
	rows, err := conn.Query(ctx, query, projectID)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list diagrams",
			},
		})
	}
	defer rows.Close()

	var diagrams []DiagramData
	for rows.Next() {
		var diagram DiagramData
		err := rows.Scan(
			&diagram.ID,
			&diagram.Name,
			&diagram.Data,
			&diagram.CreatedBy,
			&diagram.CreatedAt,
			&diagram.LatestUpdateBy,
			&diagram.LatestUpdateAt,
		)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    "INTERNAL_ERROR",
					"message": "Failed to parse diagram data",
				},
			})
		}

		diagrams = append(diagrams, diagram)
	}

	if diagrams == nil {
		diagrams = []DiagramData{}
	}

	if err := rows.Err(); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to read diagram data",
			},
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"success": true,
		"data":    diagrams,
	})
}
