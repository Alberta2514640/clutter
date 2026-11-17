package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/google/uuid"
)

// outputLauncher answers the output step with a secret on stdout and a
// warning on stderr.
type outputLauncher struct{ steps []string }

func (o *outputLauncher) Launch(_ context.Context, t runtask.Task) error {
	o.steps = append(o.steps, strings.Join(t.Args, " "))
	if t.Args[0] == runtask.OutputStep {
		fmt.Fprint(t.Output, `{"x":{"sensitive":true,"value":"s3cr3t"}}`)
		fmt.Fprintln(t.Stderr, "Warning: something")
		return nil
	}
	fmt.Fprintf(t.Output, "%s done\n", t.Args[0])
	return nil
}

func TestOutputsGoOnlyToTheirWriter(t *testing.T) {
	files := map[string][]byte{"main.tf": []byte("resource \"terraform_data\" \"n1\" {}\n")}
	for _, a := range []connectors.Action{connectors.ActionPlan, connectors.ActionApply} {
		l := &outputLauncher{}
		r := newRunner(t, "/bin/terraform-not-run", l)
		var log, outputs bytes.Buffer
		err := r.Run(context.Background(), Job{DeploymentID: uuid.NewString(), RunID: uuid.NewString(), Action: a, Files: files, Log: &log, Outputs: &outputs})
		if err != nil {
			t.Fatal(err)
		}
		if l.steps[len(l.steps)-1] != "output -json -no-color" {
			t.Fatalf("%s steps %v", a, l.steps)
		}
		if outputs.String() != `{"x":{"sensitive":true,"value":"s3cr3t"}}` {
			t.Fatalf("outputs %q", outputs.String())
		}
		if strings.Contains(log.String(), "s3cr3t") || !strings.Contains(log.String(), "Warning: something") {
			t.Fatalf("log %q", log.String())
		}
	}
	// A destroy has no outputs to hand on.
	l := &outputLauncher{}
	r := newRunner(t, "/bin/terraform-not-run", l)
	var outputs bytes.Buffer
	if err := r.Run(context.Background(), Job{DeploymentID: uuid.NewString(), RunID: uuid.NewString(), Action: connectors.ActionDestroy,
		Files: files, Log: &bytes.Buffer{}, Outputs: &outputs}); err != nil {
		t.Fatal(err)
	}
	if len(l.steps) != 2 || outputs.Len() != 0 {
		t.Fatalf("destroy steps %v", l.steps)
	}
}

// With real Terraform, a sensitive output is in the outputs and nowhere in the
// run's log.
func TestRealTerraformKeepsSensitiveOutputsOutOfTheLog(t *testing.T) {
	bin, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform is not installed")
	}
	bin, err = filepath.Abs(bin)
	if err != nil {
		t.Fatal(err)
	}
	r := newRunner(t, bin, runtask.Local{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	files := map[string][]byte{"main.tf": []byte(`resource "terraform_data" "k" {
  input = sensitive("real-s3cr3t-value")
}

output "cross_x0123456789_secret_access_key" {
  value     = terraform_data.k.output
  sensitive = true
}
`)}
	var log, outputs bytes.Buffer
	if err := r.Run(ctx, Job{DeploymentID: uuid.NewString(), RunID: uuid.NewString(), Action: connectors.ActionApply, Files: files, Log: &log, Outputs: &outputs}); err != nil {
		t.Fatalf("apply: %v\n%s", err, log.String())
	}
	if !strings.Contains(outputs.String(), "real-s3cr3t-value") || !strings.Contains(outputs.String(), `"sensitive": true`) {
		t.Fatalf("outputs %q", outputs.String())
	}
	if strings.Contains(log.String(), "real-s3cr3t-value") {
		t.Fatalf("the sensitive output reached the log:\n%s", log.String())
	}
}
