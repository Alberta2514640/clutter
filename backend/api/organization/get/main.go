package main

import (
	"log"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/organization/get/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

func main() {

	lambda.Start(handler)

}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context retains data about the user (extracted from the JWT provided in the request)
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from authorizer context"},
		)
	}
	userId := userData.Id

	// Read organizationId from query parameters
	organizationId := request.QueryStringParameters["organizationId"]

	// If organizationId is not passed then return list of organizations for user instead
	if organizationId == "" {

		userOrganizationsData, err := internal.GetAllOrgsDataForUser(userId)
		if err != nil {
			log.Printf("organization get: list organizations: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "something went wrong while getting data for all organizations",
			})
		}

		return generic.Response(http.StatusOK, generic.Json{
			"message":     "successfully returned list of organizations (no 'organizationId' passed in query parameters)",
			"data_length": len(userOrganizationsData),
			"data":        userOrganizationsData,
		})
	}

	// Otherwise there is some input and we must validate it
	isValidUuid := generic.IsValidUuid(organizationId)

	// If organizationId is a valid UUID continue to getting single organization data
	if isValidUuid {

		organizationData, err := internal.GetSingleOrgDataWithId(userId, organizationId)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("organization get: read one organization: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "something went wrong while getting data for single organization",
			})
		} else if err == pgx.ErrNoRows {
			return generic.Response(http.StatusForbidden, generic.Json{
				"message": "user does not have access to requested organization data or organization with given ID does not exist",
			})
		}

		return generic.Response(http.StatusOK, generic.Json{
			"message": "successfully retrieved organization data",
			"data":    organizationData,
		})

	} else {

		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "inputted 'organizationId' query parameter is not a valid UUID.",
		})

	}
}
