package internal

import (
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

type SupabaseWebhookPayload struct {
	Type   string                 `json:"type"`   // INSERT, UPDATE, DELETE
	Table  string                 `json:"table"`  // table name
	Schema string                 `json:"schema"` // usually "public"
	Record generic.DiagramRecord  `json:"record"` // the new/updated record
	OldRecord generic.DiagramRecord  `json:"old_record,omitempty"` // for UPDATE
}