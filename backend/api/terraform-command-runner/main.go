package main

import (
	"encoding/json"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-command-runner/internal"
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
			generic.Json{"message": "failed to retrieve user data from authorizer context", "error": err.Error()},
		)
	}
	userId := userData.Id

	var body internal.RequestBody

	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request", "error": err.Error()})
	}
	organizationId := body.OrganizationId
	projectId := body.ProjectId
	diagramId := body.DiagramId
	command := body.Command

	return generic.Response(http.StatusOK, generic.Json{
		"message": "Terraform command executed successfully",
		"data": generic.Json{
			"organizationId": organizationId,
			"projectId":      projectId,
			"diagramId":      diagramId,
			"command":        command,
			"userId":         userId,
		},
	})
}
