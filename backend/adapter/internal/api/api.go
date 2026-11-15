// Package api is B API's /v1 routes. Every route signs the caller in, proves
// they are a member of the organization in the path, and reaches a connection,
// deployment or run only through a query scoped to that organization.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/auth"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/codestore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/credentials"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/jobs"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/logstore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/store"
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// Enqueuer is the part of the job runner the API uses.
type Enqueuer interface {
	Enqueue(j jobs.Job) error
}

// API holds what the routes need.
type API struct {
	Store       *store.Store
	Registry    *connectors.Registry
	Credentials credentials.Store
	Jobs        Enqueuer
	Logs        logstore.Store
	Verifier    auth.TokenVerifier
	Logger      *slog.Logger
	// VerifyTimeout bounds one VerifyConnection call. Zero means 30 seconds.
	VerifyTimeout time.Duration
	// AllowedOrigins are the browser origins CORS answers, matched exactly.
	// Empty means no CORS headers at all.
	AllowedOrigins []string
	// MaxRunsPerOrg caps one organization's queued and running runs. Zero
	// means DefaultMaxRunsPerOrg.
	MaxRunsPerOrg int
	// Code keeps uploaded function code, and CodeTempDir holds an upload
	// while it is checked. Without Code the code routes answer 501.
	Code        codestore.Objects
	CodeTempDir string
}

// DefaultMaxRunsPerOrg is the per-organization run cap when none is set.
const DefaultMaxRunsPerOrg = 3

func (a *API) maxRunsPerOrg() int {
	if a.MaxRunsPerOrg > 0 {
		return a.MaxRunsPerOrg
	}
	return DefaultMaxRunsPerOrg
}

// maxBody caps a request body.
const maxBody = 1 << 20

// Handler returns the routes.
func (a *API) Handler() http.Handler {
	v1 := http.NewServeMux()
	v1.HandleFunc("GET /v1/catalog", a.catalog)
	v1.HandleFunc("GET /v1/examples", a.examples)
	v1.HandleFunc("GET /v1/organizations/{orgID}/connectors/{connectorID}/linked-accounts", a.listLinkedAccounts)
	v1.HandleFunc("GET /v1/organizations/{orgID}/connections", a.listConnections)
	v1.HandleFunc("POST /v1/organizations/{orgID}/connections", a.createConnection)
	v1.HandleFunc("POST /v1/organizations/{orgID}/connections/{connectionID}/verify", a.verifyConnection)
	v1.HandleFunc("GET /v1/organizations/{orgID}/connections/{connectionID}/options", a.connectionOptions)
	v1.HandleFunc("DELETE /v1/organizations/{orgID}/connections/{connectionID}", a.deleteConnection)
	v1.HandleFunc("POST /v1/organizations/{orgID}/diagrams/{diagramID}/deployments", a.createDeployment)
	v1.HandleFunc("GET /v1/organizations/{orgID}/diagrams/{diagramID}/deployments", a.listDiagramDeployments)
	v1.HandleFunc("GET /v1/organizations/{orgID}/diagrams/{diagramID}/code", a.listCode)
	v1.HandleFunc("PUT /v1/organizations/{orgID}/diagrams/{diagramID}/code/{nodeID}", a.putCode)
	v1.HandleFunc("DELETE /v1/organizations/{orgID}/diagrams/{diagramID}/code/{nodeID}", a.deleteCode)
	v1.HandleFunc("PATCH /v1/organizations/{orgID}/deployments/{deploymentID}", a.updateDeployment)
	v1.HandleFunc("DELETE /v1/organizations/{orgID}/deployments/{deploymentID}", a.deleteDeployment)
	v1.HandleFunc("GET /v1/organizations/{orgID}/deployments/{deploymentID}/compatibility", a.compatibility)
	v1.HandleFunc("GET /v1/organizations/{orgID}/deployments/{deploymentID}/preview", a.preview)
	v1.HandleFunc("POST /v1/organizations/{orgID}/deployments/{deploymentID}/runs", a.createRun)
	v1.HandleFunc("GET /v1/organizations/{orgID}/runs/{runID}", a.getRun)
	v1.HandleFunc("GET /v1/organizations/{orgID}/runs/{runID}/logs", a.runLogs)

	signedIn := auth.Middleware(a.Verifier, a.Logger, func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
	})(v1)

	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", a.healthz)
	root.Handle("/v1/", signedIn)
	root.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "Not found.")
	})
	return a.recoverer(a.cors(root))
}

