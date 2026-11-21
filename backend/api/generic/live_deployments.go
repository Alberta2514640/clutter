package generic

import (
	"errors"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jackc/pgx/v5/pgconn"
)

// restrictViolation is the SQLSTATE the trigger refuse_live_deployment_delete
// (backend/storage-sql/adapter-tables.sql) raises when a delete, direct or by
// cascade, would remove a deployment that may still hold resources in the
// user's cloud. No foreign key in the schema is ON DELETE RESTRICT, so the
// trigger is the only source of this code.
const restrictViolation = "23001"

// IsLiveDeploymentDelete reports whether err is that trigger's refusal.
func IsLiveDeploymentDelete(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == restrictViolation
}

// LiveDeploymentsResponse is the answer to a delete of a diagram, project or
// organization that still has live deployments.
func LiveDeploymentsResponse() (events.APIGatewayProxyResponse, error) {
	return Response(http.StatusConflict, Json{"message": "this has live deployments; destroy them first"})
}
