package connectors

import (
	"strings"
	"testing"
)

// displayManifest is a well-formed manifest with display data.
func displayManifest() Manifest {
	m := Manifest{Connector: "test-cloud", Title: "Test cloud", Version: "1", Display: Display{ShortName: "Test", Logo: "logo"}}
	for _, k := range StandardBlocks() {
		m.Blocks = append(m.Blocks, StandardBlock(k, string(k)).WithDisplay("Service "+string(k), "icon-"+strings.ReplaceAll(string(k), "_", "-")))
	}
	return m
}

func TestCheckAcceptsDisplay(t *testing.T) {
	if err := displayManifest().Check(); err != nil {
		t.Fatalf("well-formed display refused: %v", err)
	}
	// Display is optional to Check; the contract tests require it of a
	// real connector.
	m := displayManifest()
	m.Display = Display{}
	for i := range m.Blocks {
		m.Blocks[i].Display = nil
	}
	if err := m.Check(); err != nil {
		t.Fatalf("a manifest without display refused: %v", err)
	}
}

func TestCheckRefusesBadDisplay(t *testing.T) {
	cases := map[string]struct {
		change func(*Manifest)
		want   string
	}{
		"block display without a name": {
			change: func(m *Manifest) { m.Blocks[0].Display = &BlockDisplay{Icon: "ec2"} },
			want:   `block "vm" has a display without a name`,
		},
		"block display name too long": {
			change: func(m *Manifest) { m.Blocks[0].Display.Name = strings.Repeat("x", MaxDisplayName+1) },
			want:   "longer than",
		},
		"icon key that is a path": {
			change: func(m *Manifest) { m.Blocks[0].Display.Icon = "../aws/ec2" },
			want:   "icon key",
		},
		"icon key with an extension": {
			change: func(m *Manifest) { m.Blocks[0].Display.Icon = "ec2.svg" },
			want:   "icon key",
		},
		"removed block with a display": {
			change: func(m *Manifest) {
				m.Blocks[4] = BlockSpec{Kind: KindBucket, Treatment: TreatmentRemoved, Title: "Bucket", Display: &BlockDisplay{Name: "S3"}}
			},
			want: `removed block "bucket" declares a display`,
		},
		"logo without a short name": {
			change: func(m *Manifest) { m.Display.ShortName = "" },
			want:   "logo but no short name",
		},
		"logo key that is a path": {
			change: func(m *Manifest) { m.Display.Logo = "x/y" },
			want:   "logo key",
		},
		"short name too long": {
			change: func(m *Manifest) { m.Display.ShortName = strings.Repeat("x", MaxShortName+1) },
			want:   "short name is longer",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m := displayManifest()
			c.change(&m)
			err := m.Check()
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Check() = %v, want an error containing %q", err, c.want)
			}
		})
	}
}

func TestWithDisplayDoesNotShareState(t *testing.T) {
	a := StandardBlock(KindVM, "VM")
	b := a.WithDisplay("EC2", "ec2")
	if a.Display != nil {
		t.Fatal("WithDisplay changed the block it was called on")
	}
	if b.Display == nil || b.Display.Name != "EC2" || b.Display.Icon != "ec2" {
		t.Fatalf("WithDisplay = %+v", b.Display)
	}
}
