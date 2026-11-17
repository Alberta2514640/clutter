package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/mock"
)

// corsAPI is an API with no database: the catalog route needs none.
func corsAPI(t *testing.T, origins []string) http.Handler {
	t.Helper()
	reg := connectors.NewRegistry()
	if err := reg.Register(mock.New()); err != nil {
		t.Fatal(err)
	}
	a := &API{
		Registry:       reg,
		Verifier:       fakeVerifier{"good-token": "6f1c1f43-6ed4-4b8e-9d55-2b1a8f2f6d10"},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		AllowedOrigins: origins,
	}
	return a.Handler()
}

func serve(h http.Handler, method, origin, token string, preflight bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/v1/catalog", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	if preflight {
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCORSAllowedOrigin(t *testing.T) {
	h := corsAPI(t, []string{"http://localhost:3000", "http://127.0.0.1:3000"})

	pre := serve(h, http.MethodOptions, "http://127.0.0.1:3000", "", true)
	if pre.Code != http.StatusNoContent {
		t.Fatalf("preflight: %d %s", pre.Code, pre.Body)
	}
	for name, want := range map[string]string{
		"Access-Control-Allow-Origin":  "http://127.0.0.1:3000",
		"Access-Control-Allow-Headers": "Authorization, Content-Type",
		"Access-Control-Allow-Methods": "GET, POST, PUT, PATCH, DELETE",
	} {
		if got := pre.Header().Get(name); got != want {
			t.Errorf("preflight %s = %q, want %q", name, got, want)
		}
	}

	got := serve(h, http.MethodGet, "http://localhost:3000", "good-token", false)
	if got.Code != http.StatusOK || got.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("GET: %d, allow-origin %q", got.Code, got.Header().Get("Access-Control-Allow-Origin"))
	}

	// CORS never stands in for sign-in.
	if anon := serve(h, http.MethodGet, "http://localhost:3000", "", false); anon.Code != http.StatusUnauthorized {
		t.Fatalf("GET without a token: %d", anon.Code)
	}
}

func TestCORSRefusesOtherOrigins(t *testing.T) {
	h := corsAPI(t, []string{"http://localhost:3000"})
	for _, origin := range []string{"http://evil.example", "http://localhost:3001", "http://localhost:3000.evil.example", "null"} {
		pre := serve(h, http.MethodOptions, origin, "", true)
		if pre.Code == http.StatusNoContent || pre.Header().Get("Access-Control-Allow-Origin") != "" || pre.Header().Get("Access-Control-Allow-Headers") != "" {
			t.Errorf("preflight from %s: %d, headers %v", origin, pre.Code, pre.Header())
		}
		get := serve(h, http.MethodGet, origin, "good-token", false)
		if get.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("GET from %s carries allow-origin", origin)
		}
	}
}

func TestCORSUnsetSendsNoHeaders(t *testing.T) {
	h := corsAPI(t, nil)
	pre := serve(h, http.MethodOptions, "http://localhost:3000", "", true)
	if pre.Code == http.StatusNoContent {
		t.Fatal("a preflight was answered with CORS unset")
	}
	get := serve(h, http.MethodGet, "http://localhost:3000", "good-token", false)
	if get.Code != http.StatusOK {
		t.Fatalf("GET: %d", get.Code)
	}
	for _, rec := range []*httptest.ResponseRecorder{pre, get} {
		for name := range rec.Header() {
			if strings.HasPrefix(name, "Access-Control-") || name == "Vary" {
				t.Errorf("header %s sent with CORS unset", name)
			}
		}
	}
}
