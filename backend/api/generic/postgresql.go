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

// Querier is the part of a Postgres connection the ownership checks need.
// *pgx.Conn satisfies it, and so does pgxmock's connection in tests.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Conn is the part of a Postgres connection a handler uses. Handlers that
// open their connection through a replaceable Conn can be tested with pgxmock.
type Conn interface {
	Querier
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Close(ctx context.Context) error
}

// PsqlConnectConn is PsqlConnect behind the Conn interface. It never returns a
// non-nil Conn with an error.
func PsqlConnectConn() (Conn, error) {
	conn, err := PsqlConnect()
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
