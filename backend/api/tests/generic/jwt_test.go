package generic_test

import (
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func signToken(t *testing.T, secret []byte, alg jwa.SignatureAlgorithm, claims map[string]any) string {
	t.Helper()
	token := jwt.New()
	for k, v := range claims {
		if err := token.Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	signed, err := jwt.Sign(token, jwt.WithKey(alg, secret))
	if err != nil {
		t.Fatal(err)
	}
	return string(signed)
}

func TestVerifyUserJWT(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!!")
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	full := func() map[string]any {
		return map[string]any{
			"sub": "u1", "email": "a@example.com", "full_name": "A", "picture_url": "", "created_at": "2026-01-01",
			"exp": now.Add(time.Hour),
		}
	}
	without := func(key string) map[string]any { c := full(); delete(c, key); return c }
	with := func(key string, v any) map[string]any { c := full(); c[key] = v; return c }

	cases := []struct {
		name    string
		token   string
		secret  []byte
		wantErr bool
	}{
		{"valid, empty picture allowed", signToken(t, secret, jwa.HS256(), full()), secret, false},
		{"wrong secret", signToken(t, []byte("another-secret-at-least-32-bytes!!!"), jwa.HS256(), full()), secret, true},
		{"empty secret configured", signToken(t, secret, jwa.HS256(), full()), nil, true},
		{"expired", signToken(t, secret, jwa.HS256(), with("exp", now.Add(-time.Minute))), secret, true},
		{"no exp", signToken(t, secret, jwa.HS256(), without("exp")), secret, true},
		{"no sub", signToken(t, secret, jwa.HS256(), without("sub")), secret, true},
		{"empty email", signToken(t, secret, jwa.HS256(), with("email", "")), secret, true},
		{"claim of the wrong type", signToken(t, secret, jwa.HS256(), with("full_name", 7)), secret, true},
		{"other HMAC algorithm", signToken(t, secret, jwa.HS512(), full()), secret, true},
		{"not a token", "abc.def.ghi", secret, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			claims, err := generic.VerifyUserJWT(c.token, c.secret, now)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if err == nil && (claims.ID != "u1" || claims.Email != "a@example.com") {
				t.Fatalf("claims = %+v", claims)
			}
		})
	}
}
