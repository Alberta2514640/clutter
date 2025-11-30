package internal

import (
	"os"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// Generate JWT token
func GenerateJWT(claims map[string]any) (string, error) {

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
