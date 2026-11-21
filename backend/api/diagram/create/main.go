// Package main implements the diagram CREATE Lambda function for creating diagrams in PostgreSQL
package main

import (
	"context"
	"encoding/json"
	"log"
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

	// 2. Parse Request Body
	var req Request
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid request body",
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
	conn, err := connect()
	if err != nil {
		log.Printf("diagram create: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// 5. Authorization: the project belongs to an organization the caller is
	// in. A project of another organization gets the same 404 as one that does
	// not exist, and the row is written under the checked, canonical id.
	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userID, "", req.ProjectID, "")
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	// 6. Create Diagram
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
		log.Printf("diagram create: marshal the empty layout: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to initialize diagram data",
		})
	}
	emptyData := string(emptyDataBytes)

	_, err = conn.Exec(ctx, query, diagramID, ids.ProjectID, userID, req.Name, emptyData, timestamp)
	if err != nil {
		if strings.Contains(err.Error(), "unique_diagram_name_per_project") {
			return generic.Response(http.StatusConflict, generic.Json{
				"message": "a diagram with this name already exists in the project",
			})
		}
		log.Printf("diagram create: insert: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to save diagram",
		})
	}

	// 7. Return Success
	response := DiagramData{
		ID:             diagramID,
		ProjectID:      ids.ProjectID,
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
