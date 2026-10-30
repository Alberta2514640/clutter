package devauth

import (
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// The authorizer (backend/api/authorizer) parses with HS256 and the shared
// secret, requires these five string claims, and reads exp as a time.
func TestSignMatchesAuthorizerExpectations(t *testing.T) {
	secret := []byte("devserver-local-test-secret-0123456789abcdef")
	u := UserData{Uuid: "u-1", Email: DevEmail, FullName: DevFullName, PictureUrl: "", CreatedAt: "2026-10-30T00:00:00Z"}
	now := time.Now()
	tok, exp, err := Sign(u, secret, now)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jwt.ParseString(tok, jwt.WithKey(jwa.HS256(), secret), jwt.WithValidate(false))
	if err != nil {
		t.Fatalf("authorizer-style parse failed: %v", err)
	}
	want := map[string]string{"sub": "u-1", "email": DevEmail, "full_name": DevFullName, "picture_url": "", "created_at": u.CreatedAt}
	for k, v := range want {
		var got any
		if err := parsed.Get(k, &got); err != nil {
			t.Fatalf("claim %s: %v", k, err)
		}
		if s, ok := got.(string); !ok || s != v {
			t.Errorf("claim %s = %#v, want %q", k, got, v)
		}
	}
	var expAny any
	if err := parsed.Get("exp", &expAny); err != nil {
		t.Fatal(err)
	}
	expTime, ok := expAny.(time.Time)
	if !ok || expTime.Unix() != exp || exp != now.Add(24*time.Hour).Unix() {
		t.Errorf("exp = %#v, want time at %d", expAny, now.Add(24*time.Hour).Unix())
	}
	if _, err := jwt.ParseString(tok, jwt.WithKey(jwa.HS256(), []byte("other")), jwt.WithValidate(false)); err == nil {
		t.Error("a token verified under the wrong secret")
	}
}
