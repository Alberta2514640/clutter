package generic

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jackc/pgx/v5"
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

// GetUserIDFromRequest extracts the user's UUID from the API Gateway request.
func GetUserIDFromRequest(request events.APIGatewayProxyRequest) (string, error) {
	// Check uuid from authorizer context
	if request.RequestContext.Authorizer != nil {
		if v, ok := request.RequestContext.Authorizer["uuid"]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				return s, nil
			}
		}
	}

	return "", errors.New("missing user identity in request context")
}
