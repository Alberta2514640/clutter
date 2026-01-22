package main

import (
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
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
			http.StatusUnauthorized,
			generic.Json{"message": "unauthorized: missing user identity", "error": err.Error()},
		)
	}

	// Return the user information extracted from the JWT token
	// Using snake_case JSON keys to match existing login endpoint format
	return generic.Response(http.StatusOK, generic.Json{
		"message": "successfully retrieved user information",
		"data": map[string]string{
			"uuid":        userData.Id,
			"email":       userData.Email,
			"full_name":   userData.Name,
			"picture_url": userData.PictureUrl,
			"created_at":  userData.AccountCreatedOn,
		},
	})
}
