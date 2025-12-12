package internal

// Incoming request body
type GoogleLoginRequest struct {
	Token string `json:"token"`
}

// Outgoing user structure
type UserData struct {
	Uuid       string `json:"uuid"`
	Email      string `json:"email"`
	FullName   string `json:"full_name"`
	PictureUrl string `json:"picture_url"`
	CreatedAt  string `json:"created_at"`
}
