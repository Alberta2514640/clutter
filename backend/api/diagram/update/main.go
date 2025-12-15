// Package main implements the diagram UPDATE Lambda function for updating diagrams in PostgreSQL
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

type UpdateRequest struct {
	ProjectID string                 `json:"projectId"`
	DiagramID string                 `json:"diagramId"`
	Name      *string                `json:"name,omitempty"`
	UILayout  *generic.DiagramLayout `json:"uiLayout,omitempty"`
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

	// 2. Parse request body
	var req UpdateRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "Invalid request body",
		})
	}

	if req.ProjectID == "" || req.DiagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "Missing required fields: projectId, diagramId",
		})
	}

	// Validate at least one field is being updated
	if req.Name == nil && req.UILayout == nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "At least one field (name or uiLayout) must be provided for update",
		})
	}

	// Validate name length if provided
	if req.Name != nil && len(*req.Name) > 32 {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "Diagram name must not exceed 32 characters",
		})
	}

	// Note: Empty uiLayout {} is valid - allows clearing diagram data

	// 3. Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// 4. Authorization: Get project's organization and check membership
	orgID, err := generic.GetProjectOrganizationPSQL(ctx, conn, req.ProjectID)
	if err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"message": authErr.Message,
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to fetch project",
		})
	}

	// 5. Check organization membership
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userID, orgID); err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"message": authErr.Message,
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to check authorization",
		})
	}

	// 6. Start transaction for atomic fetch + history + update
	timestamp := time.Now().UTC().Format(time.RFC3339)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to start transaction",
		})
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// 7. Fetch current diagram state with lock (FOR UPDATE prevents race conditions)
	var oldName string
	var oldData []byte
	fetchQuery := `SELECT name, data FROM diagrams WHERE id = $1 AND project_id = $2 FOR UPDATE`
	err = tx.QueryRow(ctx, fetchQuery, req.DiagramID, req.ProjectID).Scan(&oldName, &oldData)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generic.Response(http.StatusNotFound, generic.Json{
				"message": "Diagram not found",
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to fetch diagram for update",
		})
	}

	// 8. Get next version number (safe now due to FOR UPDATE lock above)
	var nextVersion int
	versionQuery := `SELECT COALESCE(MAX(version), 0) + 1 FROM diagram_history WHERE diagram_id = $1`
	err = tx.QueryRow(ctx, versionQuery, req.DiagramID).Scan(&nextVersion)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to get version number",
		})
	}

	// Insert into history
	historyQuery := `
		INSERT INTO diagram_history (diagram_id, version, updated_by, updated_at, name, data, comment)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.Exec(ctx, historyQuery, req.DiagramID, nextVersion, userID, timestamp, oldName, oldData, "Auto-saved before update")
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to save diagram history",
		})
	}

	// 8. Build Update Query
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
				"message": "Internal server error",
			})
		}
		setClauses = append(setClauses, "data = $"+fmt.Sprintf("%d", argCount))
		args = append(args, string(uiLayoutJSON))
	}

	query := "UPDATE diagrams SET " + strings.Join(setClauses, ", ") +
		" WHERE id = $1 AND project_id = $2"

	// 9. Execute Update within transaction
	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "unique_diagram_name_per_project") {
			return generic.Response(http.StatusConflict, generic.Json{
				"message": "A diagram with this name already exists in the project",
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to update diagram",
		})
	}

	// 10. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to commit update",
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "Diagram updated successfully",
	})
}
