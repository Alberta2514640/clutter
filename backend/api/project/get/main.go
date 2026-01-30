package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DiagramSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectData struct {
	ID             string           `json:"id"`
	OrganizationID string           `json:"organizationId"`
	CreatedBy      string           `json:"createdBy"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	CreatedAt      time.Time        `json:"createdAt"`
	Diagrams       []DiagramSummary `json:"diagrams"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1) Extract user data from authorizer context (no fallbacks)
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
			"error":   err.Error(),
		})
	}
	userID := userData.Id

	// 2) Read query params
	orgID := request.QueryStringParameters["organizationId"]
	projectID := request.QueryStringParameters["projectId"] // optional

	if orgID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "missing required fields",
		})
	}

	// Validate UUID formats
	if _, err := uuid.Parse(orgID); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "organizationId must be a valid UUID",
		})
	}
	if _, err := uuid.Parse(userID); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "user id in token is not a valid UUID",
		})
	}
	if projectID != "" {
		if _, err := uuid.Parse(projectID); err != nil {
			return generic.Response(http.StatusBadRequest, generic.Json{
				"message": "projectId must be a valid UUID",
			})
		}
	}

	// 3) Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	// 4) Authorization: user must be member of this organization
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

	// 5) Branch: single project vs list
	if projectID != "" {
		// 5A) Fetch one project
		query := `
			SELECT
				p.id,
				p.organization_id,
				p.created_by,
				p.name,
				p.description,
				p.created_at,
				COALESCE(
					json_agg(
						json_build_object('id', d.id, 'name', d.name)
					) FILTER (WHERE d.id IS NOT NULL),
					'[]'
				) AS diagrams
			FROM projects p
			LEFT JOIN diagrams d ON d.project_id = p.id
			WHERE p.id = $1 AND p.organization_id = $2
			GROUP BY p.id
		`

		var p ProjectData
		var diagramsJson []byte

		if err := conn.QueryRow(ctx, query, projectID, orgID).Scan(
			&p.ID,
			&p.OrganizationID,
			&p.CreatedBy,
			&p.Name,
			&p.Description,
			&p.CreatedAt,
			&diagramsJson,
		); err != nil {
			if err == pgx.ErrNoRows {
				return generic.Response(http.StatusNotFound, generic.Json{
					"message": "project not found",
				})
			}
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to fetch project",
				"error":   err.Error(),
			})
		}

		if err := json.Unmarshal(diagramsJson, &p.Diagrams); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to parse diagrams data",
				"error":   err.Error(),
			})
		}

		return generic.Response(http.StatusOK, generic.Json{
			"message": "project fetched successfully",
			"data":    p,
		})
	}

	// 5B) List all projects in org
	query := `
		SELECT
			p.id,
			p.organization_id,
			p.created_by,
			p.name,
			p.description,
			p.created_at,
			COALESCE(
				json_agg(
					json_build_object('id', d.id, 'name', d.name)
				) FILTER (WHERE d.id IS NOT NULL),
				'[]'
			) AS diagrams
		FROM projects p
		LEFT JOIN diagrams d ON d.project_id = p.id
		WHERE p.organization_id = $1
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := conn.Query(ctx, query, orgID)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch projects",
			"error":   err.Error(),
		})
	}
	defer rows.Close()

	projects := []ProjectData{}
	for rows.Next() {
		var p ProjectData
		var diagramsJson []byte

		if err := rows.Scan(
			&p.ID,
			&p.OrganizationID,
			&p.CreatedBy,
			&p.Name,
			&p.Description,
			&p.CreatedAt,
			&diagramsJson,
		); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to parse projects",
				"error":   err.Error(),
			})
		}

		if err := json.Unmarshal(diagramsJson, &p.Diagrams); err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to parse diagrams data",
				"error":   err.Error(),
			})
		}

		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to iterate projects",
			"error":   err.Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "projects fetched successfully",
		"data":    projects,
	})
}
