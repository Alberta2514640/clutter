// Package main implements the diagram CREATE Lambda function for creating diagrams in PostgreSQL
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
	ID             string                 `json:"id"`
	ProjectID      string                 `json:"projectId"`
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

	// 2. Parse Request Body
	var req Request
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid request body",
			"error":   err.Error(),
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
			"message": "missing required fields",
		})
	}

	// Validate name length (max 32 characters per database schema)
	if len(req.Name) > 32 {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "diagram name must not exceed 32 characters",
		})
	}

	// 4. Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	// 5. Authorization: Get project's organization and check membership
	orgID, err := generic.GetProjectOrganizationPSQL(ctx, conn, req.ProjectID)
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

	// 6. Check organization membership
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

	// 7. Create Diagram
	diagramID := uuid.NewString()
	timestamp := time.Now().UTC()

	query := `
		INSERT INTO diagrams (id, project_id, created_by, name, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	// Initialize with valid empty layout
	emptyLayout := generic.DiagramLayout{
		Nodes: []generic.DiagramNode{},
		Edges: []generic.DiagramEdge{},
	}
	emptyDataBytes, err := json.Marshal(emptyLayout)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to initialize diagram data",
			"error":   err.Error(),
		})
	}
	emptyData := string(emptyDataBytes)

	_, err = conn.Exec(ctx, query, diagramID, req.ProjectID, userID, req.Name, emptyData, timestamp)
	if err != nil {
		if strings.Contains(err.Error(), "unique_diagram_name_per_project") {
			return generic.Response(http.StatusConflict, generic.Json{
				"message": "a diagram with this name already exists in the project",
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to save diagram",
			"error":   err.Error(),
		})
	}

	// 8. Return Success
	response := DiagramData{
		ID:             diagramID,
		ProjectID:      req.ProjectID,
		Name:           req.Name,
		Data:           &emptyLayout,
		CreatedBy:      userID,
		CreatedAt:      timestamp,
		LatestUpdateBy: nil,
		LatestUpdateAt: nil,
	}

	return generic.Response(http.StatusCreated, generic.Json{
		"message": "diagram created successfully",
		"data":    response,
	})
}
