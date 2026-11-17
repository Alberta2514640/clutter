package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1) Extract user identity from authorizer context
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
		})
	}
	userID := userData.Id

	// 2) Read query params
	orgID := request.QueryStringParameters["organizationId"]
	projectID := request.QueryStringParameters["projectId"]

	if orgID == "" || projectID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "organizationId and projectId query parameters are required",
		})
	}

	// 3) Validate UUID formats
	if _, err := uuid.Parse(userID); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "user id in token is not a valid UUID",
		})
	}
	if _, err := uuid.Parse(orgID); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "organizationId must be a valid UUID",
		})
	}
	if _, err := uuid.Parse(projectID); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "projectId must be a valid UUID",
		})
	}

	// 4) Connect to PostgreSQL
	conn, err := connect()
	if err != nil {
		log.Printf("project delete: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// 5) Authorization: user must be member of the organization
	if err := generic.CheckOrganizationMembershipPSQL(ctx, conn, userID, orgID); err != nil {
		return generic.AuthorizationResponse(err)
	}

	// 6) Ensure project exists and belongs to org
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
		log.Printf("project delete: read the project: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to fetch project",
		})
	}

	if actualOrgID != orgID {
		return generic.Response(http.StatusNotFound, generic.Json{
			"message": "project not found",
		})
	}

	// 7) Delete project
	queryDelete := `
		DELETE FROM projects
		WHERE id = $1 AND organization_id = $2
	`
	cmdTag, err := conn.Exec(ctx, queryDelete, projectID, orgID)
	if generic.IsLiveDeploymentDelete(err) {
		return generic.LiveDeploymentsResponse()
	}
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "violates foreign key constraint") {
			return generic.Response(http.StatusConflict, generic.Json{
				"message": "cannot delete project because it has dependent resources (e.g., diagrams). delete dependent resources first.",
			})
		}
		log.Printf("project delete: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to delete project",
		})
	}

	if cmdTag.RowsAffected() == 0 {
		return generic.Response(http.StatusNotFound, generic.Json{
			"message": "project not found",
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "project deleted successfully",
	})
}
