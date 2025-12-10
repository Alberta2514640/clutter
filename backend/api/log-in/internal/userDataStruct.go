package internal

import "time"

// Outgoing user structure
type UserData struct {
	Uuid       string    `json:"uuid"`
	Email      string    `json:"email"`
	FullName   string    `json:"full_name"`
	PictureUrl string    `json:"picture_url"`
	CreatedAt  time.Time `json:"created_at"`
}
