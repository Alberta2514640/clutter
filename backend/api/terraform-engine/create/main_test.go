package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal/generator"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	testUser    = "44444444-4444-4444-8444-444444444444"
	testOrg     = "33333333-3333-4333-8333-333333333333"
	otherOrg    = "55555555-5555-4555-8555-555555555555"
	testProject = "11111111-1111-4111-8111-111111111111"
	otherProj   = "66666666-6666-4666-8666-666666666666"
	testDiagram = "22222222-2222-4222-8222-222222222222"
)

// useMock makes the handler's database a pgxmock connection for one test.
func useMock(t *testing.T) pgxmock.PgxConnIface {
	t.Helper()
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	connect = func() (generic.Conn, error) { return mock, nil }
	t.Cleanup(func() { connect = generic.PsqlConnectConn })
	return mock
}

// noDatabase fails the test if the handler opens a connection.
func noDatabase(t *testing.T) {
	t.Helper()
	connect = func() (generic.Conn, error) {
		t.Error("the handler opened the database")
		return nil, errors.New("no database in this test")
	}
	t.Cleanup(func() { connect = generic.PsqlConnectConn })
}

// signedIn is a request from testUser, as the authorizer passes it on.
func signedIn(body any) events.APIGatewayProxyRequest {
	raw, _ := json.Marshal(body)
	return events.APIGatewayProxyRequest{
		Body:           string(raw),
		RequestContext: events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": testUser}},
	}
}

// expectOwnership expects the queries CheckOwnershipPSQL runs for testDiagram
// in testProject, where the project belongs to projectOrg and member says
// whether testUser is in that organization.
func expectOwnership(mock pgxmock.PgxConnIface, projectOrg string, member bool) {
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(testDiagram).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(testProject))
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(testProject).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(projectOrg))
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(projectOrg).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(projectOrg, testUser).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(member))
}

// expectDiagramRow expects the read of testDiagram's row with this layout.
func expectDiagramRow(mock pgxmock.PgxConnIface, layout string) {
	saved := time.Date(2026, 11, 13, 10, 0, 0, 123456000, time.UTC)
	mock.ExpectQuery(`SELECT id, project_id, data, latest_update_at`).WithArgs(testDiagram, testProject).
		WillReturnRows(pgxmock.NewRows([]string{"id", "project_id", "data", "latest_update_at"}).
			AddRow(testDiagram, testProject, []byte(layout), &saved))
}

func TestGenerationRefusesACallerWithNoIdentity(t *testing.T) {
	noDatabase(t)
	res, err := handler(context.Background(), events.APIGatewayProxyRequest{Body: `{"projectId":"` + testProject + `","diagramId":"` + testDiagram + `"}`})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
}

func TestGenerationRefusesABodyWithoutIDs(t *testing.T) {
	noDatabase(t)
	for _, body := range []any{
		map[string]string{"projectId": testProject},
		map[string]string{"projectId": "not-a-uuid", "diagramId": testDiagram},
		// The old webhook body names no projectId or diagramId.
		map[string]any{"type": "UPDATE", "record": map[string]string{"id": testDiagram, "project_id": testProject}},
	} {
		res, err := handler(context.Background(), signedIn(body))
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("%v: status %d: %s", body, res.StatusCode, res.Body)
		}
	}
}

func TestGenerationRefusesAnotherOrganizationsDiagram(t *testing.T) {
	mock := useMock(t)
	// The diagram's project belongs to otherOrg, and testUser is not a member.
	expectOwnership(mock, otherOrg, false)

	res, err := handler(context.Background(), signedIn(map[string]string{"projectId": testProject, "diagramId": testDiagram}))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	// The diagram row is never read: the mock expects no further query.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerationRefusesADiagramNamedUnderAnotherProject(t *testing.T) {
	mock := useMock(t)
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(testDiagram).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(testProject))

	res, err := handler(context.Background(), signedIn(map[string]string{"projectId": otherProj, "diagramId": testDiagram}))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestGenerationSkipsABuildingBlockDiagram: A's generator answers a
