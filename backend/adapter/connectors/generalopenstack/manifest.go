package generalopenstack

import (
	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/api/generic/hclsafe"
)

// Diagram fields. These are the portable fields every connector declares the
// same way, so one diagram passes every manifest; anything OpenStack-only is a
// deployment setting below.
var (
	fieldSize   = connectors.StandardField(connectors.KindVM, "size")
	fieldPublic = connectors.StandardField(connectors.KindVM, "public")
	fieldCIDR   = connectors.StandardField(connectors.KindNetwork, "cidr")
	// fieldIngress lists what may come in, such as "tcp/22,tcp/443,udp/1000-2000,icmp".
	fieldIngress    = connectors.StandardField(connectors.KindFirewall, "ingress")
	fieldSourceCIDR = connectors.StandardField(connectors.KindFirewall, "source_cidr")
	fieldSizeGB     = connectors.StandardField(connectors.KindVolume, "size_gb")
)

// sizes are the portable VM sizes; each maps to a flavor through a setting.
var sizes = hclsafe.NewAllowList(fieldSize.Allowed...)

// Deployment settings. The defaults suit DevStack.
var (
	// settingImage is matched as the start of an image name, and the newest
	// match wins, so "cirros" finds DevStack's versioned cirros image.
	settingImage           = connectors.FieldSpec{Name: "image", Title: "Image (a name, or the start of one; the newest match is used)", Type: connectors.FieldString, Required: true, Default: "cirros"}
	settingExternalNetwork = connectors.FieldSpec{Name: "external_network", Title: "External network (routers and public addresses use it)", Type: connectors.FieldString, Required: true, Default: "public"}
	settingFlavorSmall     = connectors.FieldSpec{Name: "flavor_small", Title: "Flavor for small VMs", Type: connectors.FieldString, Required: true, Default: "m1.tiny"}
	settingFlavorMedium    = connectors.FieldSpec{Name: "flavor_medium", Title: "Flavor for medium VMs", Type: connectors.FieldString, Required: true, Default: "m1.small"}
	settingFlavorLarge     = connectors.FieldSpec{Name: "flavor_large", Title: "Flavor for large VMs", Type: connectors.FieldString, Required: true, Default: "m1.medium"}
)

// flavorSetting maps a VM size to the setting that names its flavor.
var flavorSetting = map[string]connectors.FieldSpec{
	"small":  settingFlavorSmall,
	"medium": settingFlavorMedium,
	"large":  settingFlavorLarge,
}

// Manifest implements connectors.Connector. It lists no regions: an
// OpenStack cloud names its own, and VerifyConnection checks the connection's
// region against the cloud's catalog.
func (*Connector) Manifest() connectors.Manifest {
	return connectors.Manifest{
		Connector: ID,
		Title:     "OpenStack",
		Version:   "1",
		// The logo is a neutral tile with the name, and the blocks keep their
		// neutral icons: the OpenStack logo needs a commercial trademark
		// license (frontend/clutter/public/clouds/README.md).
		Display: connectors.Display{ShortName: "OpenStack", Logo: "logo"},
		Blocks: []connectors.BlockSpec{
			connectors.StandardBlock(connectors.KindVM, "VM").WithDisplay("Server (Nova)", ""),
			connectors.StandardBlock(connectors.KindNetwork, "Network").WithDisplay("Network (Neutron)", ""),
			connectors.StandardBlock(connectors.KindFirewall, "Firewall").WithDisplay("Security group", ""),
			connectors.StandardBlock(connectors.KindVolume, "Volume").WithDisplay("Volume (Cinder)", ""),
			{Kind: connectors.KindBucket, Treatment: connectors.TreatmentRemoved, Title: "Bucket"},
			{Kind: connectors.KindFunction, Treatment: connectors.TreatmentRemoved, Title: "Function"},
			{Kind: connectors.KindTable, Treatment: connectors.TreatmentRemoved, Title: "Table"},
			{Kind: connectors.KindHTTPAPI, Treatment: connectors.TreatmentRemoved, Title: "HTTP API"},
		},
		Edges: []connectors.EdgeSpec{
			{Source: connectors.KindVM, Target: connectors.KindNetwork, Meaning: "the VM is attached to the network"},
			{Source: connectors.KindFirewall, Target: connectors.KindVM, Meaning: "the firewall rules apply to the VM"},
			{Source: connectors.KindVolume, Target: connectors.KindVM, Meaning: "the volume is attached to the VM"},
			// Across clouds this is a split deployment's address edge
			// (connectors.CrossClass).
			{Source: connectors.KindVM, Target: connectors.KindVM, Meaning: "the VM gets the other VM's address when it first starts"},
		},
		Settings: specs(settingImage, settingExternalNetwork, settingFlavorSmall, settingFlavorMedium, settingFlavorLarge),
	}
}

// specs copies field specs, so a caller that changes the manifest it was
// given cannot change the connector's own lists.
func specs(fs ...connectors.FieldSpec) []connectors.FieldSpec {
	out := make([]connectors.FieldSpec, len(fs))
	for i, f := range fs {
		f.Allowed = append([]string(nil), f.Allowed...)
		out[i] = f
	}
	return out
}
