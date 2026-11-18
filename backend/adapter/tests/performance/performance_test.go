// Package performance times System B through its real HTTP routes: 20
// Capability Check requests (GET .../compatibility) and 5 deployments (an
// apply, polled to completion), and reports the median and maximum of each.
//
// The cloud is General's OpenStack connector with the terraform binary faked,
// so the numbers are B's own cost: sign-in, the store, the translator, the
// capability check, rendering, the run task's folder and checks, and the job
// runner. A real cloud adds its own minutes on top. Set PERF_RESULTS to a
// file path to also write the numbers as JSON.
//
// It needs Docker for Postgres, and skips without it.
package performance

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/tests/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

// Pass bars on a developer laptop. They are generous on purpose: the test is
// for spotting a regression of an order of magnitude, not for tuning.
const (
	checkBar  = 500 * time.Millisecond
	deployBar = 10 * time.Second
)

const doc = `{"engine":"blocks","nodes":[
 {"id":"vm","type":"vm","data":{"name":"web","fields":{"size":"small","public":true}}},
 {"id":"vm2","type":"vm","data":{"name":"api","fields":{"size":"medium"}}},
 {"id":"net","type":"network","data":{"name":"net","fields":{"cidr":"10.20.0.0/24"}}},
 {"id":"fw","type":"firewall","data":{"name":"fw","fields":{"ingress":"tcp/22,tcp/80","source_cidr":"0.0.0.0/0"}}},
 {"id":"vol","type":"volume","data":{"name":"data","fields":{"size_gb":5}}}],
 "edges":[{"source":"vm","target":"net"},{"source":"vm2","target":"net"},{"source":"fw","target":"vm"},{"source":"vol","target":"vm"},{"source":"vm","target":"vm2"}]}`

type stats struct {
	Name    string    `json:"name"`
	Count   int       `json:"count"`
	Median  float64   `json:"median_ms"`
	Max     float64   `json:"max_ms"`
	Min     float64   `json:"min_ms"`
	Samples []float64 `json:"samples_ms"`
	Bar     float64   `json:"bar_ms"`
}

func summarize(name string, ds []time.Duration, bar time.Duration) stats {
	ms := make([]float64, len(ds))
	for i, d := range ds {
		ms[i] = float64(d.Microseconds()) / 1000
	}
	sorted := append([]float64(nil), ms...)
	sort.Float64s(sorted)
	n := len(sorted)
	median := sorted[n/2]
	if n%2 == 0 {
		median = (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return stats{Name: name, Count: n, Median: median, Max: sorted[n-1], Min: sorted[0], Samples: ms, Bar: float64(bar.Milliseconds())}
}

func TestCapabilityCheckAndDeploymentTiming(t *testing.T) {
	keystone := testkit.NewKeystone(t)
	env := testkit.NewEnv(t, testkit.Options{Workers: 2}, testkit.NewOpenStack(t, &testkit.FakeTerraform{}))
	tn := env.NewTenant(t, doc)
	creds := keystone.Secrets(testkit.OpenStackCredID, testkit.OpenStackSecret)

	conn := env.Connect(t, &tn, "general-openstack", testkit.OpenStackRegion, creds)
	dep := env.Deploy(t, &tn, conn, map[string]any{"image": "cirros"}, nil)

	// One untimed call warms the connection pool, as a running server is.
	env.Call(t, &tn, "GET", "/v1/organizations/"+tn.Org+"/deployments/"+dep+"/compatibility", nil)

	var checks []time.Duration
	for i := 0; i < 20; i++ {
		start := time.Now()
		code, body := env.Call(t, &tn, "GET", "/v1/organizations/"+tn.Org+"/deployments/"+dep+"/compatibility", nil)
		checks = append(checks, time.Since(start))
		if code != http.StatusOK || !strings.Contains(body, `"ready":true`) {
			t.Fatalf("check %d: %d %s", i+1, code, body)
		}
	}

	// Five deployments, each its own connection and deployment, applied one
	// after another and timed from the click to "completed".
	var deploys []time.Duration
	for i := 0; i < 5; i++ {
		c := env.Connect(t, &tn, "general-openstack", testkit.OpenStackRegion, creds)
		d := env.Deploy(t, &tn, c, map[string]any{"image": "cirros"}, nil)
		start := time.Now()
		run := env.StartRun(t, &tn, d, "apply")
		seen := env.WaitRun(t, &tn, run, time.Minute)
		deploys = append(deploys, time.Since(start))
		if seen[len(seen)-1] != "completed" {
			t.Fatalf("deployment %d ended %v:\n%s", i+1, seen, env.RunLog(t, &tn, run))
		}
	}

	results := []stats{
		summarize("Capability Check (GET compatibility)", checks, checkBar),
		summarize("Deployment (apply, click to completed)", deploys, deployBar),
	}
	for _, s := range results {
		t.Logf("%-40s n=%2d  median %8.1f ms  max %8.1f ms  min %8.1f ms  (bar %.0f ms)", s.Name, s.Count, s.Median, s.Max, s.Min, s.Bar)
	}
	if path := os.Getenv("PERF_RESULTS"); path != "" {
		b, _ := json.MarshalIndent(map[string]any{"when": time.Now().UTC().Format(time.RFC3339), "results": results}, "", "  ")
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range results {
		if s.Max > s.Bar {
			t.Errorf("%s: max %.1f ms is over the %.0f ms bar", s.Name, s.Max, s.Bar)
		}
	}
}
