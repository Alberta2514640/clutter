package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/log-in/internal"
)

func main() {
	userID := envOrDefault("CLUTTER_TEST_USER_ID", "11111111-1111-1111-1111-111111111111")
	email := envOrDefault("CLUTTER_TEST_USER_EMAIL", "clutter-test-user@example.com")
	fullName := envOrDefault("CLUTTER_TEST_USER_NAME", "Clutter Test User")
	pictureURL := envOrDefault("CLUTTER_TEST_USER_PICTURE_URL", "https://example.com/clutter-test-user.png")
	createdAt := envOrDefault("CLUTTER_TEST_USER_CREATED_AT", time.Now().UTC().Format(time.RFC3339))

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	token, exp, err := internal.GenerateUserJWT(&internal.UserData{
		Uuid:       userID,
		Email:      email,
		FullName:   fullName,
		PictureUrl: pictureURL,
		CreatedAt:  createdAt,
	})
	if err != nil {
		log.Fatalf("failed to generate JWT: %v", err)
	}

	fmt.Printf("TOKEN=%s\n", token)
	fmt.Printf("EXP=%d\n", exp)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
