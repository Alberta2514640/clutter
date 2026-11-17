package aws

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/google/uuid"
)

func TestAnEndpointMustBeOnThisMachine(t *testing.T) {
	for _, ok := range []string{"http://127.0.0.1:4566", "http://localhost:4566/", "https://[::1]:4566"} {
		c, _, _ := newTestConnector(t)
		if _, err := c.WithEndpoint(ok); err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
		if _, err := NewLocalStackAssumer(ok); err != nil {
			t.Errorf("assumer %q refused: %v", ok, err)
		}
	}
	for _, bad := range []string{
		"", "127.0.0.1:4566", "ftp://127.0.0.1:4566", "http://127.0.0.1", "http://10.0.0.5:4566",
		"http://localstack.example.com:4566", "http://169.254.169.254:80", "http://127.0.0.1:4566/path",
		"http://127.0.0.1:4566?x=1", "http://user:pass@127.0.0.1:4566", `http://127.0.0.1:4566/"}`,
	} {
		c, _, _ := newTestConnector(t)
		if _, err := c.WithEndpoint(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
		if _, err := NewLocalStackAssumer(bad); err == nil {
			t.Errorf("assumer %q accepted", bad)
		}
	}
}

// TestAnEndpointReachesEveryRunAndOnlyThen checks that, with an endpoint, the
// preview, a plan and a destroy all carry the override file and the run's
// environment names the endpoint, which the runner accepts; and that without
// one, neither appears.
func TestAnEndpointReachesEveryRunAndOnlyThen(t *testing.T) {
	for _, endpoint := range []string{"", "http://127.0.0.1:4566"} {
		c, _, ft := newTestConnector(t)
		if endpoint != "" {
			if _, err := c.WithEndpoint(endpoint); err != nil {
				t.Fatal(err)
			}
		}
		d := testDeployment("ca-central-1")
		files, err := c.Preview(context.Background(), fullGraph(), d)
		if err != nil {
			t.Fatal(err)
		}
		if err := terraform.CheckFiles(files, providers); err != nil {
			t.Fatalf("the runner refuses the files: %v", err)
		}
		override, has := files["endpoint_override.tf"]
		if has != (endpoint != "") {
			t.Fatalf("endpoint %q: override file present = %v", endpoint, has)
		}
		if has {
			for _, svc := range []string{"apigateway", "dynamodb", "ec2", "iam", "lambda", "s3", "ssm", "sts"} {
				if !strings.Contains(string(override), svc) || !strings.Contains(string(override), `"`+endpoint+`"`) {
					t.Errorf("the override does not send %s to the endpoint:\n%s", svc, override)
				}
			}
		}

		rec := &fileRecorder{inner: ft}
		c.tf.Launcher = rec
		var log bytes.Buffer
		if _, err := c.Plan(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionPlan, Log: &log}, fullGraph(), d); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Destroy(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionDestroy, Log: &log}, d); err != nil {
			t.Fatal(err)
		}
		if len(rec.steps) == 0 {
			t.Fatal("no Terraform step ran")
		}
		for i, names := range rec.steps {
			if names["endpoint_override.tf"] != (endpoint != "") {
				t.Errorf("endpoint %q: step %d has the override file = %v", endpoint, i, names["endpoint_override.tf"])
			}
		}
		for _, env := range ft.envs {
			if env["AWS_ENDPOINT_URL"] != endpoint {
				t.Errorf("run environment AWS_ENDPOINT_URL = %q, want %q", env["AWS_ENDPOINT_URL"], endpoint)
			}
		}
	}
}

// fileRecorder records the file names in each step's folder, then runs the
// fake terraform.
type fileRecorder struct {
	inner *fakeTerraform
	steps []map[string]bool
}

func (r *fileRecorder) Launch(ctx context.Context, task runtask.Task) error {
	entries, err := os.ReadDir(task.Dir)
	if err != nil {
		return err
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
	}
	r.steps = append(r.steps, names)
	return r.inner.Launch(ctx, task)
}
