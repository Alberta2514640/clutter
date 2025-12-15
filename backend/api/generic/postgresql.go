package generic

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PsqlConnect establishes a connection to the PostgreSQL database using the
// connection string from the PSQL_CONNECTION_STRING environment variable.
func PsqlConnect() (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("PSQL_CONNECTION_STRING"))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// IsUniqueViolation checks if the error is a PostgreSQL unique constraint violation
// PostgreSQL error code 23505 indicates a unique_violation
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
