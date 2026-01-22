package main

import (
	"context"
	"encoding/json"
	"net/http"
	// "strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
)

// Request defines the expected input body
type Request struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
}

// ProjectData defines the response data object
type ProjectData struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	CreatedBy      string    `json:"createdBy"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"createdAt"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1) Extract user data from authorizer context
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
			"error":   err.Error(),
		})
	}
	userID := userData.Id

	// 2) Parse request body
	var req Request
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid request body",
			"error":   err.Error(),
		})
	}

	// 3) Validate required fields
	var missingFields []string
	if req.OrganizationID == "" {
		missingFields = append(missingFields, "organizationId")
	}
	if req.Name == "" {
		missingFields = append(missingFields, "name")
	}
	if len(missingFields) > 0 {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "missing required fields",
		})
	}

	// 4) Validate UUID formats
	if _, err := uuid.Parse(req.OrganizationID); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "organizationId in token is not valid",
		})
	}
	if _, err := uuid.Parse(userID); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "user id in token is not valid",
		})
	}

	// 5) Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	// 6) Mandatory authorization: check organization membership
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userID, req.OrganizationID); err != nil {
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

	// 7) Create Project
	projectID := uuid.NewString()
	timestamp := time.Now().UTC()

	query := `
		INSERT INTO projects (id, organization_id, created_by, name, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = conn.Exec(ctx, query, projectID, req.OrganizationID, userID, req.Name, req.Description, timestamp)
	if err != nil {
		// if strings.Contains(err.Error(), "unique_project_name_per_org") {
		// 	return generic.Response(http.StatusConflict, generic.Json{
		// 		"message": "a project with this name already exists in the organization",
		// 	})
		// }

		// if generic.IsUniqueViolation(err) {
		// 	return generic.Response(http.StatusConflict, generic.Json{
		// 		"message": "project already exists",
		// 	})
		// }

		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to save project",
			"error":   err.Error(),
		})
	}

	// 8) Return Success
	response := ProjectData{
		ID:             projectID,
		OrganizationID: req.OrganizationID,
		CreatedBy:      userID,
		Name:           req.Name,
		Description:    req.Description,
		CreatedAt:      timestamp,
	}

	return generic.Response(http.StatusCreated, generic.Json{
		"message": "project created successfully",
		"data":    response,
	})
}