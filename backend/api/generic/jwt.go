package generic

import (
	"errors"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// UserClaims are the claims log-in puts in Clutter's own JWT.
type UserClaims struct {
	ID         string // sub
	Email      string
	FullName   string
	PictureURL string
	CreatedAt  string
	ExpiresAt  time.Time
}

// ErrInvalidToken is returned for every token VerifyUserJWT refuses.
var ErrInvalidToken = errors.New("invalid token")

// VerifyUserJWT checks a token log-in issued: an HS256 signature with secret, the
// five user claims as strings, and an exp later than now. A's authorizer and B
// both call it, so the two accept exactly the same tokens.
func VerifyUserJWT(token string, secret []byte, now time.Time) (UserClaims, error) {
	if len(secret) == 0 {
		return UserClaims{}, errors.New("JWT secret is not set")
	}
	parsed, err := jwt.ParseString(token, jwt.WithKey(jwa.HS256(), secret), jwt.WithValidate(false))
	if err != nil {
		return UserClaims{}, ErrInvalidToken
	}

	var claims UserClaims
	textClaims := []struct {
		key      string
		dest     *string
		required bool // must be non-empty; the others may be empty (a Google account with no picture)
	}{
		{"sub", &claims.ID, true},
		{"email", &claims.Email, true},
		{"full_name", &claims.FullName, false},
		{"picture_url", &claims.PictureURL, false},
		{"created_at", &claims.CreatedAt, false},
	}
	for _, c := range textClaims {
		var value any
		if err := parsed.Get(c.key, &value); err != nil {
			return UserClaims{}, ErrInvalidToken
		}
		text, ok := value.(string)
		if !ok || (c.required && text == "") {
			return UserClaims{}, ErrInvalidToken
		}
		*c.dest = text
	}

	var exp any
	if err := parsed.Get("exp", &exp); err != nil {
		return UserClaims{}, ErrInvalidToken
	}
	expiresAt, ok := exp.(time.Time)
	if !ok || !now.Before(expiresAt) {
		return UserClaims{}, ErrInvalidToken
	}
	claims.ExpiresAt = expiresAt
	return claims, nil
}