// building-block diagram with nothing generated, before it loads a template
// or writes a file (ADR-014).
func TestGenerationSkipsABuildingBlockDiagram(t *testing.T) {
	mock := useMock(t)
	expectOwnership(mock, testOrg, true)
	expectDiagramRow(mock, `{"engine":"blocks","nodes":[{"id":"a","type":"vm","data":{"name":"web"}}],"edges":[]}`)
	// No bucket is set: reaching the generator or the writer would fail.
	t.Setenv("TEMPLATE_BUCKET_NAME", "")
	t.Setenv("S3_BUCKET_NAME", "")

	res, err := handler(context.Background(), signedIn(map[string]string{"projectId": testProject, "diagramId": testDiagram}))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	var body struct {
		DiagramID string `json:"diagram_id"`
		Generated *bool  `json:"generated"`
	}
	if err := json.Unmarshal([]byte(res.Body), &body); err != nil {
		t.Fatal(err)
	}
	if body.DiagramID != testDiagram || body.Generated == nil || *body.Generated {
		t.Fatalf("body: %s", res.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDiagramKeepsTheSaveTimeToTheMicrosecond(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	expectDiagramRow(mock, `{"nodes":[{"id":"a","data":{"label":"S3"}}],"edges":[]}`)
	record, err := loadDiagram(context.Background(), mock, testDiagram, testProject)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != testDiagram || record.ProjectID != testProject || len(record.Data.Nodes) != 1 {
		t.Fatalf("record: %+v", record)
	}
	if record.LatestUpdateAt == nil || *record.LatestUpdateAt != "2026-11-13T10:00:00.123456Z" {
		t.Fatalf("latest_update_at: %v", record.LatestUpdateAt)
	}
}

// TestGenerateOnly is an integration test: it loads the templates from a live S3 bucket.
func TestGenerateOnly(t *testing.T) {
	if os.Getenv("TEMPLATE_BUCKET_NAME") == "" {
		t.Skip("Skipping TestGenerateOnly: TEMPLATE_BUCKET_NAME not set (needs a live template bucket)")
	}

	ctx := context.Background()

	gen, err := generator.NewTerraformGenerator(ctx, os.Getenv("TEMPLATE_BUCKET_NAME"))
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Variables nested inside Data["variables"] to match the frontend shape
	// SanitizeNodes extracts them into node.Variables before generation
	layout := generic.DiagramLayout{
		Nodes: []generic.DiagramNode{
			{
				ID:   "node-apigw-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "API-Gateway",
					"variables": map[string]interface{}{
						"resource_name": "my_api",
						"http_methods":  "GET,POST,DELETE",
					},
				},
			},
			{
				ID:   "node-lambda-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "Lambda",
					"variables": map[string]interface{}{
						"resource_name": "my_handler_one",
						"timeout":       30,
						"handler":       "bootstrap",
					},
				},
			},
			{
				ID:   "node-lambda-2",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "Lambda",
					"variables": map[string]interface{}{
						"resource_name": "my_handler_two",
						"timeout":       30,
						"handler":       "bootstrap",
					},
				},
			},
			{
				ID:   "node-lambda-3",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "Lambda",
					"variables": map[string]interface{}{
						"resource_name": "my_handler_three",
						"timeout":       30,
						"handler":       "bootstrap",
					},
				},
			},
			{
				ID:   "node-lambda-4",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "Lambda",
					"variables": map[string]interface{}{
						"resource_name": "my_handler_four",
						"timeout":       30,
						"handler":       "bootstrap",
					},
				},
			},
			{
				ID:   "node-dynamodb-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "DynamoDB",
					"variables": map[string]interface{}{
						"resource_name": "my_users_table",
						"hash_key":      "userId",
					},
				},
			},
			{
				ID:   "node-s3-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "S3",
					"variables": map[string]interface{}{
						"resource_name": "my_assets_bucket",
					},
				},
			},
			{
				ID:   "node-ec2-1",
				Type: "awsNode",
				Data: map[string]interface{}{
					"label": "EC2",
					"variables": map[string]interface{}{
						"resource_name": "my_server",
					},
				},
			},
		},
		Edges: []generic.DiagramEdge{
			{ID: "e1", Source: "node-apigw-1", Target: "node-lambda-1"},
			{ID: "e2", Source: "node-apigw-1", Target: "node-lambda-2"},
			{ID: "e3", Source: "node-apigw-1", Target: "node-lambda-3"},
			{ID: "e4", Source: "node-apigw-1", Target: "node-lambda-4"},
			{ID: "e5", Source: "node-lambda-1", Target: "node-dynamodb-1"},
			{ID: "e6", Source: "node-lambda-2", Target: "node-dynamodb-1"},
			{ID: "e7", Source: "node-lambda-3", Target: "node-dynamodb-1"},
			{ID: "e8", Source: "node-lambda-4", Target: "node-dynamodb-1"},
			{ID: "e9", Source: "node-lambda-1", Target: "node-s3-1"},
			{ID: "e10", Source: "node-lambda-2", Target: "node-s3-1"},
			{ID: "e11", Source: "node-lambda-3", Target: "node-s3-1"},
			{ID: "e12", Source: "node-lambda-4", Target: "node-s3-1"},
		},
	}

	layout.Nodes = internal.SanitizeNodes(layout.Nodes)

	tf, errors := gen.Generate(ctx, "test-diagram-123", layout)

	if len(errors) > 0 {
		t.Log("Generation errors:")
		for _, e := range errors {
			t.Logf("  [%s] %s: %s", e.ErrorType, e.NodeLabel, e.Message)
		}
		t.FailNow()
	}

	fmt.Println("\n========== main.tf ==========")
	fmt.Println(tf.MainTF)
	fmt.Println("\n========== resources.tf ==========")
	fmt.Println(tf.ResourcesTF)
	fmt.Println("\n========== iam.tf ==========")
	fmt.Println(tf.IAMTF)
	fmt.Println("\n========== outputs.tf ==========")
	fmt.Println(tf.OutputsTF)
}
