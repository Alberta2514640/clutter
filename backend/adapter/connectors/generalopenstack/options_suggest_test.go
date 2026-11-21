package generalopenstack

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

// The lists of a public cloud without DevStack's defaults, shaped like
// VEXXHOST's ca-ymq-1 on 2026-11-20. The flavors are its real ones up to
// 4 GB (name, vCPUs, RAM and disk, from `openstack flavor list`); every image
// property and upload time is a stand-in. The arm64 Ubuntu twin is uploaded
// later than the x86_64 one, and the Kubernetes images last, so the picker
// must refuse them on purpose rather than by age.
const (
	vexxhostFlavors = `{"flavors": [
	  {"id": "1", "name": "v1-standard-1", "vcpus": 2, "ram": 1024, "disk": 40},
	  {"id": "2", "name": "v2-highcpu-1", "vcpus": 1, "ram": 1024, "disk": 0},
	  {"id": "3", "name": "v2-highcpu-2", "vcpus": 2, "ram": 2048, "disk": 0},
	  {"id": "4", "name": "v3-starter-1", "vcpus": 1, "ram": 2048, "disk": 10},
	  {"id": "5", "name": "v1-standard-2", "vcpus": 4, "ram": 2048, "disk": 50},
	  {"id": "6", "name": "v2-highcpu-4", "vcpus": 4, "ram": 4096, "disk": 0},
	  {"id": "7", "name": "v2-standard-1", "vcpus": 1, "ram": 4096, "disk": 0},
	  {"id": "8", "name": "v3-standard-1", "vcpus": 1, "ram": 4096, "disk": 20},
	  {"id": "9", "name": "v1-standard-4", "vcpus": 4, "ram": 4096, "disk": 100},
	  {"id": "10", "name": "v3-starter-2", "vcpus": 2, "ram": 4096, "disk": 20}]}`
	vexxhostImages = `{"images": [
	  {"id": "a", "name": "Ubuntu 24.04.2 LTS (x86_64) [2025-04-03]", "status": "active", "visibility": "public",
	   "os_distro": "ubuntu", "os_version": "24.04", "architecture": "x86_64", "created_at": "2025-04-03T10:00:00Z"},
	  {"id": "b", "name": "Ubuntu 24.04.2 LTS (arm64) [2025-04-03]", "status": "active", "visibility": "public",
	   "os_distro": "ubuntu", "os_version": "24.04", "architecture": "aarch64", "created_at": "2025-04-03T11:00:00Z"},
	  {"id": "c", "name": "Ubuntu 22.04.5 LTS (x86_64) [2025-03-27]", "status": "active", "visibility": "public",
	   "os_distro": "ubuntu", "os_version": "22.04", "architecture": "x86_64", "created_at": "2025-03-27T10:00:00Z"},
	  {"id": "d", "name": "ubuntu-2204-kube-v1.30.4", "status": "active", "visibility": "public",
	   "os_distro": "ubuntu", "architecture": "x86_64", "created_at": "2025-06-01T10:00:00Z"},
	  {"id": "e", "name": "ubuntu-2204-kube-v1.29.8", "status": "active", "visibility": "public",
	   "os_distro": "ubuntu", "architecture": "x86_64", "created_at": "2025-05-01T10:00:00Z"},
	  {"id": "f", "name": "debian-12-generic-amd64-20241201", "status": "active", "visibility": "public",
	   "os_distro": "debian", "architecture": "x86_64", "created_at": "2024-12-01T10:00:00Z"},
	  {"id": "g", "name": "Fedora CoreOS 34", "status": "active", "visibility": "public",
	   "os_distro": "fedora-coreos", "architecture": "x86_64", "created_at": "2021-06-01T10:00:00Z"}]}`
	// The same images with no properties at all, so the names alone decide.
	vexxhostImagesByName = `{"images": [
	  {"id": "a", "name": "Ubuntu 24.04.2 LTS (x86_64) [2025-04-03]", "status": "active", "created_at": "2025-04-03T10:00:00Z"},
	  {"id": "b", "name": "Ubuntu 24.04.2 LTS (arm64) [2025-04-03]", "status": "active", "created_at": "2025-04-03T11:00:00Z"},
	  {"id": "c", "name": "Ubuntu 22.04.5 LTS (x86_64) [2025-03-27]", "status": "active", "created_at": "2025-03-27T10:00:00Z"},
	  {"id": "d", "name": "ubuntu-2204-kube-v1.30.4", "status": "active", "created_at": "2025-06-01T10:00:00Z"},
	  {"id": "f", "name": "debian-12-generic-amd64-20241201", "status": "active", "created_at": "2024-12-01T10:00:00Z"},
	  {"id": "g", "name": "Fedora CoreOS 34", "status": "active", "created_at": "2021-06-01T10:00:00Z"}]}`
	vexxhostNetworks = `{"networks": [{"id": "n", "name": "public", "status": "ACTIVE", "router:external": true}]}`
)

