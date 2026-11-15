package auth

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

var secret = []byte("test-jwt-secret")

func sign(t *testing.T, key []byte, sub string, exp time.Time) string {
	t.Helper()
	tok, err := jwt.NewBuilder().
		Subject(sub).
		Expiration(exp).
		Claim("email", "a@example.com").
		Claim("full_name", "A").
		Claim("picture_url", "").
		Claim("created_at", "2026-01-01").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	b, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256(), key))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMiddleware(t *testing.T) {
	user := uuid.NewString()
	var got string
	h := Middleware(JWTVerifier{Secret: secret}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) },
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := FromContext(r.Context())
		if !ok {
			t.Fatal("no identity in the context")
		}
		got = id.UserID
		w.WriteHeader(http.StatusOK)
	}))

	good := sign(t, secret, user, time.Now().Add(time.Hour))
	cases := []struct {
		name, header string
		want         int
	}{
		{"raw token as A sends it", good, http.StatusOK},
		{"bearer token", "Bearer " + good, http.StatusOK},
		{"no header", "", http.StatusUnauthorized},
		{"expired", sign(t, secret, user, time.Now().Add(-time.Minute)), http.StatusUnauthorized},
		{"other secret", sign(t, []byte("other-secret"), user, time.Now().Add(time.Hour)), http.StatusUnauthorized},
		{"sub is not a uuid", sign(t, secret, "not-a-uuid", time.Now().Add(time.Hour)), http.StatusUnauthorized},
		{"garbage", "abc.def.ghi", http.StatusUnauthorized},
	}
	for _, c := range cases {
		got = ""
		req := httptest.NewRequest("GET", "/v1/catalog", nil)
		if c.header != "" {
			req.Header.Set("Authorization", c.header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: %d, want %d", c.name, rec.Code, c.want)
		}
		if c.want == http.StatusOK && got != user {
			t.Errorf("%s: caller %q, want %q", c.name, got, user)
		}
	}
}

func TestVerifierWithoutSecretRefuses(t *testing.T) {
	tok := sign(t, secret, uuid.NewString(), time.Now().Add(time.Hour))
	if _, err := (JWTVerifier{}).Verify(t.Context(), tok); err == nil {
		t.Fatal("an empty secret accepted a token")
	}
}
