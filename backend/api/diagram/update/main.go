// Package main implements the diagram UPDATE Lambda function for updating diagrams in PostgreSQL
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type UpdateRequest struct {
	ProjectID string                 `json:"projectId"`
	DiagramID string                 `json:"diagramId"`
	Name      *string                `json:"name,omitempty"`
	UILayout  map[string]interface{} `json:"uiLayout,omitempty"`
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

	// 2. Parse request body
	var req UpdateRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INVALID_JSON",
				"message": "Invalid request body",
			},
		})
	}

	if req.ProjectID == "" || req.DiagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": "Missing required fields: projectId, diagramId",
			},
		})
	}

	// 3. Connect to PostgreSQL
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

	// 4. Authorization: Get project's organization and check membership
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

	// 5. Check organization membership
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

	// 6. Build Update Query
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Always update latest_update_at and latest_update_by
	setClauses := []string{"latest_update_at = $3", "latest_update_by = $4"}
	args := []interface{}{req.DiagramID, req.ProjectID, timestamp, userID}
	argCount := 4

	if req.Name != nil {
		argCount++
		setClauses = append(setClauses, "name = $"+fmt.Sprintf("%d", argCount))
		args = append(args, *req.Name)
	}

	if req.UILayout != nil {
		argCount++
		uiLayoutJSON, err := json.Marshal(req.UILayout)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    "INTERNAL_ERROR",
					"message": "Internal server error",
				},
			})
		}
		setClauses = append(setClauses, "data = $"+fmt.Sprintf("%d", argCount))
		args = append(args, string(uiLayoutJSON))
	}

	query := "UPDATE diagrams SET " + strings.Join(setClauses, ", ") +
		" WHERE id = $1 AND project_id = $2"

	// 7. Execute Update
	cmdTag, err := conn.Exec(ctx, query, args...)
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
				"message": "Failed to update diagram",
			},
		})
	}

	if cmdTag.RowsAffected() == 0 {
		return generic.Response(http.StatusNotFound, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "NOT_FOUND",
				"message": "Diagram not found",
			},
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"success": true,
		"message": "Diagram updated successfully",
	})
}