const vexxhostUbuntu = "Ubuntu 24.04.2 LTS (x86_64) [2025-04-03]"

func TestOptionsSuggestForAPublicCloud(t *testing.T) {
	want := map[string]string{
		"image":         vexxhostUbuntu,
		"flavor_small":  "v3-starter-1",
		"flavor_medium": "v3-starter-1",
		"flavor_large":  "v3-standard-1",
		// "public" is the default and VEXXHOST has it: no suggestion.
	}
	for name, images := range map[string]string{"with image properties": vexxhostImages, "by name alone": vexxhostImagesByName} {
		t.Run(name, func(t *testing.T) {
			o := newOSCloud(t)
			o.flavors, o.images, o.networks = vexxhostFlavors, images, vexxhostNetworks
			got, err := newConnector(nil, AllowPrivateTargets()).Options(context.Background(), conn(o, goodSecret))
			if err != nil {
				t.Fatal(err)
			}
			if !maps.Equal(got.Suggested, want) {
				t.Fatalf("suggested %v, want %v", got.Suggested, want)
			}
			for setting, v := range got.Suggested {
				if !slices.Contains(values(got.Settings[setting]), v) {
					t.Errorf("%s suggests %q, which is not one of its choices", setting, v)
				}
				if !validCloudName(v) {
					t.Errorf("%s suggests %q, which the renderer refuses", setting, v)
				}
			}
		})
	}
}

func TestOptionsSuggestionsAreCachedAsCopies(t *testing.T) {
	o := newOSCloud(t)
	o.flavors, o.images, o.networks = vexxhostFlavors, vexxhostImages, vexxhostNetworks
	c := newConnector(nil, AllowPrivateTargets())
	first, err := c.Options(context.Background(), conn(o, goodSecret))
	if err != nil {
		t.Fatal(err)
	}
	first.Suggested["image"] = "changed"
	again, err := c.Options(context.Background(), conn(o, goodSecret))
	if err != nil {
		t.Fatal(err)
	}
	if again.Suggested["image"] != vexxhostUbuntu {
		t.Fatalf("the cache was changed through a caller's copy: %v", again.Suggested)
	}
}

