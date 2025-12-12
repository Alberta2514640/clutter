package generic

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5"
)

func PsqlConnect() (*pgx.Conn, error) {

	conn, err := pgx.Connect(context.Background(), os.Getenv("PSQL_CONNECTION_STRING"))
	if err != nil {
		return nil, err
	}

	return conn, err

}
