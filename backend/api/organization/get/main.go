package main

import (
	"fmt"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/organization/get/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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
			generic.Json{"message": "failed to retrieve user data from autorizer context", "error": err.Error()},
		)
	}
	userId := userData.Id

	// Read organizationId from query parameters
	organizationId := request.QueryStringParameters["organizationId"]

	// If organizationId is not passed then return list of organizations for user instead
	if organizationId == "" {

		userOrganizationsData, err := internal.GetAllOrgsDataForUser(userId)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "something went wrong while getting data for all organizations",
				"error":   err.Error(),
			})
		}

		return generic.Response(200, generic.Json{
			"message": "successfully returned list of organizations (no organizationId passed)",
			"data":    userOrganizationsData,
		})
	}

	// Otherwise there is some input and we must validate it
	isValidUuid := generic.IsValidUuid(organizationId)

	// If organizationId is a valid UUID continue to getting single organization data
	if isValidUuid {

		organizationData, err := internal.GetSingleOrgDataWithId(userId, organizationId)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "something went wrong while getting data for single organization",
				"error":   err.Error(),
			})
		}

		return generic.Response(http.StatusOK, generic.Json{
			"message": fmt.Sprintf("successfully retrieved organization data for ORG#%s", organizationId),
			"data":    organizationData,
		})

	} else {

		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "inputted 'organizationId' query parameter is not a valid UUID.",
		})

	}
}
