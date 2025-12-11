package internal

import (
	"os"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// Generate user specific JWT
func GenerateUserJWT(user *UserData) (string, int64, error) {

	exp := time.Now().Add(24 * time.Hour).Unix()

	claims := user.toJWTClaims()
	claims["exp"] = exp

	token, err := generateJWT(claims)
	if err != nil {
		return "", 0, err
	}

	return token, exp, nil
}

// Generate JWT token
func generateJWT(claims map[string]any) (string, error) {

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

func (u UserData) toJWTClaims() map[string]any {
	return map[string]any{
		"sub":         u.Uuid,
		"email":       u.Email,
		"full_name":   u.FullName,
		"picture_url": u.PictureUrl,
		"created_at":  u.CreatedAt,
	}
}
