package api

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/codestore"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/store"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/translate"
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// Function code: one zip per function block of a diagram, uploaded with one
// PUT to B in both places (locally and in the cloud), so B checks every byte
// (size, zip, paths) and hashes it before anything is stored. A presigned
// upload straight to S3 would store the object first and check it after.

// codeView is one uploaded zip as the editor shows it. It never carries the
// store's key.
type codeView struct {
	NodeID     string    `json:"node_id"`
	FileName   string    `json:"file_name"`
	Size       int64     `json:"size"`
	SHA256     string    `json:"sha256"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func codeOut(c store.FunctionCode) codeView {
	return codeView{NodeID: c.NodeID, FileName: c.FileName, Size: c.Size, SHA256: c.SHA256, UploadedAt: c.UploadedAt}
}

// uploadDeadline bounds reading one upload: 50 MB at about 1 Mbit/s. The
// server's own ReadTimeout is far shorter, for every other route.
const uploadDeadline = 8 * time.Minute

// zipTypes are the Content-Types a browser sends with a .zip file.
var zipTypes = map[string]bool{"application/zip": true, "application/x-zip-compressed": true, "application/octet-stream": true}

// codeDiagram signs the caller into the organization in the path and proves
// the diagram in the path belongs to one of its projects.
func (a *API) codeDiagram(w http.ResponseWriter, r *http.Request) (userID, orgID, diagramID string, ok bool) {
	userID, orgID, ok = a.caller(w, r)
	if !ok {
		return "", "", "", false
	}
	ids, err := generic.CheckOwnedIDsPSQL(r.Context(), a.Store.Pool, userID, orgID, "", r.PathValue("diagramID"))
	if err != nil {
		a.authError(w, r, err)
		return "", "", "", false
	}
	return userID, orgID, ids.DiagramID, true
}

func (a *API) listCode(w http.ResponseWriter, r *http.Request) {
	_, orgID, diagramID, ok := a.codeDiagram(w, r)
	if !ok {
		return
	}
	rows, err := a.Store.ListFunctionCode(r.Context(), orgID, diagramID)
	if err != nil {
		a.internal(w, r, "list function code", err)
		return
	}
	out := make([]codeView, 0, len(rows))
	for _, c := range rows {
		out = append(out, codeOut(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// putCode stores the request body as a function block's code: a .zip of at
// most codestore.MaxSize, named by the file_name query value.
func (a *API) putCode(w http.ResponseWriter, r *http.Request) {
	if a.Code == nil {
		writeError(w, http.StatusNotImplemented, "no_code_store", "This Clutter server does not keep function code.")
		return
	}
	userID, orgID, diagramID, ok := a.codeDiagram(w, r)
	if !ok {
		return
	}
	nodeID := r.PathValue("nodeID")
	if !translate.NodeIDPattern.MatchString(nodeID) {
		notFound(w)
		return
	}
	fileName := r.URL.Query().Get("file_name")
	if !codestore.ValidFileName(fileName) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_file_name", "Choose a .zip file. Its name must end in .zip and be at most 255 characters.")
		return
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || !zipTypes[mt] {
		writeError(w, http.StatusUnsupportedMediaType, "not_zip", "Upload the code as one .zip file.")
		return
	}
	if r.ContentLength > codestore.MaxSize {
		writeError(w, http.StatusRequestEntityTooLarge, "too_large", tooLarge)
		return
	}
	// The block must be a function in the diagram as saved: code for any
	// other block, or for a block not saved yet, would never be used.
	doc, _, err := a.Store.DiagramDocument(r.Context(), orgID, diagramID)
	if err != nil {
		a.internal(w, r, "read diagram", err)
		return
	}
	g, err := translate.Translate(doc)
	if err != nil {
		a.translateError(w, r, err)
		return
	}
	b, found := g.Blocks[nodeID]
	if !found {
		writeError(w, http.StatusUnprocessableEntity, "unknown_block", "This block is not in the saved diagram. Save the diagram, then upload the code.")
		return
	}
	if b.Kind != connectors.KindFunction {
		writeError(w, http.StatusUnprocessableEntity, "not_a_function", "Only a function block takes uploaded code.")
		return
	}

	rc := http.NewResponseController(w)
	deadline := time.Now().Add(uploadDeadline)
	if err := rc.SetReadDeadline(deadline); err != nil && !errors.Is(err, http.ErrNotSupported) {
		a.internal(w, r, "extend the upload's read deadline", err)
		return
	}
	if err := rc.SetWriteDeadline(deadline.Add(time.Minute)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		a.internal(w, r, "extend the upload's write deadline", err)
		return
	}
	rec, err := codestore.Receive(a.CodeTempDir, http.MaxBytesReader(w, r.Body, codestore.MaxSize), codestore.MaxSize)
	var refused *codestore.RefusedError
	switch {
	case errors.Is(err, codestore.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "too_large", tooLarge)
		return
	case errors.As(err, &refused):
		writeError(w, http.StatusUnprocessableEntity, "invalid_code", refused.Message)
		return
	case err != nil:
		a.internal(w, r, "receive function code", err)
		return
	}
	defer func() {
		if err := rec.Close(); err != nil {
			a.Logger.Warn("could not remove an upload's temporary file", "err", err)
		}
	}()

	ctx := context.WithoutCancel(r.Context())
	key := codestore.Key(orgID, diagramID, nodeID, rec.SHA256)
	put := func(ctx context.Context) error {
		if _, err := rec.File.Seek(0, io.SeekStart); err != nil {
			return err
		}
		return a.Code.Put(ctx, key, rec.File, rec.Size)
	}
	// The object goes to the store before the node's lock is taken, so the
	// lock is never held through the upload. Under the lock, ensure puts it
	// again when an overlapping upload or the sweep deleted it in between
	// (store.PutFunctionCode says why that keeps the two consistent).
	if err := put(ctx); err != nil {
		a.internal(w, r, "store function code", err)
		return
	}
	var ensureErr error
	row, previous, err := a.Store.PutFunctionCode(ctx, store.FunctionCode{
		DiagramID: diagramID, NodeID: nodeID, OrganizationID: orgID, ObjectKey: key,
		FileName: fileName, Size: rec.Size, SHA256: rec.SHA256, UploadedBy: userID,
	}, func(ctx context.Context) error {
		ok, err := a.Code.Exists(ctx, key)
		if err == nil && !ok {
			err = put(ctx)
		}
		ensureErr = err
		return err
	})
	if err != nil {
		// The same zip may already be this node's recorded code, so the
		// object goes only when no record names it.
		a.removeUnusedCode(ctx, diagramID, nodeID, key, "unrecorded")
		switch {
		case errors.Is(err, store.ErrNotFound):
			notFound(w)
		case ensureErr != nil:
			a.internal(w, r, "store function code", err)
		default:
			a.internal(w, r, "record function code", err)
		}
		return
	}
	if previous != "" {
		a.removeUnusedCode(ctx, diagramID, nodeID, previous, "replaced")
	}
	writeJSON(w, http.StatusOK, codeOut(row))
}

// removeUnusedCode deletes the object at key unless a record names it. A
// failure leaves an unused object, which the service log names and the
// sweep (codestore.Sweeper) removes later.
func (a *API) removeUnusedCode(ctx context.Context, diagramID, nodeID, key, what string) {
	_, err := a.Store.DeleteUnreferencedCode(ctx, diagramID, nodeID, key, func(ctx context.Context) error {
		return a.Code.Delete(ctx, key)
	})
	if err != nil {
		a.Logger.Error("could not remove "+what+" function code", "diagram_id", diagramID, "node_id", nodeID, "err", err)
	}
}

const tooLarge = "The .zip is larger than 50 MB, the most a function can take as an upload."

// deleteCode removes a function block's code, so the block deploys with
// placeholder code again.
func (a *API) deleteCode(w http.ResponseWriter, r *http.Request) {
	if a.Code == nil {
		writeError(w, http.StatusNotImplemented, "no_code_store", "This Clutter server does not keep function code.")
		return
	}
	_, orgID, diagramID, ok := a.codeDiagram(w, r)
	if !ok {
		return
	}
	nodeID := r.PathValue("nodeID")
	if !translate.NodeIDPattern.MatchString(nodeID) {
		notFound(w)
		return
	}
	row, err := a.Store.DeleteFunctionCode(r.Context(), orgID, diagramID, nodeID)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		a.internal(w, r, "delete function code", err)
		return
	}
	// The record is gone, so no run stages the object again. An upload of the
	// same zip that recorded it since keeps it.
	a.removeUnusedCode(context.WithoutCancel(r.Context()), diagramID, nodeID, row.ObjectKey, "deleted")
	w.WriteHeader(http.StatusNoContent)
}

// attachCode puts each function block's uploaded code on the graph, from
// B's own records for the organization's diagram.
func (a *API) attachCode(ctx context.Context, orgID, diagramID string, g *connectors.Graph) error {
	rows, err := a.Store.ListFunctionCode(ctx, orgID, diagramID)
	if err != nil {
		return err
	}
	for _, c := range rows {
		b, ok := g.Blocks[c.NodeID]
		if !ok || b.Kind != connectors.KindFunction {
			continue
		}
		b.Code = &connectors.FunctionCode{Key: c.ObjectKey, SHA256: c.SHA256, Size: c.Size}
		g.Blocks[c.NodeID] = b
	}
	return nil
}
