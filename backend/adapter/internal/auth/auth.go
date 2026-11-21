// Package auth signs in every /v1 request. It reads the raw Authorization
// header (A's frontend sends the JWT without "Bearer"), verifies it through a
// TokenVerifier, and puts the caller's user id in the request context.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/google/uuid"
)

// Identity is a signed-in caller.
type Identity struct {
	UserID string
}

// TokenVerifier checks a token and returns who it belongs to. It returns an
// error for a token that is malformed, badly signed, expired or meant for
// another audience.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (Identity, error)
}

// JWTVerifier accepts exactly the tokens A's authorizer accepts: Clutter's own
// HS256 JWT from log-in, checked by generic.VerifyUserJWT.
type JWTVerifier struct {
	Secret []byte
	// Now is the clock; nil means time.Now.
	Now func() time.Time
}

// Verify implements TokenVerifier.
func (v JWTVerifier) Verify(_ context.Context, token string) (Identity, error) {
	if len(v.Secret) == 0 {
		return Identity{}, errors.New("the JWT secret is not set")
	}
	now := time.Now
	if v.Now != nil {
		now = v.Now
	}
	claims, err := generic.VerifyUserJWT(token, v.Secret, now())
	if err != nil {
		return Identity{}, err
	}
	return Identity{UserID: claims.ID}, nil
}

type ctxKey struct{}

// FromContext returns the caller the middleware signed in.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}

// WithIdentity returns ctx carrying id, for tests of handlers.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// Middleware refuses a request without a valid token with 401, and passes the
// caller on otherwise. unauthorized writes the refusal.
func Middleware(v TokenVerifier, log *slog.Logger, unauthorized http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimSpace(r.Header.Get("Authorization"))
			if t, ok := strings.CutPrefix(token, "Bearer "); ok {
				token = strings.TrimSpace(t)
			}
			if token == "" {
				unauthorized(w, r)
				return
			}
			id, err := v.Verify(r.Context(), token)
			if err != nil {
				log.Info("sign-in refused", "path", r.URL.Path, "err", err)
				unauthorized(w, r)
				return
			}
			u, err := uuid.Parse(id.UserID)
			if err != nil {
				log.Warn("verified token has a user id that is not a UUID")
				unauthorized(w, r)
				return
			}
			id.UserID = u.String()
			next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), id)))
		})
	}
}
