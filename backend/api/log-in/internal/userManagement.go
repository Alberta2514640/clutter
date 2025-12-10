package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func CreateNewUser(conn *pgx.Conn, ctx context.Context, email string, fullName string, pictureUrl string) (UserData, error) {

	// Create required fields for new user
	uuid := uuid.NewString()

	// Build newUserData struct
	newUserData := UserData{
		Uuid:       uuid,
		Email:      email,
		FullName:   fullName,
		PictureUrl: pictureUrl,
	}

	// Create query to insert new user
	queryInsertNewUser := `
		INSERT INTO users (id, email, full_name, picture_url)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`

	// Initialize createdAt as a time.Time
	var createdAt time.Time

	insertedUserRow := conn.QueryRow(
		ctx,
		queryInsertNewUser,
		newUserData.Uuid,
		newUserData.Email,
		newUserData.FullName,
		newUserData.PictureUrl,
	)
	err := insertedUserRow.Scan(&createdAt)
	if err != nil {
		return UserData{}, err
	}

	// Convert time.Time to string to be used in UserData{} struct
	newUserData.CreatedAt = createdAt.Format(time.RFC3339)

	return newUserData, nil

}

// Check if the user's full name or picture URL has changed and update the database if needed
// Source: ChatGPT
func UpdateExistingUserIfNeeded(ctx context.Context, conn *pgx.Conn, existingUser *UserData, fullNameFromGoogle, pictureUrlFromGoogle string) error {

	updateNeeded := false
	query := "UPDATE users SET "
	args := []any{}
	argIndex := 1

	if existingUser.FullName != fullNameFromGoogle {
		query += fmt.Sprintf("full_name = $%d, ", argIndex)
		args = append(args, fullNameFromGoogle)
		existingUser.FullName = fullNameFromGoogle
		argIndex++
		updateNeeded = true
	}

	if existingUser.PictureUrl != pictureUrlFromGoogle {
		query += fmt.Sprintf("picture_url = $%d, ", argIndex)
		args = append(args, pictureUrlFromGoogle)
		existingUser.PictureUrl = pictureUrlFromGoogle
		argIndex++
		updateNeeded = true
	}

	if !updateNeeded {
		// Nothing changed, nothing to update
		return nil
	}

	// Remove trailing comma and space
	query = query[:len(query)-2]

	// WHERE clause
	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, existingUser.Uuid)

	// Execute the update
	_, err := conn.Exec(ctx, query, args...)
	return err

}
