package main

import (
	"log"
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

// What the user reads when sign-in fails on our side. The cause stays in the
// log.
const signInFailure = "sign-in failed; try again"

func main() {

	lambda.Start(handler)

}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	ctx := context.Background()

	// Parse JSON body
	var req internal.GoogleLoginRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"error": "invalid request body"})
	}

	// Validate Google ID token. idtoken.Validate skips the audience check when clientID is empty,
	// so an unset GOOGLE_CLIENT_ID is refused before it is called.
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		log.Print("log-in: GOOGLE_CLIENT_ID is not set")
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "sign-in is not configured"})
	}
	payload, err := idtoken.Validate(ctx, req.Token, clientID)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{"error": "invalid Google token"})
	}

	googleUser, err := internal.CheckGoogleToken(clientID, payload.Audience, payload.Claims)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{"error": "invalid Google token"})
	}
	emailFromGoogle := googleUser.Email
	fullNameFromGoogle := googleUser.FullName
	pictureUrlFromGoogle := googleUser.PictureURL

	// Connect to PostgreSQL
	conn, err := generic.PsqlConnect()
	if err != nil {
		log.Printf("log-in: connect to the database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": signInFailure})
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
		log.Printf("log-in: read the user: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": signInFailure})
	}

	// If the returned error is indicating there are no rows this user does not exist and we can proceed with new user creation
	if err == pgx.ErrNoRows {

		// Create the new user given the information from Google
		newUserData, err := internal.CreateNewUser(conn, ctx, emailFromGoogle, fullNameFromGoogle, pictureUrlFromGoogle)
		if err != nil {
			log.Printf("log-in: create the user: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{"error": signInFailure})
		}

		// Generate JWT token for new user
		token, exp, err := internal.GenerateUserJWT(&newUserData)
		if err != nil {
			log.Printf("log-in: sign the token for a new user: %v", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{"error": signInFailure})
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
		log.Printf("log-in: update the user: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": signInFailure})
	}

	// Generate JWT token for existing user
	token, exp, err := internal.GenerateUserJWT(&existingUser)
	if err != nil {
		log.Printf("log-in: sign the token: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": signInFailure})
	}

	// Return succes status with token, expiry in Unix time format, and up to date user data
	return generic.Response(http.StatusCreated, generic.Json{
		"message":   "existing user retrieved",
		"token":     token,
		"exp":       exp,
		"user_data": existingUser,
	})

}
