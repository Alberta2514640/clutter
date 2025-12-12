package generic

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func PsqlConnect() (*pgx.Conn, error) {

	conn, err := pgx.Connect(context.Background(), os.Getenv("PSQL_CONNECTION_STRING"))
	if err != nil {
		return nil, err
	}

	return conn, err

}

// Table Violations

// When a table enforces that some parameter can not be duplicate
// within the table this function should be used to check the error
//
// Ex. The organization_members table enforces that no user may have
// a duplicate organization name. In the scenario that there is a duplicate,
// this Postgres error code is displayed
//
// (Eg. ERROR: duplicate key value violates unique constraint \"unique_org_name_per_user\" (SQLSTATE ***23505***))
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
