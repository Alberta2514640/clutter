package aws

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/google/uuid"
)

// results.tf names a VM's private address, and its public one only when it is
// public; a bucket's and a table's name; and each HTTP API's address on its
// first route, in AWS's form or, with an endpoint, LocalStack's.
func TestResultsOutputs(t *testing.T) {
	for _, endpoint := range []string{"", "http://127.0.0.1:4566"} {
		c, _, _ := newTestConnector(t)
		if endpoint != "" {
			if _, err := c.WithEndpoint(endpoint); err != nil {
				t.Fatal(err)
			}
		}
		files, err := c.Preview(context.Background(), fullGraph(), testDeployment("ca-central-1"))
		if err != nil {
			t.Fatal(err)
		}
		if err := terraform.CheckFiles(files, providers); err != nil {
			t.Fatalf("the runner refuses the files: %v", err)
		}
		results := string(files["results.tf"])
		for _, want := range []string{
			`output "result_private_address_web"`,
			`output "result_public_address_web"`,
			`output "result_private_address_worker"`,
			`output "result_bucket_bucket"`,
			`output "result_table_table"`,
			`output "result_url_api"`,
			`output "result_url_api2"`,
		} {
			if !strings.Contains(results, want) {
				t.Errorf("endpoint %q: results.tf lacks %s:\n%s", endpoint, want, results)
			}
		}
		if strings.Contains(results, "result_public_address_worker") {
			t.Errorf("a private VM has a public address output:\n%s", results)
		}
		aws := `invoke_url}/api_handler"`
		local := `"http://127.0.0.1:4566/restapis/${aws_api_gateway_rest_api.`
		if endpoint == "" && (!strings.Contains(results, aws) || strings.Contains(results, "restapis")) {
			t.Errorf("on AWS the API's address is not its invoke URL:\n%s", results)
		}
		if endpoint != "" && (!strings.Contains(results, local) || strings.Contains(results, "invoke_url")) {
			t.Errorf("on LocalStack the API's address is not the endpoint's form:\n%s", results)
		}
	}
}

// An apply returns each block's results, read from the run's outputs.
func TestApplyReturnsResults(t *testing.T) {
	c, _, _ := newTestConnector(t)
	var log bytes.Buffer
	res, err := c.Apply(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &log}, fullGraph(), testDeployment("ca-central-1"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]connectors.Result{
		"web": {
			{Name: connectors.ResultPrivateAddress, Value: fakeResults[connectors.ResultPrivateAddress]},
			{Name: connectors.ResultPublicAddress, Value: fakeResults[connectors.ResultPublicAddress]},
		},
		"worker": {{Name: connectors.ResultPrivateAddress, Value: fakeResults[connectors.ResultPrivateAddress]}},
		"bucket": {{Name: connectors.ResultBucket, Value: fakeResults[connectors.ResultBucket]}},
		"table":  {{Name: connectors.ResultTable, Value: fakeResults[connectors.ResultTable]}},
		"api":    {{Name: connectors.ResultURL, Value: fakeResults[connectors.ResultURL]}},
		"api2":   {{Name: connectors.ResultURL, Value: fakeResults[connectors.ResultURL]}},
	}
	if !reflect.DeepEqual(res.Results, want) {
		t.Fatalf("results = %+v, want %+v", res.Results, want)
	}
	if strings.Contains(log.String(), `"value"`) {
		t.Fatal("the run's outputs reached its log")
	}
}
