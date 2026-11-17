package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/authorizer/internal"
	"github.com/Alberta2514640/clutter/backend/api/generic"
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

	// Verify the JWT with the same check B uses (generic.VerifyUserJWT)
	claims, err := generic.VerifyUserJWT(jwtToken, []byte(os.Getenv("JWT_SECRET")), time.Now().UTC())
	if err != nil {
		if !errors.Is(err, generic.ErrInvalidToken) {
			log.Printf("authorizer: %v", err)
		}
		return internal.CreatePolicy("Unauthorized", "Deny", event.MethodArn), nil
	}

	// Include user data in context
	context := map[string]interface{}{
		"id":          claims.ID,
		"email":       claims.Email,
		"name":        claims.FullName,
		"picture_url": claims.PictureURL,
		"created_at":  claims.CreatedAt,
	}

	// Return Allow policy on user ID (sub)
	policyResponse := internal.CreatePolicy(claims.ID, "Allow", event.MethodArn, context)
	return policyResponse, nil

}
