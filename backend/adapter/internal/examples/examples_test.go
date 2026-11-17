package examples_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/aws"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/generalopenstack"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/capability"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/examples"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/split"
	"github.com/Alberta2514640/clutter/backend/api/generic/tftemplate"
	"github.com/google/uuid"
)

// regions are a region each connector accepts. OpenStack's manifest lists
// none; DevStack's is RegionOne.
var regions = map[string]string{aws.ID: "ca-central-1", generalopenstack.ID: "RegionOne"}

// registry holds the real connectors, rendering from the repo's templates.
// Nothing here runs Terraform against a cloud.
func registry(t *testing.T) *connectors.Registry {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(here), "..", "..", "..", "infra", "modules", "s3", "uploads", "aws")
	templates, err := tftemplate.NewTemplateLoaderFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	reg := connectors.NewRegistry()
	for _, c := range []connectors.Connector{aws.New(nil, templates, nil), generalopenstack.New(nil)} {
		if err := reg.Register(c); err != nil {
			t.Fatal(err)
		}
	}
	return reg
}

func all(t *testing.T) []examples.Example {
	t.Helper()
	list, err := examples.All()
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestExamplesLoad(t *testing.T) {
	list := all(t)
	want := []string{"single-server", "file-storage", "web-server", "serverless-api", "two-tier-app", "full-app", "hybrid"}
	if len(list) != len(want) {
		t.Fatalf("%d examples, want %d", len(list), len(want))
	}
	for i, e := range list {
		if e.ID != want[i] {
			t.Errorf("example %d is %q, want %q", i, e.ID, want[i])
		}
		g, err := e.Graph()
		if err != nil {
			t.Fatalf("%s: %v", e.ID, err)
		}
		// Every block of a larger example is linked: it shows how blocks
		// work together.
		linked := map[string]bool{}
		for _, ed := range g.Edges {
			linked[ed.Source], linked[ed.Target] = true, true
		}
		for id := range g.Blocks {
			if len(g.Blocks) > 1 && !linked[id] {
				t.Errorf("%s: block %s has no link", e.ID, id)
			}
		}
		if strings.ContainsRune(e.Name+e.Description, '—') {
			t.Errorf("%s: the text holds an em dash", e.ID)
		}
	}
}

func TestFittingKeepsOnlyOfferedClouds(t *testing.T) {
	list := all(t)
	onlyAWS := examples.Fitting(list, func(id string) bool { return id == aws.ID })
	got := map[string][]string{}
	for _, e := range onlyAWS {
		got[e.ID] = e.Clouds
	}
	if len(got) != len(list)-1 || len(got["web-server"]) != 1 || got["web-server"][0] != aws.ID {
		t.Fatalf("with AWS alone: %v", got)
	}
	if _, ok := got["hybrid"]; ok {
		t.Fatal("a split example is offered without every one of its clouds")
	}
	if none := examples.Fitting(list, func(string) bool { return false }); len(none) != 0 {
		t.Fatalf("with no cloud: %d examples", len(none))
	}
	if every := examples.Fitting(list, func(string) bool { return true }); len(every) != len(list) {
		t.Fatalf("with every cloud: %d of %d examples", len(every), len(list))
	}
}

// target is one cloud of one example: the files it renders, by folder name.
type target struct {
	name  string
	files map[string][]byte
}

// rendered checks one example on the clouds it names and returns what each
// cloud renders: for a plain example, each cloud alone; for a split one, one
// deployment over all of them. Every cloud must pass the capability check,
// its connector's own checks and, in a split, its cross-cloud checks, with no
// problem and no note.
func rendered(t *testing.T, reg *connectors.Registry, e examples.Example) []target {
	t.Helper()
	g, err := e.Graph()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if !e.Split() {
		var out []target
		for _, id := range e.Clouds {
			c, ok := reg.Get(id)
			if !ok {
				t.Fatalf("%s names %s, which is not a connector", e.ID, id)
			}
			depID := uuid.NewString()
			d := connectors.Deployment{ID: depID, Prefix: connectors.PrefixFor(depID),
				Connection: connectors.Connection{ID: uuid.NewString(), ConnectorID: id, Region: regions[id]}}
			rep := capability.CheckConnector(c, g, d)
			// A cost note (a NAT gateway) is expected; any other note is not.
			var notes []connectors.Problem
			for _, n := range rep.Notes {
				if n.Code != connectors.CodeCost {
					notes = append(notes, n)
				}
			}
			if !rep.Ready || len(rep.Problems)+len(notes) > 0 {
				t.Fatalf("%s on %s: problems %+v, notes %+v", e.ID, id, rep.Problems, notes)
			}
			files, err := c.(connectors.Previewer).Preview(ctx, g, d)
			if err != nil {
				t.Fatalf("%s on %s: render: %v", e.ID, id, err)
			}
			out = append(out, target{name: id, files: files})
		}
		return out
	}

	var clouds []split.Cloud
	for i, id := range e.Clouds {
		c, ok := reg.Get(id)
		if !ok {
			t.Fatalf("%s names %s, which is not a connector", e.ID, id)
		}
		cl := split.Cloud{ConnectionID: uuid.NewString(), Name: id, Connector: c, Region: regions[id], Primary: i == 0}
		for _, p := range e.Placements {
			if p.Connector == id {
				cl.Nodes = p.NodeIDs
			}
		}
		clouds = append(clouds, cl)
	}
	plan := split.Build(g, uuid.NewString(), nil, clouds)
	if len(plan.Problems) > 0 {
		t.Fatalf("%s does not split: %+v", e.ID, plan.Problems)
	}
	var out []target
	for _, p := range plan.Parts {
		rep := capability.CheckConnector(p.Connector, g, p.Deployment)
		problems := append(rep.Problems, p.Connector.(connectors.CrossCloud).CrossCheck(g, p.Deployment)...)
		if len(problems) > 0 {
			t.Fatalf("%s, part on %s: %+v", e.ID, p.Name, problems)
		}
		files, err := p.Connector.(connectors.Previewer).Preview(ctx, g, p.Deployment)
		if err != nil {
			t.Fatalf("%s, part on %s: render: %v", e.ID, p.Name, err)
		}
		out = append(out, target{name: "split-" + p.Name, files: files})
	}
	return out
}

func TestEveryExamplePassesEveryCloudItNames(t *testing.T) {
	reg := registry(t)
	for _, e := range all(t) {
		t.Run(e.ID, func(t *testing.T) {
			for _, tg := range rendered(t, reg, e) {
				if len(tg.files["main.tf"]) == 0 {
					t.Errorf("%s renders no main.tf", tg.name)
				}
			}
		})
	}
}

// A starter example must not cost money nobody expects: only an example with
// a private VM gets a NAT gateway on AWS (about $40 a month), and every
// example names a box for each of its networks.
func TestOnlyPrivateVMsBringANATGateway(t *testing.T) {
	reg := registry(t)
	for _, e := range all(t) {
		g, err := e.Graph()
		if err != nil {
			t.Fatal(err)
		}
		private := false
		for _, b := range g.Blocks {
			if b.Kind == connectors.KindVM && b.Fields["public"] != true {
				private = true
			}
		}
		if e.Level == "starter" && private {
			t.Errorf("%s is a starter example with a private VM", e.ID)
		}
		for _, tg := range rendered(t, reg, e) {
			if tg.name != aws.ID && tg.name != "split-"+aws.ID {
				continue
			}
			nat := strings.Contains(joined(tg.files), `resource "aws_nat_gateway"`)
			if nat != private {
				t.Errorf("%s on AWS: NAT gateway = %v, private VM = %v", e.ID, nat, private)
			}
		}
	}
}

// The two-tier app proves its link: the web VM pings the database VM at the
// address Clutter hands it and says on its console whether it got through,
// with commands both cirros and Amazon Linux have.
func TestTwoTierWebSaysWhetherItReachedTheDatabase(t *testing.T) {
	for _, e := range all(t) {
		if e.ID != "two-tier-app" {
			continue
		}
		g, err := e.Graph()
		if err != nil {
			t.Fatal(err)
		}
		script, _ := g.Blocks["web"].Fields[connectors.FieldStartupScript].(string)
		for _, want := range []string{
			`ping -c 1 -W 2 "$` + connectors.EnvName("db", connectors.ValueAddress) + `"`,
			`result="web reached db"`,
			`result="web could not reach db"`,
			`>/dev/console`,
		} {
			if !strings.Contains(script, want) {
				t.Errorf("the web VM's script lacks %q", want)
			}
		}
		return
	}
	t.Fatal("no two-tier-app example")
}

func joined(files map[string][]byte) string {
	var b strings.Builder
	for _, body := range files {
		b.Write(body)
	}
	return b.String()
}

// TestExamplesTerraformValidates runs terraform init and validate on what
// every example renders for every cloud it names. It downloads providers, so
// it runs only with CLUTTER_TF_VALIDATE=1; it needs no cloud and no
// credentials.
func TestExamplesTerraformValidates(t *testing.T) {
	if os.Getenv("CLUTTER_TF_VALIDATE") != "1" {
		t.Skip("set CLUTTER_TF_VALIDATE=1 to run terraform validate")
	}
	bin, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("CLUTTER_TF_VALIDATE=1 but terraform is not installed")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	cache = filepath.Join(cache, "clutter-tf-plugins")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	reg := registry(t)
	for _, e := range all(t) {
		for _, tg := range rendered(t, reg, e) {
			t.Run(e.ID+"/"+tg.name, func(t *testing.T) {
				dir := t.TempDir()
				for n, body := range tg.files {
					if err := os.WriteFile(filepath.Join(dir, n), body, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				// Built from nothing, as the shared runner does.
				env := []string{"PATH=/usr/bin:/bin", "HOME=" + dir, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1", "TF_PLUGIN_CACHE_DIR=" + cache}
				for _, args := range [][]string{{"init", "-backend=false", "-input=false", "-no-color"}, {"validate", "-no-color"}} {
					cmd := exec.Command(bin, args...)
					cmd.Dir, cmd.Env = dir, env
					out, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatalf("terraform %s: %v\n%s", args[0], err, out)
					}
					if args[0] == "validate" && !strings.Contains(string(out), "Success! The configuration is valid.") {
						t.Fatalf("validate: %s", out)
					}
				}
			})
		}
	}
}
