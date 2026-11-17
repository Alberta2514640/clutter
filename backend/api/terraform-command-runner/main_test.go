package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pashagolub/pgxmock/v4"
)

// defaultTaskRunner is the production newTaskRunner, restored after a test.
var defaultTaskRunner = newTaskRunner

const (
	userID    = "44444444-4444-4444-8444-444444444444"
	orgID     = "33333333-3333-4333-8333-333333333333"
	otherOrg  = "55555555-5555-4555-8555-555555555555"
	projectID = "11111111-1111-4111-8111-111111111111"
	diagramID = "22222222-2222-4222-8222-222222222222"
	accountID = "77777777-7777-4777-8777-777777777777"
)

func request(t *testing.T, body map[string]string) events.APIGatewayProxyRequest {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return events.APIGatewayProxyRequest{
		Body:           string(raw),
		RequestContext: events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	}
}

func TestRunnerRefusesAnotherOrganizationsProject(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	connect = func() (generic.Conn, error) { return mock, nil }
	t.Cleanup(func() { connect = generic.PsqlConnectConn })

	// The caller is a member of orgID and names its own account, but the
	// diagram's project is in otherOrg.
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(orgID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(orgID, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(diagramID).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(projectID).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(otherOrg))

	res, err := handler(context.Background(), request(t, map[string]string{
		"organizationId": orgID, "projectId": projectID, "diagramId": diagramID,
		"accountAccessRoleId": accountID, "command": "apply",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	// No role is read and no log row is written.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

const (
	roleARN    = "arn:aws:iam::111122223333:role/AllowClutterToDeployTerraformRole-abc"
	externalID = "this-orgs-external-id"
)

// fakeRunner records the RunTask input and answers with out or err.
type fakeRunner struct {
	input *ecs.RunTaskInput
	out   *ecs.RunTaskOutput
	err   error
}

func (f *fakeRunner) RunTask(_ context.Context, in *ecs.RunTaskInput, _ ...func(*ecs.Options)) (*ecs.RunTaskOutput, error) {
	f.input = in
	return f.out, f.err
}

// expectStart sets up the queries of a run the caller may start, up to the
// RUNNING row.
func expectStart(t *testing.T, runner *fakeRunner) pgxmock.PgxConnIface {
	t.Helper()
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	connect = func() (generic.Conn, error) { return mock, nil }
	newTaskRunner = func(context.Context) (taskRunner, error) { return runner, nil }
	t.Cleanup(func() {
		connect = generic.PsqlConnectConn
		newTaskRunner = defaultTaskRunner
	})
	// The Lambda holds the database string; the task must never get it.
	t.Setenv("PSQL_CONNECTION_STRING", "postgres://clutter:secret@db.example:5432/postgres")
	t.Setenv("CONTAINER_NAME", "terraform-deployer")

	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(orgID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(orgID, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(diagramID).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(projectID).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(orgID))
	mock.ExpectQuery(`FROM aws_account_access_roles`).WithArgs(accountID, orgID).
		WillReturnRows(pgxmock.NewRows([]string{"role_arn", "external_id"}).AddRow(roleARN, externalID))
	mock.ExpectExec(`INSERT INTO public.diagram_deployment_logs`).
		WithArgs(diagramID, pgxmock.AnyArg(), "apply", "RUNNING").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	return mock
}

func startRequest(t *testing.T) events.APIGatewayProxyRequest {
	return request(t, map[string]string{
		"organizationId": orgID, "projectId": projectID, "diagramId": diagramID,
		"accountAccessRoleId": accountID, "command": "apply",
	})
}

func TestRunnerTaskGetsOnlyThisRunsRoleAndNoDatabase(t *testing.T) {
	runner := &fakeRunner{out: &ecs.RunTaskOutput{Tasks: []ecstypes.Task{{TaskArn: aws.String("arn:aws:ecs:ca-central-1:1:task/clutter/abc")}}}}
	mock := expectStart(t, runner)

	res, err := handler(context.Background(), startRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if runner.input == nil || runner.input.Overrides == nil || len(runner.input.Overrides.ContainerOverrides) != 1 {
		t.Fatalf("want one container override, got %+v", runner.input)
	}
	override := runner.input.Overrides.ContainerOverrides[0]
	if len(override.EnvironmentFiles) != 0 {
		t.Fatalf("the task gets environment files: %+v", override.EnvironmentFiles)
	}

	// The task's whole environment is this list: nothing else, and nothing
	// that names or holds a database.
	want := map[string]string{
		"AWS_REGION":              generic.DefaultRegion(),
		"TERRAFORM_DIRECTORY":     orgID + "/" + projectID + "/" + diagramID + "/terraform",
		"CLIENT_ROLE_ARN":         roleARN,
		"ASSUME_ROLE_EXTERNAL_ID": externalID,
		"COMMAND":                 "apply",
		"DIAGRAM_ID":              diagramID,
	}
	got := map[string]string{}
	for _, pair := range override.Environment {
		name, value := aws.ToString(pair.Name), aws.ToString(pair.Value)
		if _, dup := got[name]; dup {
			t.Fatalf("%s is set twice", name)
		}
		got[name] = value
		lower := strings.ToLower(name + "=" + value)
		for _, word := range []string{"psql", "postgres", "database", "secret"} {
			if strings.Contains(lower, word) {
				t.Errorf("override %s holds %q", name, word)
			}
		}
	}
	commandID, ok := got["COMMAND_ID"]
	if !ok || commandID == "" {
		t.Fatalf("no COMMAND_ID in %v", got)
	}
	delete(got, "COMMAND_ID")
	if len(got) != len(want) {
		t.Fatalf("overrides %v, want %v plus COMMAND_ID", got, want)
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("%s = %q, want %q", name, got[name], value)
		}
	}
}

func TestRunnerMarksARunThatDidNotStartFailed(t *testing.T) {
	cases := map[string]*fakeRunner{
		"RunTask error":  {err: errors.New("throttled")},
		"no task placed": {out: &ecs.RunTaskOutput{Failures: []ecstypes.Failure{{Reason: aws.String("RESOURCE:MEMORY")}}}},
	}
	for name, runner := range cases {
		t.Run(name, func(t *testing.T) {
			mock := expectStart(t, runner)
			mock.ExpectExec(`UPDATE public.diagram_deployment_logs\s+SET status = 'FAILED'`).
				WithArgs(diagramID, pgxmock.AnyArg()).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))

			res, err := handler(context.Background(), startRequest(t))
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusInternalServerError {
				t.Fatalf("status %d: %s", res.StatusCode, res.Body)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRunnerRefusesAMissingDiagramID(t *testing.T) {
	connect = func() (generic.Conn, error) {
		t.Error("the handler opened the database")
		return nil, context.Canceled
	}
	t.Cleanup(func() { connect = generic.PsqlConnectConn })

	res, err := handler(context.Background(), request(t, map[string]string{
		"organizationId": orgID, "projectId": projectID, "accountAccessRoleId": accountID, "command": "apply",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d: %s", res.StatusCode, res.Body)
	}
}
