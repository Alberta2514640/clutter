package main

import (
	"context"
	"fmt"
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

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Get org ID from path paremeter
	orgId := request.PathParameters["organizationId"]
	// Check if org ID is a valid UUID
	isValidUuid := generic.IsValidUuid(orgId)
	if !isValidUuid {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "inputted 'organizationId' path parameter is not a valid UUID.",
		})
	}

	// The authorizer context retains data about the user extracted from the JWT inside the authorizer Lambda function
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to retrieve user data from authorizer context",
		})
	}
	// Get user ID from user authorizer data
	userId := userData.Id

	// Connect to PostgreSQL
	ctx := context.Background()
	conn, err := connect()
	if err != nil {
		log.Printf("organization delete: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// Check membership of user to organization
	err = generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, orgId)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	// Create query to delete organization
	queryDeleteOrg := `
		DELETE FROM organizations
		WHERE id = $1
		RETURNING id, name;
	`

	// Delete organization
	var deletedOrgId string
	var deletedOrgName string
	err = conn.QueryRow(ctx, queryDeleteOrg, orgId).Scan(&deletedOrgId, &deletedOrgName)
	if generic.IsLiveDeploymentDelete(err) {
		return generic.LiveDeploymentsResponse()
	}
	if err != nil {
		log.Printf("organization delete: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to delete organization",
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message":          fmt.Sprintf("successfully deleted organization with ID %s", deletedOrgId),
		"deleted_org_id":   deletedOrgId,
		"deleted_org_name": deletedOrgName,
	})

}
