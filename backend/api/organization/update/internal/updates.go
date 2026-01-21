package internal

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func UpdateOrgName(tx pgx.Tx, ctx context.Context, newOrgName string, orgId string) error {

	queryUpdateOrgName := `
		UPDATE organizations
		SET name = $1
		WHERE id = $2
	`

	if _, err := tx.Exec(ctx, queryUpdateOrgName, newOrgName, orgId); err != nil {
		return err
	}

	return nil

}

func UpdateOrgDescription(tx pgx.Tx, ctx context.Context, newDescription string, orgId string) error {

	queryUpdateOrgDescription := `
		UPDATE organizations
		SET description = $1
		WHERE id = $2
	`

	if _, err := tx.Exec(ctx, queryUpdateOrgDescription, newDescription, orgId); err != nil {
		return err
	}

	return nil
}