func TestSuggestImage(t *testing.T) {
	x86 := func(name, created string) image {
		return image{Name: name, Status: "active", OSDistro: "ubuntu", Architecture: "x86_64", CreatedAt: propString(created)}
	}
	cases := []struct {
		name   string
		images []image
		want   string
	}{
		{"the default is the start of a name", []image{{Name: "cirros-0.6.3-x86_64-disk", Status: "active"}, x86("Ubuntu 24.04 LTS", "")}, ""},
		{"no Ubuntu image", []image{{Name: "debian-12-generic-amd64", Status: "active", OSDistro: "debian", Architecture: "x86_64"}}, ""},
		{"an LTS release before a newer interim one", []image{x86("Ubuntu 25.10 (x86_64)", "2025-10-10T00:00:00Z"), x86("Ubuntu 24.04.3 LTS (x86_64)", "2025-08-01T00:00:00Z")}, "Ubuntu 24.04.3 LTS (x86_64)"},
		{"an even year's April release is LTS without the word", []image{x86("ubuntu-25.04-x86_64", "2025-05-01T00:00:00Z"), x86("ubuntu-24.04-x86_64", "2024-05-01T00:00:00Z")}, "ubuntu-24.04-x86_64"},
		{"the newer point release", []image{x86("Ubuntu 24.04.1 LTS", "2025-01-01T00:00:00Z"), x86("Ubuntu 24.04.2 LTS", "2024-01-01T00:00:00Z")}, "Ubuntu 24.04.2 LTS"},
		{"the release from os_version when the name has none", []image{
			{Name: "ubuntu-jammy", Status: "active", OSVersion: "22.04", Architecture: "amd64"},
			{Name: "ubuntu-noble", Status: "active", OSVersion: "24.04", Architecture: "amd64"}}, "ubuntu-noble"},
		{"the same release: the newest upload", []image{x86("Ubuntu 24.04 LTS a", "2025-01-01T00:00:00Z"), x86("Ubuntu 24.04 LTS b", "2025-03-01T00:00:00Z")}, "Ubuntu 24.04 LTS b"},
		{"os_distro ubuntu with another name", []image{x86("noble-server-cloudimg", "")}, "noble-server-cloudimg"},
		{"an arm64 architecture property", []image{{Name: "Ubuntu 24.04 LTS", Status: "active", Architecture: "aarch64"}}, ""},
		{"another architecture", []image{{Name: "Ubuntu 24.04 LTS", Status: "active", Architecture: "ppc64le"}}, ""},
		{"an x86_64 property but an arm64 name", []image{x86("Ubuntu 24.04 LTS (arm64)", "")}, ""},
		{"no architecture said anywhere", []image{{Name: "Ubuntu 24.04 LTS", Status: "active"}}, ""},
		{"amd64 in the name", []image{{Name: "ubuntu-24.04-server-cloudimg-amd64", Status: "active"}}, "ubuntu-24.04-server-cloudimg-amd64"},
		{"a Kubernetes image", []image{x86("Ubuntu-2404-Kube-v1.31", "")}, ""},
		{"an image that is not active", []image{{Name: "Ubuntu 24.04 LTS (x86_64)", Status: "queued"}}, ""},
		{"a name the renderer refuses", []image{x86("Ubuntu 24.04 LTS {x86_64}", "")}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			names := make([]string, len(tc.images))
			for i, im := range tc.images {
				names[i] = im.Name
			}
			if got := suggestImage(nameChoices(names), tc.images); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSuggestFlavors(t *testing.T) {
	cases := []struct {
		name    string
		flavors []flavor
		want    map[string]string
	}{
		{"DevStack's defaults", []flavor{
			{Name: "m1.tiny", VCPUs: 1, RAM: 512, Disk: 1}, {Name: "m1.small", VCPUs: 1, RAM: 2048, Disk: 20}, {Name: "m1.medium", VCPUs: 2, RAM: 4096, Disk: 40}}, nil},
		{"only the missing defaults", []flavor{
			{Name: "m1.tiny", VCPUs: 1, RAM: 512, Disk: 1}, {Name: "c2", VCPUs: 2, RAM: 2048, Disk: 20}, {Name: "c4", VCPUs: 2, RAM: 4096, Disk: 20}},
			map[string]string{"flavor_medium": "c2", "flavor_large": "c4"}},
		{"ties go to fewer vCPUs, then the name", []flavor{
			{Name: "b", VCPUs: 1, RAM: 1024, Disk: 10}, {Name: "a", VCPUs: 1, RAM: 1024, Disk: 10}, {Name: "c", VCPUs: 2, RAM: 2048, Disk: 10}, {Name: "d", VCPUs: 1, RAM: 2048, Disk: 10}},
			map[string]string{"flavor_small": "a", "flavor_medium": "d"}},
		{"a flavor with a disk before a smaller one without", []flavor{
			{Name: "volume-only-1", VCPUs: 1, RAM: 1024, Disk: 0}, {Name: "disk-2", VCPUs: 1, RAM: 2048, Disk: 10}, {Name: "volume-only-4", VCPUs: 2, RAM: 4096, Disk: 0}},
			map[string]string{"flavor_small": "disk-2", "flavor_medium": "disk-2", "flavor_large": "volume-only-4"}},
		{"too small and no vCPU", []flavor{{Name: "tiny", VCPUs: 1, RAM: 512, Disk: 1}, {Name: "zero", VCPUs: 0, RAM: 8192, Disk: 10}}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := suggestFlavors(tc.flavors); !maps.Equal(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSuggestNetwork(t *testing.T) {
	ch := func(names ...string) []connectors.Choice { return nameChoices(names) }
	cases := []struct {
		name    string
		offered []connectors.Choice
		total   int
		want    string
	}{
		{"the one external network", ch("ext-net"), 1, "ext-net"},
		{"the default", ch("public"), 1, ""},
		{"several", ch("ext-a", "ext-b"), 2, ""},
		{"one usable name among two networks", ch("ext-a"), 2, ""},
		{"none", ch(), 0, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := suggestNetwork(tc.offered, tc.total); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
