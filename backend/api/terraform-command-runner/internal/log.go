package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/jackc/pgx/v5/pgconn"
)

func InsertIntoLogTable(conn generic.Conn, ctx context.Context, diagramId string, command string) (string, error) {

	const maxRetries = 5
	var commandId string
	var completed bool = false

	for range maxRetries {

		commandId = generic.RandomID(8)

		_, err := conn.Exec(
			ctx,
			`
			INSERT INTO public.diagram_deployment_logs (
				diagram_id,
				command_id,
				command,
				status
			)
			VALUES ($1, $2, $3, $4);
			`,
			diagramId,
			commandId,
			command,
			"RUNNING",
		)
		// If we succeeded break from the loop
		if err == nil {
			completed = true
			break
		}

		// Check if it's a unique constraint violation in which case continue retry
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			continue
		}

		// If it's not a unique constraint violation then return
		return "", err

	}

	if completed == false {
		return "", fmt.Errorf("failed to insert into log table due to repeated unique constraint violations on command id")
	} else {
		return commandId, nil
	}

}

// MarkRunNotStarted marks a run FAILED, with no duration, when its task never
// started. The caller already answers with an error, so a failure here is
// only logged.
func MarkRunNotStarted(conn generic.Conn, ctx context.Context, diagramId string, commandId string) {
	_, err := conn.Exec(
		ctx,
		`
		UPDATE public.diagram_deployment_logs
		SET status = 'FAILED', duration_seconds = 0
		WHERE diagram_id = $1 AND command_id = $2 AND status = 'RUNNING';
		`,
		diagramId,
		commandId,
	)
	if err != nil {
		log.Printf("runner: mark the run that did not start as FAILED: %v", err)
	}
}
