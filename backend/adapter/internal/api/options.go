package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/store"
)

// optionsTimeout bounds one Options call: a token and three list requests.
const optionsTimeout = 60 * time.Second

// Caps on what the options route passes on from a cloud.
const (
	maxChoices     = 500
	maxChoiceLabel = 128
)

// connectionOptions answers the values the connection's cloud offers for the
// connector's deployment settings (connectors.OptionLister), so the deploy
// panel can show lists instead of text boxes. The credentials are read for
// this one call and never reach the response.
func (a *API) connectionOptions(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := a.caller(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "connectionID")
	if !ok {
		return
	}
	row, err := a.Store.GetConnection(r.Context(), orgID, id)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		a.internal(w, r, "get connection", err)
		return
	}
	c, found := a.Registry.Get(row.ConnectorID)
	if !found {
		writeError(w, http.StatusUnprocessableEntity, "unknown_connector", "This cloud is not available.")
		return
	}
	lister, ok := c.(connectors.OptionLister)
	if !ok {
		writeError(w, http.StatusNotImplemented, "no_options", "This cloud does not list choices for its settings. Type the values instead.")
		return
	}
	secrets, err := a.Credentials.Get(r.Context(), row.SecretRef)
	if err != nil {
		a.internal(w, r, "read connection credentials", err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), optionsTimeout)
	defer cancel()
	opts, err := lister.Options(ctx, connectors.Connection{ID: row.ID, ConnectorID: c.ID(), Region: row.Region, Credentials: secrets})
	if err != nil {
		var ue *connectors.UserError
		if errors.As(err, &ue) {
			a.Logger.Info("listing setting choices refused", "connector", c.ID(), "org", orgID)
			writeError(w, http.StatusBadGateway, "options_failed", ue.Message)
			return
		}
		a.internal(w, r, "list setting choices", err)
		return
	}
	writeJSON(w, http.StatusOK, cleanOptions(c.Manifest(), opts))
}

// cleanOptions keeps only the settings the manifest declares and the values
// CheckValue accepts for each, without repeats, with a plain one-line label.
// A suggestion is kept only when it is one of the choices kept for its
// setting, so it passes the same checks.
func cleanOptions(m connectors.Manifest, in connectors.Options) connectors.Options {
	out := connectors.Options{Settings: map[string][]connectors.Choice{}}
	suggest := func(name, value string) {
		if out.Suggested == nil {
			out.Suggested = map[string]string{}
		}
		out.Suggested[name] = value
	}
	for _, f := range m.Settings {
		choices, ok := in.Settings[f.Name]
		if !ok {
			continue
		}
		kept := []connectors.Choice{}
		seen := map[string]bool{}
		for _, ch := range choices {
			if len(kept) >= maxChoices {
				break
			}
			if ch.Value == "" || seen[ch.Value] || connectors.CheckValue(f, ch.Value) != nil {
				continue
			}
			seen[ch.Value] = true
			label, ok := cleanLabel(ch.Label)
			if !ok {
				label = ch.Value
			}
			kept = append(kept, connectors.Choice{Value: ch.Value, Label: label})
		}
		out.Settings[f.Name] = kept
		if s, ok := in.Suggested[f.Name]; ok && seen[s] {
			suggest(f.Name, s)
		}
	}
	return out
}

// cleanLabel trims a label, refuses one with control characters or invalid
// UTF-8, and cuts it to maxChoiceLabel characters.
func cleanLabel(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || !utf8.ValidString(s) {
		return "", false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return "", false
		}
	}
	if r := []rune(s); len(r) > maxChoiceLabel {
		s = string(r[:maxChoiceLabel])
	}
	return s, true
}
