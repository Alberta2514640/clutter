// Package main implements the diagram DELETE Lambda function for deleting diagrams from PostgreSQL
package main

import (
	"context"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

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

	projectID := request.QueryStringParameters["projectId"]
	diagramID := request.QueryStringParameters["diagramId"]

	if projectID == "" || diagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": "Missing required query parameters: projectId, diagramId",
			},
		})
	}

	// 2. Connect to PostgreSQL
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

	// 3. Authorization: Get project's organization and check membership
	orgID, err := generic.GetProjectOrganizationPSQL(ctx, conn, projectID)
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

	// 4. Check organization membership
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

	// 5. Delete diagram
	query := `DELETE FROM diagrams WHERE id = $1 AND project_id = $2`
	cmdTag, err := conn.Exec(ctx, query, diagramID, projectID)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete diagram",
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
		"message": "Diagram deleted successfully",
	})
}
