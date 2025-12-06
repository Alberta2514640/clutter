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

type requestBody struct {
	OrganizationName string `json:"organizationName"`
	Description      string `json:"description"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context retains data about the user extracted from the JWT inside the authorizer Lambda function
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "Failed to retrieve user data from autorizer context", "error": err.Error()},
		)
	}

	// Process body from API Gateway request
	var body requestBody

	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request", "error": err.Error()})
	}
	organizationName := body.OrganizationName
	description := body.Description

	return generic.Response(
		http.StatusOK,
		generic.Json{"message": "organization-create Lambda is working!", "organizationName": organizationName, "description": description, "userData": userData},
	)
}
