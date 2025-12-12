package generic

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jackc/pgx/v5"
)

func PsqlConnect() (*pgx.Conn, error) {

	conn, err := pgx.Connect(context.Background(), os.Getenv("PSQL_CONNECTION_STRING"))
	if err != nil {
		return nil, err
	}

	return conn, err

}

func GetUserIDFromRequest(req events.APIGatewayProxyRequest) (string, error) {
	if req.RequestContext.Authorizer != nil {
		if v, ok := req.RequestContext.Authorizer["userId"]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				return s, nil
			}
		}
		if v, ok := req.RequestContext.Authorizer["sub"]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				return s, nil
			}
		}
		if v, ok := req.RequestContext.Authorizer["email"]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				return s, nil
			}
		}
	}

	for k, v := range req.Headers {
		if strings.ToLower(k) == "x-user-id" && v != "" {
			return v, nil
		}
	}

	return "", errors.New("missing user identity in request context")
}
