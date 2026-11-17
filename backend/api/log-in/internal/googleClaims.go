package internal

import (
	"errors"
	"strings"
)

// GoogleUser is what log-in takes from a validated Google ID token.
type GoogleUser struct {
	Email      string
	FullName   string
	PictureURL string
}

// CheckGoogleToken checks what idtoken.Validate does not: that the token was minted for
// Clutter's own Google client, and that Google has verified the email address. clientID
// must be non-empty, because idtoken.Validate skips the audience check when it is empty.
func CheckGoogleToken(clientID, audience string, claims map[string]interface{}) (GoogleUser, error) {
	if clientID == "" {
		return GoogleUser{}, errors.New("GOOGLE_CLIENT_ID is not set")
	}
	if audience != clientID {
		return GoogleUser{}, errors.New("token was issued for another app")
	}
	if verified, ok := claims["email_verified"].(bool); !ok || !verified {
		return GoogleUser{}, errors.New("email is not verified")
	}
	email, ok := claims["email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		return GoogleUser{}, errors.New("token has no email")
	}
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)
	return GoogleUser{Email: email, FullName: name, PictureURL: picture}, nil
}
