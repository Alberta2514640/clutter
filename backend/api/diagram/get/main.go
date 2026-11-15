// Package main implements the diagram GET Lambda function for retrieving diagrams from PostgreSQL
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
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

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Extract user data from authorizer context
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil || userData.Id == "" {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
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
	conn, err := connect()
	if err != nil {
		log.Printf("diagram get: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// 3. Authorization: the project (and the diagram, when one is named) belongs
	// to an organization the caller is in. A project or diagram of another
	// organization gets the same 404 as one that does not exist.
	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userID, "", projectID, diagramID)
	if err != nil {
		return generic.AuthorizationResponse(err)
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
		err := conn.QueryRow(ctx, query, ids.DiagramID, ids.ProjectID).Scan(
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
			log.Printf("diagram get: read the diagram: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to fetch diagram",
			})
		}
		if err := json.Unmarshal(rawData, &diagram.Data); err != nil {
			log.Printf("diagram get: parse diagram %s: %v", diagram.ID, err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to parse diagram data",
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
	rows, err := conn.Query(ctx, query, ids.ProjectID)
	if err != nil {
		log.Printf("diagram get: list diagrams: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to list diagrams",
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
			log.Printf("diagram get: scan a diagram row: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to list diagrams",
			})
		}
		if err := json.Unmarshal(rawData, &diagram.Data); err != nil {
			log.Printf("diagram get: parse diagram %s: %v", diagram.ID, err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to parse diagram data",
			})
		}

		diagrams = append(diagrams, diagram)
	}

	if diagrams == nil {
		diagrams = []DiagramData{}
	}

	if err := rows.Err(); err != nil {
		log.Printf("diagram get: read diagram rows: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to list diagrams",
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "diagrams retrieved successfully",
		"data":    diagrams,
	})
}
