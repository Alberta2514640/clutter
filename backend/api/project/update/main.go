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
	"github.com/jackc/pgx/v5"
)

type UpdateProjectRequest struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
}

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

	// 2) Query param: projectId
	projectID := request.QueryStringParameters["projectId"]
	if projectID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "projectId query parameter is required",
		})
	}

	// Validate UUIDs
	if _, err := uuid.Parse(projectID); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "projectId must be a valid UUID",
		})
	}

	// 3) Parse request body
	var body UpdateProjectRequest
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid request body",
			"error":   err.Error(),
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

	// 6) Authorization: user must be member of the organization
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userID, body.OrganizationID); err != nil {
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

	// 7) Ensure project exists and belongs to organizationId
	queryProjectOrg := `
		SELECT organization_id
		FROM projects
		WHERE id = $1
	`
	var actualOrgID string
	if err := conn.QueryRow(ctx, queryProjectOrg, projectID).Scan(&actualOrgID); err != nil {
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

	if actualOrgID != body.OrganizationID {
		return generic.Response(http.StatusNotFound, generic.Json{
			"message": "project not found",
		})
	}

	// 8) Update project and return updated row
	queryUpdate := `
		UPDATE projects
		SET name = $1,
		    description = $2
		WHERE id = $3 AND organization_id = $4
		RETURNING id, organization_id, created_by, name, description, created_at
	`

	var updated ProjectData
	err = conn.QueryRow(ctx, queryUpdate,
		body.Name,
		body.Description,
		projectID,
		body.OrganizationID,
	).Scan(
		&updated.ID,
		&updated.OrganizationID,
		&updated.CreatedBy,
		&updated.Name,
		&updated.Description,
		&updated.CreatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "unique_project_name_per_org") || generic.IsUniqueViolation(err) {
			return generic.Response(http.StatusConflict, generic.Json{
				"message": "a project with this name already exists in the organization",
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to update project",
			"error":   err.Error(),
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "project updated successfully",
		"data":    updated,
	})
}
