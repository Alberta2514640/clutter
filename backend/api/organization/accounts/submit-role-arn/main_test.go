package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// A role the stack did not create is refused before the database is reached,
// so B never meets a linked account it cannot deploy with.
func TestHandlerRefusesARoleTheStackDidNotCreate(t *testing.T) {
	for _, arn := range []string{
		"arn:aws:iam::123456789012:role/SomeOtherRole-ab12cd34",
		"arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-abc",
		"",
	} {
		res, err := handler(events.APIGatewayProxyRequest{
			Body: `{"role_arn":"` + arn + `"}`,
			PathParameters: map[string]string{
				"organizationId": "11111111-1111-4111-8111-111111111111",
				"accountId":      "22222222-2222-4222-8222-222222222222",
			},
			RequestContext: events.APIGatewayProxyRequestContext{Authorizer: map[string]any{"id": "33333333-3333-4333-8333-333333333333"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusBadRequest || !strings.Contains(res.Body, "AllowClutterToDeployTerraformRole") {
			t.Errorf("role %q: got %d %s, want 400 naming the stack's role", arn, res.StatusCode, res.Body)
		}
	}
}
