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
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Data           *generic.DiagramLayout `json:"data"`
	CreatedBy      string                 `json:"createdBy"`
	CreatedAt      time.Time              `json:"createdAt"`
	LatestUpdateBy *string                `json:"latestUpdateBy,omitempty"`
	LatestUpdateAt *time.Time             `json:"latestUpdateAt,omitempty"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Extract user data from authorizer context
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
			"error":   err.Error(),
		})
	}
	userID := userData.Id

	projectID := request.QueryStringParameters["projectId"]
	diagramID := request.QueryStringParameters["diagramId"]

	if projectID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "missing required query parameter: projectId",
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

	// 3. Authorization: Get project's organization and check membership
	orgID, err := generic.GetProjectOrganizationPSQL(ctx, conn, projectID)
	if err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"message": authErr.Message,
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch project",
			"error":   err.Error(),
		})
	}

	// 4. Check organization membership
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userID, orgID); err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"message": authErr.Message,
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to check authorization",
			"error":   err.Error(),
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
		var rawData []byte
		err := conn.QueryRow(ctx, query, diagramID, projectID).Scan(
			&diagram.ID,
			&diagram.Name,
			&rawData,
			&diagram.CreatedBy,
			&diagram.CreatedAt,
			&diagram.LatestUpdateBy,
			&diagram.LatestUpdateAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return generic.Response(http.StatusNotFound, generic.Json{
					"message": "diagram not found",
				})
			}
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to fetch diagram",
				"error":   err.Error(),
			})
		}
		if err := json.Unmarshal(rawData, &diagram.Data); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to parse diagram data",
				"error":   err.Error(),
			})
		}

		return generic.Response(http.StatusOK, generic.Json{
			"message": "diagram retrieved successfully",
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
			"message": "failed to list diagrams",
			"error":   err.Error(),
		})
	}
	defer rows.Close()

	var diagrams []DiagramData
	for rows.Next() {
		var diagram DiagramData
		var rawData []byte
		err := rows.Scan(
			&diagram.ID,
			&diagram.Name,
			&rawData,
			&diagram.CreatedBy,
			&diagram.CreatedAt,
			&diagram.LatestUpdateBy,
			&diagram.LatestUpdateAt,
		)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to scan diagram row",
				"error":   err.Error(),
			})
		}
		if err := json.Unmarshal(rawData, &diagram.Data); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to unmarshal diagram data",
				"error":   err.Error(),
			})
		}

		diagrams = append(diagrams, diagram)
	}

	if diagrams == nil {
		diagrams = []DiagramData{}
	}

	if err := rows.Err(); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to read diagram data",
			"error":   err.Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "diagrams retrieved successfully",
		"data":    diagrams,
	})
}
