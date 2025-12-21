package main

import (
	"encoding/json"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

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
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from autorizer context", "error": err.Error()},
		)
	}
	// Get user ID from user authorizer data
	userId := userData.Id

	// Process body from API Gateway request
	var body generic.OrgRequestBody
	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request", "error": err.Error()})
	}
	// Get org name and/or description from request body
	orgName := body.OrganizationName
	description := body.Description

	return generic.Response(200, generic.Json{"orgId": orgId, "userId": userId, "orgName": orgName, "description": description})
}