// cors answers browser calls from AllowedOrigins only. A preflight from an
// allowed origin is answered here, before sign-in, because a browser sends it
// without the Authorization header. Any other request passes on unchanged,
// with no CORS headers unless its Origin is allowed.
func (a *API) cors(next http.Handler) http.Handler {
	if len(a.AllowedOrigins) == 0 {
		return next
	}
	allowed := make(map[string]bool, len(a.AllowedOrigins))
	for _, o := range a.AllowedOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Add("Vary", "Origin")
		origin := r.Header.Get("Origin")
		if origin == "" || !allowed[origin] {
			next.ServeHTTP(w, r)
			return
		}
		h.Set("Access-Control-Allow-Origin", origin)
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			h.Add("Vary", "Access-Control-Request-Method")
			h.Add("Vary", "Access-Control-Request-Headers")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			h.Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				a.Logger.Error("handler panicked", "path", r.URL.Path, "panic", fmt.Sprint(p))
				writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (a *API) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.Store.Pool.Ping(ctx); err != nil {
		a.Logger.Error("health check: database unreachable", "err", err)
		writeError(w, http.StatusServiceUnavailable, "unavailable", "The service is not ready.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// errorBody is the one error envelope.
type errorBody struct {
	Error struct {
		Code     string               `json:"code"`
		Message  string               `json:"message"`
		Problems []connectors.Problem `json:"problems,omitempty"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errcheck // the client went away
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeProblems(w, status, code, message, nil)
}

func writeProblems(w http.ResponseWriter, status int, code, message string, problems []connectors.Problem) {
	var b errorBody
	b.Error.Code, b.Error.Message, b.Error.Problems = code, message, problems
	writeJSON(w, status, b)
}

// internal logs err and answers with a generic 500.
func (a *API) internal(w http.ResponseWriter, r *http.Request, what string, err error) {
	a.Logger.Error(what, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
}

func notFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "not_found", "Not found.")
}

// authError answers an ownership check's error. Its own message, which can
// name ids, goes to the log only.
func (a *API) authError(w http.ResponseWriter, r *http.Request, err error) {
	var ae *generic.AuthorizationError
	if errors.As(err, &ae) {
		a.Logger.Info("ownership check refused", "path", r.URL.Path, "status", ae.StatusCode, "reason", ae.Message)
		if ae.StatusCode == http.StatusForbidden {
			writeError(w, http.StatusForbidden, "forbidden", "You do not have access to this organization.")
			return
		}
		notFound(w)
		return
	}
	a.internal(w, r, "ownership check failed", err)
}

// caller signs the request's caller into the organization in its path and
// returns the user id and the organization id in canonical form.
func (a *API) caller(w http.ResponseWriter, r *http.Request) (userID, orgID string, ok bool) {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
		return "", "", false
	}
	orgID, err := generic.CheckOwnershipPSQL(r.Context(), a.Store.Pool, id.UserID, r.PathValue("orgID"), "", "")
	if err != nil {
		a.authError(w, r, err)
		return "", "", false
	}
	return id.UserID, orgID, true
}

// pathID returns a path id in canonical form, or answers 404.
func pathID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id, ok := generic.CanonicalUuid(r.PathValue(name))
	if !ok {
		notFound(w)
	}
	return id, ok
}

// decode reads one JSON object with no unknown fields.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "The request body is not valid JSON for this route.")
		return false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "bad_request", "The request body holds more than one JSON value.")
		return false
	}
	return true
}

func cleanName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || len([]rune(s)) > 64 {
		return "", false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return "", false
		}
	}
	return s, true
}
