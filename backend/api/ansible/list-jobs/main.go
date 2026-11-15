package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/ansible/shared/jobsutils"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

// newStatusStore opens the bucket the runner writes job status files to.
// Tests replace it with a fake.
var newStatusStore = generic.StatusStoreFromEnv

// Bounds the S3 reads one list can cause.
const maxJobsPerReconcile = 10

// jobRow is one listed job as scanned, before it is answered.
type jobRow struct {
	state              jobsutils.JobState
	jobType            string
	createdAt          time.Time
	extraVars          []byte
	targetInstanceIDs  []byte
	taskArn            *string
	terraformDirectory *string
	roleArn            *string
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1. Extract authenticated user from authorizer context
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		log.Printf("ERROR: unauthorized request: %v", err)
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
		})
	}

	// Get user ID for filtering
	userID := userData.Id

	// Validate optional status filter
	statusFilter := request.QueryStringParameters["status"]
	if statusFilter != "" && !generic.AllowedJobStatuses[statusFilter] {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid status filter. Allowed: QUEUED, STARTING, RUNNING, COMPLETED, FAILED",
		})
	}

	// Validate optional job_type filter
	jobTypeFilter := request.QueryStringParameters["job_type"]
	if jobTypeFilter != "" && !jobsutils.AllowedJobTypes[jobTypeFilter] {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid job_type filter. Allowed: ansible, terraform",
		})
	}

	// Jobs are listed for one organization the caller belongs to
	orgParam := request.QueryStringParameters["orgId"]
	if !generic.IsValidUuid(orgParam) {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "orgId must be a valid UUID",
		})
	}

	// Connect to PostgreSQL using shared helper
	conn, err := connect()
	if err != nil {
		log.Printf("ERROR: failed to connect to database: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	orgID, err := generic.CheckOwnershipPSQL(ctx, conn, userID, orgParam, "", "")
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	// Build query dynamically based on filters. A job's organization is the
	// first segment of its playbook key, or of its Terraform directory when it
	// has no playbook key, as ansible/get-job reads it.
	query := `
		SELECT id, COALESCE(job_type, 'ansible') as job_type, status, created_at, updated_at,
		       extra_vars, task_arn, error_message,
		       target_instance_ids, playbook_s3_key,
		       terraform_directory, role_arn
		FROM jobs
		WHERE created_by = $1
		  AND split_part(COALESCE(NULLIF(playbook_s3_key, ''), terraform_directory), '/', 1) = $2`
	args := []interface{}{userID, orgID}
	argNum := 3

	if statusFilter != "" {
		query += ` AND status = $` + jobsutils.Itoa(argNum)
		args = append(args, statusFilter)
		argNum++
	}
	if jobTypeFilter != "" {
		query += ` AND COALESCE(job_type, 'ansible') = $` + jobsutils.Itoa(argNum)
		args = append(args, jobTypeFilter)
		argNum++
	}

	query += ` ORDER BY created_at DESC LIMIT 50`

	var rows pgx.Rows
	rows, err = conn.Query(ctx, query, args...)

	if err != nil {
		log.Printf("ERROR: failed to query jobs: %v", err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to query jobs",
		})
	}
	defer rows.Close()

	var scanned []jobRow
	for rows.Next() {
		var row jobRow
		err := rows.Scan(
			&row.state.ID, &row.jobType, &row.state.Status, &row.createdAt, &row.state.UpdatedAt,
			&row.extraVars, &row.taskArn, &row.state.ErrorMessage,
			&row.targetInstanceIDs, &row.state.PlaybookS3Key,
			&row.terraformDirectory, &row.roleArn,
		)
		if err != nil {
			log.Printf("ERROR: failed to scan row: %v", err)
			rows.Close()
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"message": "failed to process job list",
			})
		}
		scanned = append(scanned, row)
	}

	if rows.Err() != nil {
		log.Printf("ERROR: rows iteration error: %v", rows.Err())
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to process job list",
		})
	}
	// The connection cannot write while the rows are open.
	rows.Close()

	// The runner has no database access; the newest unfinished jobs take
	// their status from the status file beside their logs. Every job here is
	// the caller's own in the checked organization (the query above).
	var store *generic.StatusStore
	reconciled := 0
	for i := range scanned {
		if generic.IsJobFinished(scanned[i].state.Status) || reconciled == maxJobsPerReconcile {
			continue
		}
		if store == nil {
			if store = newStatusStore(ctx); store == nil {
				break
			}
		}
		jobsutils.Reconcile(ctx, conn, store, &scanned[i].state)
		reconciled++
	}

	jobs := []map[string]interface{}{}
	for _, row := range scanned {
		// A job that finished just now no longer matches a status filter.
		if statusFilter != "" && row.state.Status != statusFilter {
			continue
		}
		jobs = append(jobs, generic.BuildJobResponse(
			row.state.ID, row.jobType, row.state.Status,
			row.createdAt, row.state.UpdatedAt,
			row.extraVars, row.taskArn, row.state.ErrorMessage,
			row.targetInstanceIDs, row.state.PlaybookS3Key,
			row.terraformDirectory, row.roleArn,
		))
	}

	return generic.Response(http.StatusOK, generic.Json{
		"data":  jobs,
		"count": len(jobs),
	})
}
