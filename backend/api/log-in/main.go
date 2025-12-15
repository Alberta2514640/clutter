package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/log-in/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
	"google.golang.org/api/idtoken"
)

func main() {

	lambda.Start(handler)

}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	ctx := context.Background()

	// Parse JSON body
	var req internal.GoogleLoginRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"error": "invalid request body", "message": err.Error()})
	}

	// Validate Google ID token
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	payload, err := idtoken.Validate(ctx, req.Token, clientID)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{"error": "invalid Google token", "message": err.Error()})
	}

	// Extract user info from claims
	emailFromGoogle := payload.Claims["email"].(string)
	fullNameFromGoogle := payload.Claims["name"].(string)
	pictureUrlFromGoogle := payload.Claims["picture"].(string)

	// Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to connect to PostgreSQL database", "message": err.Error()})
	}
	defer conn.Close(ctx)

	// Initialize vars that the table will scan onto
	var existingUser internal.UserData
	var createdAt time.Time

	// Create query to get user using their e-mail
	queryExistingUser := `
		SELECT id, email, full_name, picture_url, created_at
		FROM users
		WHERE email=$1
	`

	// Query row and store returned values (if any) into UserData{} struct and createdAt time.Time value
	existingUserRow := conn.QueryRow(ctx, queryExistingUser, emailFromGoogle)
	err = existingUserRow.Scan(
		&existingUser.Uuid,
		&existingUser.Email,
		&existingUser.FullName,
		&existingUser.PictureUrl,
		&createdAt,
	)

	// Convert time.Time to string to be used in UserData{} struct
	existingUser.CreatedAt = createdAt.Format(time.RFC3339)

	// Ensure error for if no rows are returned does not lead to API failure response
	if err != nil && err != pgx.ErrNoRows {
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to query for existing user from table", "message": err.Error()})
	}

	// If the returned error is indicating there are no rows this user does not exist and we can proceed with new user creation
	if err == pgx.ErrNoRows {

		// Create the new user given the information from Google
		newUserData, err := internal.CreateNewUser(conn, ctx, emailFromGoogle, fullNameFromGoogle, pictureUrlFromGoogle)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to create new user", "message": err.Error()})
		}

		// Generate JWT token for new user
		token, exp, err := internal.GenerateUserJWT(&newUserData)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to generate JWT token for new user", "message": err.Error()})
		}

		// Return succes status with token, expiry in Unix time format, and user data
		return generic.Response(http.StatusCreated, generic.Json{
			"message":   "new user created successfully",
			"token":     token,
			"exp":       exp,
			"user_data": newUserData,
		})

	}

	// Otherwise user exists and we should check for any updates to their full name and picture URL
	err = internal.UpdateExistingUserIfNeeded(ctx, conn, &existingUser, fullNameFromGoogle, pictureUrlFromGoogle)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to update existing user",
			"message": err.Error(),
		})
	}

	// Generate JWT token for existing user
	token, exp, err := internal.GenerateUserJWT(&existingUser)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to generate JWT token for existing user",
			"message": err.Error(),
		})
	}

	// Return succes status with token, expiry in Unix time format, and up to date user data
	return generic.Response(http.StatusCreated, generic.Json{
		"message":   "existing user retrieved",
		"token":     token,
		"exp":       exp,
		"user_data": existingUser,
	})

}
