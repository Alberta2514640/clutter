package internal

import (
	"os"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// Generate JWT token
func GenerateJWT(email string, name string, picture string) (string, error) {

	timeNowUTC := time.Now().UTC()

	claims := map[string]any{
		jwt.ExpirationKey: timeNowUTC.Add(24 * time.Hour),
		"email":           email,
		"name":            name,
		"picture":         picture,
	}

	token := jwt.New()

	for k, v := range claims {
		token.Set(k, v)
	}

	jwtAlgorithm := jwa.HS256()
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))

	signedToken, err := jwt.Sign(
		token,
		jwt.WithKey(jwtAlgorithm, jwtSecret),
	)
	if err != nil {
		return "", err
	}

	return string(signedToken), nil

}
