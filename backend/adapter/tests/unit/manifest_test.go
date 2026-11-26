package unit

import (
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/tests/testkit"
)

// realConnectors are every connector B ships, wired to fakes.
func realConnectors(t *testing.T) []connectors.Connector {
	fake := &testkit.FakeTerraform{}
	return []connectors.Connector{testkit.NewMock(), testkit.NewOpenStack(t, fake), testkit.NewAWS(t, fake)}
}

func TestEveryShippedManifestIsWellFormed(t *testing.T) {
	for _, c := range realConnectors(t) {
		t.Run(c.ID(), func(t *testing.T) {
			m := c.Manifest()
			if err := m.Check(); err != nil {
				t.Fatal(err)
			}
			if err := c.Credentials().Check(); err != nil {
				t.Fatal(err)
			}
			for _, k := range connectors.StandardBlocks() {
				if _, ok := m.Block(k); !ok {
					t.Fatalf("no treatment for the standard block %q", k)
				}
			}
		})
	}
}

func TestManifestLookups(t *testing.T) {
	m := testkit.NewMock().Manifest()
	if b, ok := m.Block(connectors.KindVM); !ok || b.Treatment != connectors.TreatmentAsIs {
		t.Fatalf("vm: %+v %v", b, ok)
	}
	if b, ok := m.Block(connectors.KindBucket); !ok || b.Treatment != connectors.TreatmentRemoved {
		t.Fatalf("bucket: %+v %v", b, ok)
	}
	if _, ok := m.Block("gpu"); ok {
		t.Fatal("an undeclared kind was found")
	}
	if !m.AllowsEdge(connectors.KindVM, connectors.KindNetwork) || m.AllowsEdge(connectors.KindNetwork, connectors.KindVM) {
		t.Fatal("edges are directed: vm to network only")
	}
	if !m.AllowsRegion("mock-1") || m.AllowsRegion("us-east-1") || m.AllowsRegion("") {
		t.Fatal("regions")
	}
}

func TestManifestCheckRefusesBrokenManifests(t *testing.T) {
	base := testkit.NewMock().Manifest
	cases := map[string]func() connectors.Manifest{
		"bad connector id": func() connectors.Manifest { m := base(); m.Connector = "Mock Cloud!"; return m },
		"no title":         func() connectors.Manifest { m := base(); m.Title = ""; return m },
		"a standard block missing": func() connectors.Manifest {
			m := base()
			m.Blocks = m.Blocks[1:]
			return m
		},
		"an edge to an undeclared kind": func() connectors.Manifest {
			m := base()
			m.Edges = append(m.Edges, connectors.EdgeSpec{Source: connectors.KindVM, Target: "gpu", Meaning: "x", Why: "y"})
			return m
		},
		"a repeated region": func() connectors.Manifest { m := base(); m.Regions = []string{"mock-1", "mock-1"}; return m },
		"a default outside its allowed values": func() connectors.Manifest {
			m := base()
			m.Settings = []connectors.FieldSpec{{Name: "image", Title: "Image", Type: connectors.FieldEnum, Default: "windows", Allowed: []string{"debian-12"}}}
			return m
		},
		"a field name with capitals": func() connectors.Manifest {
			m := base()
			m.Settings = []connectors.FieldSpec{{Name: "Image", Title: "Image", Type: connectors.FieldString}}
			return m
		},
		"a new kind not marked new": func() connectors.Manifest {
			m := base()
			m.Blocks = append(m.Blocks, connectors.BlockSpec{Kind: "gpu", Treatment: connectors.TreatmentAsIs})
			return m
		},
	}
	for name, mk := range cases {
		t.Run(name, func(t *testing.T) {
			if err := mk().Check(); err == nil {
				t.Fatal("Check accepted it")
			}
		})
	}
}

func TestCheckValueByType(t *testing.T) {
	long := make([]byte, connectors.MaxStringField+1)
	for i := range long {
		long[i] = 'a'
	}
	cases := []struct {
		f    connectors.FieldSpec
		v    any
		good bool
	}{
		{connectors.FieldSpec{Type: connectors.FieldString}, "ok", true},
		{connectors.FieldSpec{Type: connectors.FieldString}, 5.0, false},
		{connectors.FieldSpec{Type: connectors.FieldString}, string(long), false},
		{connectors.FieldSpec{Type: connectors.FieldInt}, float64(20), true},
		{connectors.FieldSpec{Type: connectors.FieldInt}, 20.5, false},
		{connectors.FieldSpec{Type: connectors.FieldInt}, "20", false},
		{connectors.FieldSpec{Type: connectors.FieldBool}, true, true},
		{connectors.FieldSpec{Type: connectors.FieldBool}, "true", false},
		{connectors.FieldSpec{Type: connectors.FieldEnum, Allowed: []string{"small"}}, "small", true},
		{connectors.FieldSpec{Type: connectors.FieldEnum, Allowed: []string{"small"}}, "huge", false},
		{connectors.FieldSpec{Type: connectors.FieldText}, "#!/bin/sh\necho hi\n", true},
		{connectors.FieldSpec{Type: "date"}, "2026-11-18", false},
	}
	for i, c := range cases {
		err := connectors.CheckValue(c.f, c.v)
		if (err == nil) != c.good {
			t.Errorf("case %d (%s, %#v): err %v, want good=%v", i, c.f.Type, c.v, err, c.good)
		}
	}
}

func TestPrefixForADeployment(t *testing.T) {
	p := connectors.PrefixFor("3F2A9C1E-7b4d-4e2a-9c1e-000000000001")
	if p != "c3f2a9c1e" || !connectors.ValidPrefix(p) {
		t.Fatalf("prefix %q", p)
	}
	for _, bad := range []string{"", "3f2a9c1e", "c3f2a9c1", "c3F2A9C1E", "c3f2a9c1e-x"} {
		if connectors.ValidPrefix(bad) {
			t.Errorf("%q passed ValidPrefix", bad)
		}
	}
}

func TestSecretsNeverPrint(t *testing.T) {
	s := connectors.Secrets{"token": "super-secret-value"}
	for _, out := range []string{s.String(), s.GoString()} {
		if out != connectors.Redacted {
			t.Fatalf("printed %q", out)
		}
	}
	b, _ := s.MarshalJSON()
	if string(b) != `"[redacted]"` {
		t.Fatalf("json %s", b)
	}
}
