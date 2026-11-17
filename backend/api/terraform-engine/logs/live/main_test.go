package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	userID    = "44444444-4444-4444-8444-444444444444"
	orgID     = "33333333-3333-4333-8333-333333333333"
	otherOrg  = "55555555-5555-4555-8555-555555555555"
	projectID = "11111111-1111-4111-8111-111111111111"
	diagramID = "22222222-2222-4222-8222-222222222222"
	otherDia  = "88888888-8888-4888-8888-888888888888"
	taskArn   = "arn:aws:ecs:us-west-2:000000000000:task/cluster/0123456789abcdef0123456789abcdef"
)

func TestLiveLogsRefuseAnotherOrganizationsDiagram(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	connect = func() (generic.Conn, error) { return mock, nil }
	t.Cleanup(func() { connect = generic.PsqlConnectConn })

	// The caller is a member of orgID, but the diagram's project is in otherOrg.
	mock.ExpectQuery(`FROM organizations WHERE id`).WithArgs(orgID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM organization_members`).WithArgs(orgID, userID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT project_id\s+FROM diagrams`).WithArgs(diagramID).
		WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectQuery(`SELECT organization_id\s+FROM projects`).WithArgs(projectID).
		WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow(otherOrg))

	// No ECS settings are set: a request that got past the check would stop
	// with a 500, not the 404 below.
	res, err := handler(context.Background(), events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{"orgId": orgID, "projId": projectID, "diagramId": diagramID, "taskArn": taskArn},
		RequestContext:        events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": userID}},
	})
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

// deployTask is a task as terraform-command-runner starts it.
func deployTask(org, project, diagram string) ecstypes.Task {
	return ecstypes.Task{Overrides: &ecstypes.TaskOverride{ContainerOverrides: []ecstypes.ContainerOverride{{
		Name: aws.String("terraform-deployer"),
		Environment: []ecstypes.KeyValuePair{
			{Name: aws.String("TERRAFORM_DIRECTORY"), Value: aws.String(org + "/" + project + "/" + diagram + "/terraform")},
			{Name: aws.String("DIAGRAM_ID"), Value: aws.String(diagram)},
		},
	}}}}
}

func TestATaskMustBeThisDiagramsDeploy(t *testing.T) {
	cases := []struct {
		name string
		task ecstypes.Task
		want bool
	}{
		{"this diagram's deploy", deployTask(orgID, projectID, diagramID), true},
		{"another diagram's deploy", deployTask(orgID, projectID, otherDia), false},
		{"another organization's deploy", deployTask(otherOrg, projectID, diagramID), false},
		{"a task with no overrides", ecstypes.Task{}, false},
		{"the diagram id without the directory", ecstypes.Task{Overrides: &ecstypes.TaskOverride{ContainerOverrides: []ecstypes.ContainerOverride{{
			Environment: []ecstypes.KeyValuePair{{Name: aws.String("DIAGRAM_ID"), Value: aws.String(diagramID)}},
		}}}}, false},
	}
	for _, c := range cases {
		if got := taskBelongsTo(c.task, orgID, projectID, diagramID); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
