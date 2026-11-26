package generalopenstack

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/runtask"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
	"github.com/google/uuid"
)

// TestRealCloud verifies, plans, applies and destroys the sample diagram on a
// real OpenStack cloud, such as DevStack. It creates resources, so a person
// runs it; it runs only when all four credential variables are set:
//
//	CLUTTER_OPENSTACK_AUTH_URL         Keystone v3 address, ending in /v3
//	CLUTTER_OPENSTACK_REGION           for DevStack, RegionOne
//	CLUTTER_OPENSTACK_APP_CRED_ID      an application credential's id
//	CLUTTER_OPENSTACK_APP_CRED_SECRET  its secret
//
// Optional: CLUTTER_OPENSTACK_IMAGE (default "cirros", matched as the start of
// an image name), CLUTTER_OPENSTACK_FLAVOR (default "m1.tiny") and
// CLUTTER_OPENSTACK_EXTERNAL_NETWORK (default "public"). Terraform must be on
// PATH. A failed apply is destroyed before the test ends.
func TestRealCloud(t *testing.T) {
	rc := newRealCloud(t)
	c, d, g := rc.c, rc.deployment(), sampleGraph()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	// The log stands in for the job runner's, which removes credential values.
	var log bytes.Buffer
	show := func() string { return rc.redact(log.String()) }
	run := func(a connectors.Action) connectors.Run {
		return connectors.Run{ID: uuid.NewString(), Action: a, Log: &log}
	}

	if err := c.VerifyConnection(ctx, d.Connection); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ps := c.Validate(g, d); len(ps) > 0 {
		t.Fatalf("validate: %+v", ps)
	}
	plan, err := c.Plan(ctx, run(connectors.ActionPlan), g, d)
	if err != nil {
		t.Fatalf("plan: %v\n%s", err, show())
	}
	t.Logf("plan: %+v", plan)
	if plan.Add == 0 {
		t.Fatalf("the plan adds nothing\n%s", show())
	}

	t.Cleanup(func() {
		dctx, dcancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer dcancel()
		if _, err := c.Destroy(dctx, run(connectors.ActionDestroy), d); err != nil {
			t.Errorf("destroy: %v\n%s", err, show())
		}
		t.Logf("destroyed deployment %s (names start with %s)", d.ID, d.Prefix)
	})
	res, err := c.Apply(ctx, run(connectors.ActionApply), g, d)
	if err != nil {
		t.Fatalf("apply: %v\n%s", err, show())
	}
	for _, nodeID := range []string{"n1", "n2", "n3", "n4"} {
		if res.Resources[nodeID] == "" {
			t.Errorf("apply returned no cloud id for %s: %v", nodeID, res.Resources)
		}
	}
	t.Logf("built: %v", res.Resources)
}

// realCloud is the real OpenStack cloud the CLUTTER_OPENSTACK_* variables
// name (see TestRealCloud), with this connector running a real terraform.
type realCloud struct {
	c        connectors.Connector
	creds    connectors.Secrets
	region   string
	settings map[string]any
}

// newRealCloud skips the test unless the four credential variables are set.
func newRealCloud(t *testing.T) *realCloud {
	t.Helper()
	vars := map[string]string{}
	for _, name := range []string{"CLUTTER_OPENSTACK_AUTH_URL", "CLUTTER_OPENSTACK_REGION", "CLUTTER_OPENSTACK_APP_CRED_ID", "CLUTTER_OPENSTACK_APP_CRED_SECRET"} {
		vars[name] = os.Getenv(name)
		if vars[name] == "" {
			t.Skip("set the CLUTTER_OPENSTACK_* variables to run against a real OpenStack cloud")
		}
	}
	bin, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("terraform is not installed")
	}
	if bin, err = filepath.Abs(bin); err != nil {
		t.Fatal(err)
	}
	setting := func(name, def string) string {
		if v := os.Getenv(name); v != "" {
			return v
		}
		return def
	}

	root := t.TempDir()
	c := New(&terraform.Runner{
		Bin:            bin,
		WorkDir:        filepath.Join(root, "work"),
		StateDir:       filepath.Join(root, "state"),
		PluginCacheDir: filepath.Join(root, "plugins"),
		Launcher:       runtask.Local{},
	}, AllowPrivateTargets()) // a laptop's DevStack is at a private address, over http
	return &realCloud{
		c: c,
		creds: connectors.Secrets{
			credAuthURL: vars["CLUTTER_OPENSTACK_AUTH_URL"],
			credID:      vars["CLUTTER_OPENSTACK_APP_CRED_ID"],
			credSecret:  vars["CLUTTER_OPENSTACK_APP_CRED_SECRET"],
		},
		region: vars["CLUTTER_OPENSTACK_REGION"],
		settings: map[string]any{
			"image":            setting("CLUTTER_OPENSTACK_IMAGE", "cirros"),
			"flavor_small":     setting("CLUTTER_OPENSTACK_FLAVOR", "m1.tiny"),
			"external_network": setting("CLUTTER_OPENSTACK_EXTERNAL_NETWORK", "public"),
		},
	}
}

// deployment is a new deployment on the cloud, with its own name prefix.
func (r *realCloud) deployment() connectors.Deployment {
	id := uuid.NewString()
	return connectors.Deployment{
		ID:     id,
		Prefix: connectors.PrefixFor(id),
		Connection: connectors.Connection{
			ID: uuid.NewString(), ConnectorID: ID, Region: r.region, Credentials: r.creds,
		},
		Settings: r.settings,
	}
}

// redact removes the credential values from s, as the job runner does.
func (r *realCloud) redact(s string) string {
	for _, v := range r.creds.Values() {
		s = strings.ReplaceAll(s, v, "[redacted]")
	}
	return s
}
