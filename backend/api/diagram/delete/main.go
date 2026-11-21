// Package main implements the diagram DELETE Lambda function for deleting diagrams from PostgreSQL
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

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

	projectID := request.QueryStringParameters["projectId"]
	diagramID := request.QueryStringParameters["diagramId"]

	if projectID == "" || diagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "Missing required query parameters: projectId, diagramId",
		})
	}

	// 2. Connect to PostgreSQL
	conn, err := connect()
	if err != nil {
		log.Printf("diagram delete: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// 3. Authorization: the diagram belongs to the project, the project to an
	// organization the caller is in
	ids, err := generic.CheckOwnedIDsPSQL(ctx, conn, userID, "", projectID, diagramID)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	// 4. Delete diagram
	query := `DELETE FROM diagrams WHERE id = $1 AND project_id = $2`
	cmdTag, err := conn.Exec(ctx, query, ids.DiagramID, ids.ProjectID)
	if generic.IsLiveDeploymentDelete(err) {
		return generic.LiveDeploymentsResponse()
	}
	if err != nil {
		log.Printf("diagram delete: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "Failed to delete diagram",
		})
	}

	if cmdTag.RowsAffected() == 0 {
		return generic.Response(http.StatusNotFound, generic.Json{
			"message": "Diagram not found",
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "Diagram deleted successfully",
	})
}
