package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator/writer"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

// What the user reads when the generator cannot start. The cause stays in the
// log.
const startFailureMessage = "The Terraform generator could not start. Save the diagram again to retry."

// requestBody names the diagram to generate. The diagram itself is read from
// the database, never taken from the request.
type requestBody struct {
	ProjectID string `json:"projectId"`
	DiagramID string `json:"diagramId"`
}

// diagramQuery reads the row the generator needs. The project id in the WHERE
// clause repeats the ownership check, so a row moved between the two queries
// is not generated under the wrong project.
const diagramQuery = `
	SELECT id, project_id, data, latest_update_at
	FROM diagrams
	WHERE id = $1 AND project_id = $2`

// connect opens the database. Tests replace it with a mock.
var connect = generic.PsqlConnectConn

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context holds the caller, taken from the JWT
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil || userData.Id == "" {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"message": "unauthorized: missing user identity",
		})
	}

	var body requestBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "invalid JSON body",
		})
	}
	if !generic.IsValidUuid(body.ProjectID) || !generic.IsValidUuid(body.DiagramID) {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "valid projectId and diagramId are required",
		})
	}

	conn, err := connect()
	if err != nil {
		log.Printf("diagram %s: connect to the database: %v", body.DiagramID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to database",
		})
	}
	defer conn.Close(ctx)

	orgID, err := generic.CheckOwnershipPSQL(ctx, conn, userData.Id, "", body.ProjectID, body.DiagramID)
	if err != nil {
		return generic.AuthorizationResponse(err)
	}

	record, err := loadDiagram(ctx, conn, body.DiagramID, body.ProjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return generic.Response(http.StatusNotFound, generic.Json{
			"message": "Diagram not found",
		})
	}
	if err != nil {
		log.Printf("diagram %s: read the diagram: %v", body.DiagramID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to read the diagram",
		})
	}

	// A building-block diagram is B's: it deploys through its deployments, and
	// A generates nothing for it (ADR-014). Nothing is written, so no earlier
	// files or status are touched.
	if record.Data.Engine == generic.EngineBlocks {
		return generic.Response(http.StatusOK, generic.Json{
			"message":    "Nothing was generated: this is a building-block diagram, which deploys through its deployments.",
			"diagram_id": record.ID,
			"generated":  false,
		})
	}

	// Clean the nodes in the record (Remove irrelevant fields)
	record.Data.Nodes = internal.SanitizeNodes(record.Data.Nodes)

	// Initialize terraform generator with template bucket
	templateBucket := os.Getenv("TEMPLATE_BUCKET_NAME")
	gen, err := generator.NewTerraformGenerator(ctx, templateBucket)
	if err != nil {
		log.Printf("diagram %s: %v", record.ID, err)
		recordFailure(ctx, orgID, record, startFailureMessage)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": startFailureMessage,
		})
	}

	// Generate terraform from diagram
	terraform, generationErrors := gen.Generate(ctx, record.ID, record.Data)

	if len(generationErrors) > 0 {
		for _, generationError := range generationErrors {
			if generationError.ErrorType == internal.ErrorTypeTemplate {
				log.Printf("diagram %s: %s", record.ID, generationError.Message)
			}
		}
		// The summary keeps the diagram's own errors and replaces template
		// errors, which hold bucket paths and SDK text, with a fixed sentence.
		summary := internal.SummarizeErrors(generationErrors)
		recordFailure(ctx, orgID, record, summary)
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "Terraform generation completed with errors",
			"error":   summary,
		})
	}

	terraform.ProjectID = record.ProjectID
	terraform.OrgID = orgID

	// Upload to S3
	s3BucketName := os.Getenv("S3_BUCKET_NAME")
	tfWriter, err := writer.NewTerraformWriter(ctx, s3BucketName)
	if err != nil {
		log.Printf("diagram %s: S3 writer: %v", record.ID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to upload the Terraform files",
		})
	}

	if err := tfWriter.Publish(ctx, terraform, record.LatestUpdateAt); err != nil {
		log.Printf("diagram %s: publish: %v", record.ID, err)
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to upload the Terraform files",
		})
	}

	// Return success response
	return generic.Response(http.StatusOK, generic.Json{
		"message":    "Terraform files generated successfully",
		"diagram_id": record.ID,
	})
}

// loadDiagram reads one diagram row into the shape the generator takes. The
// save time is kept to the microsecond, so the editor can tell whether the
// files cover the newest save.
func loadDiagram(ctx context.Context, conn generic.Querier, diagramID, projectID string) (generic.DiagramRecord, error) {
	var record generic.DiagramRecord
	var data []byte
	var updatedAt *time.Time
	err := conn.QueryRow(ctx, diagramQuery, diagramID, projectID).Scan(&record.ID, &record.ProjectID, &data, &updatedAt)
	if err != nil {
		return record, err
	}
	if err := json.Unmarshal(data, &record.Data); err != nil {
		return record, err
	}
	if updatedAt != nil {
		saved := updatedAt.UTC().Format(time.RFC3339Nano)
		record.LatestUpdateAt = &saved
	}
	return record, nil
}

// recordFailure writes a failed status beside the diagram's files, so the
// editor can say why they did not change. It runs on the way to an error
// response and never changes that response: when it cannot write, the log is
// the only record.
func recordFailure(ctx context.Context, orgID string, record generic.DiagramRecord, message string) {
	if orgID == "" || record.ProjectID == "" || record.ID == "" {
		log.Printf("diagram %q: an id is missing, so the failed run has no status", record.ID)
		return
	}
	tfWriter, err := writer.NewTerraformWriter(ctx, os.Getenv("S3_BUCKET_NAME"))
	if err != nil {
		log.Printf("diagram %s: no S3 writer, so the failed run has no status", record.ID)
		return
	}
	if err := tfWriter.RecordFailure(ctx, orgID, record.ProjectID, record.ID, message, record.LatestUpdateAt); err != nil {
		log.Printf("diagram %s: %v", record.ID, err)
	}
}
