package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/organization/accounts/get/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context retains data about the user (extracted from the JWT provided in the Authorization header)
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from authorizer context"},
		)
	}
	userId := userData.Id

	// Read path parameters
	// Required: organizationId (Path Param)
	organizationId := request.PathParameters["organizationId"]
	// Check to see if organization ID is not passed in
	if organizationId == "" {
		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": "no 'organizationId' path parameter inputted."},
		)
	}
	// Check to see if organization ID passed in is valid UUID
	isValidUuid := generic.IsValidUuid(organizationId)
	if !isValidUuid {
		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": "inputted 'organizationId' path parameter is not a valid UUID."},
		)
	}

	// Connect to PostgreSQL
	ctx := context.Background()
	conn, err := generic.PsqlConnect()
	if err != nil {
		log.Printf("accounts get: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	// Check membership of user to organization
	err = generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, organizationId)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	// Fetch all account access roles for the organization
	rows, err := conn.Query(ctx,
		`SELECT id, account_name, role_arn, status
		FROM aws_account_access_roles
		WHERE organization_id = $1`,
		organizationId,
	)
	if err != nil {
		log.Printf("accounts get: query: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to query account access roles",
		})
	}
	defer rows.Close()

	var roles []internal.AccountAccessRole
	for rows.Next() {
		var role internal.AccountAccessRole
		err := rows.Scan(&role.Id, &role.AccountName, &role.RoleArn, &role.Status)
		if err != nil {
			log.Printf("accounts get: scan: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to query account access roles",
			})
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		log.Printf("accounts get: rows: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to query account access roles",
		})
	}

	// Separate complete vs incomplete — incomplete ones may have a nil role_arn so skip that field

	var completeRoles []internal.AccountAccessRole
	var incompleteRoles []internal.AccountAccessRoleIncomplete

	for _, role := range roles {
		if role.Status == "incomplete" {
			incompleteRoles = append(incompleteRoles, internal.AccountAccessRoleIncomplete{
				Id:          role.Id,
				AccountName: role.AccountName,
				Status:      role.Status,
			})
		} else {
			completeRoles = append(completeRoles, role)
		}
	}

	return generic.Response(http.StatusOK, generic.Json{
		"message": "successfully returned list of account access roles",
		"data": generic.Json{
			"complete":   completeRoles,
			"incomplete": incompleteRoles,
		},
	})
}
