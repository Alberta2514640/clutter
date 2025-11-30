package main

import (
	"context"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/authorizer/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {

	lambda.Start(handler)

}

func handler(_ context.Context, event events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {

	// Get Google ID Token from Authorization header
	headers := event.Headers

	jwtToken, err := internal.GetHeader(headers, "Authorization")
	if err != nil {
		return internal.CreatePolicy("Unauthorized", "Deny", event.MethodArn), nil
	}

	// Decode JWT token
	jwtTokenClaims, err := internal.ParseAndVerifyJWTClaims(jwtToken)
	if err != nil {
		return internal.CreatePolicy("Unauthorized", "Deny", event.MethodArn), nil
	}

	// Iterate through required claims
	requiredClaims := []string{"sub", "email", "name", "pictureUrl", "accountCreatedOn"}
	claimValues := make(map[string]string)

	for _, claim := range requiredClaims {
		value, ok := jwtTokenClaims[claim].(string)
		if !ok {
			return internal.CreatePolicy("Unauthorized", "Deny", event.MethodArn), nil
		}
		claimValues[claim] = value
	}

	// Check if accessToken is expired
	accessTokenExpiration := jwtTokenClaims["exp"].(time.Time)
	timeNowUTC := time.Now().UTC()

	if timeNowUTC.After(accessTokenExpiration) {
		return internal.CreatePolicy("Unauthorized", "Deny", event.MethodArn), nil
	}

	// Include user data in context
	context := map[string]interface{}{
		"uuid":             claimValues["sub"],
		"email":            claimValues["email"],
		"name":             claimValues["name"],
		"pictureUrl":       claimValues["pictureUrl"],
		"accountCreatedOn": claimValues["accountCreatedOn"],
	}

	// Return Allow policy on user UUID (sub)
	policyResponse := internal.CreatePolicy(claimValues["sub"], "Allow", event.MethodArn, context)
	return policyResponse, nil

}
