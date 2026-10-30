// Package devauth is the devserver's stand-in for Google sign-in. It exists
// only in the devserver, never in a Lambda: it upserts a fixed local user and
// signs a token with the same claims, algorithm and secret as the log-in
// Lambda (backend/api/log-in/internal/jwtGeneration.go), so the real
// authorizer and handlers accept it unchanged.
package devauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// The fixed local identity. The email domain is reserved (RFC 6761), so it
// can never collide with a real Google account in a shared database.
const (
	DevEmail    = "dev@clutter.localhost"
	DevFullName = "Dev User"
	DevPicture  = ""
)

// Token lifetime, as in the log-in Lambda.
const tokenTTL = 24 * time.Hour

// The cookie the frontend writes after log-in (frontend/clutter/src/lib/features/user/api.ts).
const (
	cookieName   = "clutter_token"
	cookieMaxAge = 60 * 60 * 24 * 7
)

// UserData has the JSON shape of the log-in Lambda's user_data.
type UserData struct {
	Uuid       string `json:"uuid"`
	Email      string `json:"email"`
	FullName   string `json:"full_name"`
	PictureUrl string `json:"picture_url"`
	CreatedAt  string `json:"created_at"`
}

// LoginResponse has the JSON shape of a 201 from POST /log-in.
type LoginResponse struct {
	Message  string   `json:"message"`
	Token    string   `json:"token"`
	Exp      int64    `json:"exp"`
	UserData UserData `json:"user_data"`
}

// Issuer signs tokens for the dev user.
type Issuer struct {
	ConnString string
	Secret     []byte
	// FrontendOrigin is where the browser lands after a dev login.
	FrontendOrigin string
}

// UpsertDevUser finds or creates the dev user with the queries the log-in
// Lambda uses.
func UpsertDevUser(ctx context.Context, conn *pgx.Conn) (UserData, bool, error) {
	var u UserData
	var createdAt time.Time
	err := conn.QueryRow(ctx,
		`SELECT id, email, full_name, picture_url, created_at FROM users WHERE email=$1`,
		DevEmail,
	).Scan(&u.Uuid, &u.Email, &u.FullName, &u.PictureUrl, &createdAt)
	if err == nil {
		u.CreatedAt = createdAt.Format(time.RFC3339)
		return u, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return UserData{}, false, err
	}
	u = UserData{Uuid: uuid.NewString(), Email: DevEmail, FullName: DevFullName, PictureUrl: DevPicture}
	err = conn.QueryRow(ctx,
		`INSERT INTO users (id, email, full_name, picture_url) VALUES ($1, $2, $3, $4) RETURNING created_at`,
		u.Uuid, u.Email, u.FullName, u.PictureUrl,
	).Scan(&createdAt)
	if err != nil {
		return UserData{}, false, err
	}
	u.CreatedAt = createdAt.Format(time.RFC3339)
	return u, true, nil
}

// Sign builds the same claims as the log-in Lambda and signs them with HS256.
func Sign(u UserData, secret []byte, now time.Time) (string, int64, error) {
	exp := now.Add(tokenTTL).Unix()
	claims := map[string]any{
		"sub":         u.Uuid,
		"email":       u.Email,
		"full_name":   u.FullName,
		"picture_url": u.PictureUrl,
		"created_at":  u.CreatedAt,
		"exp":         exp,
	}
	tok := jwt.New()
	for k, v := range claims {
		if err := tok.Set(k, v); err != nil {
			return "", 0, err
		}
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256(), secret))
	if err != nil {
		return "", 0, err
	}
	return string(signed), exp, nil
}

// Login upserts the dev user and returns a log-in-shaped response.
func (i *Issuer) Login(ctx context.Context) (LoginResponse, error) {
	conn, err := pgx.Connect(ctx, i.ConnString)
	if err != nil {
		return LoginResponse{}, errors.New("failed to connect to the local Postgres")
	}
	defer conn.Close(ctx)
	u, created, err := UpsertDevUser(ctx, conn)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to upsert the dev user: %w", err)
	}
	token, exp, err := Sign(u, i.Secret, time.Now())
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to sign the dev token: %w", err)
	}
	msg := "existing user retrieved"
	if created {
		msg = "new user created successfully"
	}
	return LoginResponse{Message: msg, Token: token, Exp: exp, UserData: u}, nil
}

// JSONHandler answers with the log-in-shaped JSON. The smoke script and curl
// use it.
func (i *Issuer) JSONHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "use GET", http.StatusMethodNotAllowed)
			return
		}
		resp, err := i.Login(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, resp)
	})
}

// RedirectHandler sends a browser to the front door's /dev/login, which runs
// on the frontend's origin and can therefore write its localStorage.
func (i *Issuer) RedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") == "json" {
			i.JSONHandler().ServeHTTP(w, r)
			return
		}
		target, err := url.JoinPath(i.FrontendOrigin, "/dev/login")
		if err != nil {
			http.Error(w, "bad frontend origin", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, target, http.StatusFound)
	})
}

var browserPage = template.Must(template.New("devlogin").Parse(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Clutter dev sign-in</title></head>
<body style="font-family: system-ui, sans-serif; background: #000; color: #ddd; padding: 2rem">
<p>Signed in as {{.Email}} (devserver only). Opening the dashboard.</p>
<script>
  // Same storage the frontend writes after POST /log-in (src/lib/features/user/api.ts).
  window.localStorage.setItem("google_data", JSON.stringify({{.Login}}));
  window.location.replace("/dashboard");
</script>
</body>
</html>
`))

// BrowserHandler serves /dev/login on the frontend's origin. It sets the
// clutter_token cookie the route guard checks and the google_data
// localStorage entry the data hooks read the token from, then opens
// /dashboard.
func (i *Issuer) BrowserHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "use GET", http.StatusMethodNotAllowed)
			return
		}
		resp, err := i.Login(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// The frontend writes this cookie from JavaScript with
		// encodeURIComponent; a JWT needs no encoding.
		w.Header().Add("Set-Cookie", fmt.Sprintf("%s=%s; Path=/; SameSite=Lax; Max-Age=%d", cookieName, url.QueryEscape(resp.Token), cookieMaxAge))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = browserPage.Execute(w, map[string]any{"Email": resp.UserData.Email, "Login": resp})
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
