package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
)

// Request defines the expected input body
type Request struct {
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
}

// DiagramData defines the response data object
type DiagramData struct {
	ID             string  `json:"id"`
	ProjectID      string  `json:"projectId"`
	Name           string  `json:"name"`
	Data           string  `json:"data"`
	CreatedBy      string  `json:"createdBy"`
	CreatedAt      string  `json:"createdAt"`
	LatestUpdateBy *string `json:"latestUpdateBy,omitempty"`
	LatestUpdateAt *string `json:"latestUpdateAt,omitempty"`
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

	// 2. Parse Request Body
	var req Request
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INVALID_JSON",
				"message": "Invalid request body",
			},
		})
	}

	// 3. Validate Required Fields
	var missingFields []string
	if req.ProjectID == "" {
		missingFields = append(missingFields, "projectId")
	}
	if req.Name == "" {
		missingFields = append(missingFields, "name")
	}
	if len(missingFields) > 0 {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": "Missing required fields",
			},
		})
	}

	// Validate name length (max 32 characters per database schema)
	if len(req.Name) > 32 {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": "Diagram name must not exceed 32 characters",
			},
		})
	}

	// 4. Connect to PostgreSQL
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
	if conn == nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Database connection is invalid",
			},
		})
	}
	defer conn.Close(ctx)

	// 5. Authorization: Get project's organization and check membership
	orgID, err := generic.GetProjectOrganizationPSQL(ctx, conn, req.ProjectID)
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

	// 6. Check organization membership
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

	// 7. Create Diagram
	diagramID := uuid.NewString()
	timestamp := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO diagrams (id, project_id, created_by, name, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	emptyData := "{}"
	_, err = conn.Exec(ctx, query, diagramID, req.ProjectID, userID, req.Name, emptyData, timestamp)
	if err != nil {
		if strings.Contains(err.Error(), "unique_diagram_name_per_project") {
			return generic.Response(http.StatusConflict, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    "CONFLICT",
					"message": "A diagram with this name already exists in the project",
				},
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to save diagram",
			},
		})
	}

	// 8. Return Success
	response := DiagramData{
		ID:             diagramID,
		ProjectID:      req.ProjectID,
		Name:           req.Name,
		Data:           emptyData,
		CreatedBy:      userID,
		CreatedAt:      timestamp,
		LatestUpdateBy: nil,
		LatestUpdateAt: nil,
	}

	return generic.Response(http.StatusCreated, generic.Json{
		"success": true,
		"data":    response,
	})
}
