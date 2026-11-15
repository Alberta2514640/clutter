package connectors

// Problem codes for what every connector can report. A code a connector alone
// reports stays in that connector; B's capability check owns its own codes.
const (
	// CodeInvalidValue is a field or setting value the cloud cannot take.
	CodeInvalidValue = "invalid_value"
	// CodeMissingValue is a required field or setting left empty.
	CodeMissingValue = "missing_value"
	// CodeDuplicateName is a block whose cloud-side name another block has.
	CodeDuplicateName = "duplicate_name"
	// CodeDeploymentInvalid is a deployment whose id, prefix or region the
	// cloud cannot take.
	CodeDeploymentInvalid = "deployment_invalid"
	// CodeRenderRefused is rendered Terraform the shared runner refuses.
	CodeRenderRefused = "render_refused"
	// CodeVolumeTwoVMs is a volume connected to more than one VM.
	CodeVolumeTwoVMs = "volume_on_two_vms"
	// CodeAddressCycle is a circle of VMs that get each other's address.
	CodeAddressCycle = "vm_address_cycle"
	// CodePeerUnreachable is a VM to VM edge whose two VMs are in networks
	// that cannot reach each other.
	CodePeerUnreachable = "vm_peer_unreachable"
	// CodeCrossNeedsPublic is a VM another cloud reaches that is not public.
	CodeCrossNeedsPublic = "cross_needs_public"
	// CodeCrossInvalid is a cross-cloud edge this cloud cannot serve.
	CodeCrossInvalid = "cross_invalid"
	// CodeCost is a note (Advisor) naming what a deployment costs beyond its
	// blocks, such as a gateway billed by the hour.
	CodeCost = "cost"
)

// Messages every connector shows for the same event.
const (
	// MsgCannotBuild is a run's or a preview's failure when the deployment
	// has problems.
	MsgCannotBuild = "The diagram has blocks or settings this cloud cannot build. Check the deployment's compatibility."
	// MsgRenderRefused is CodeRenderRefused's message.
	MsgRenderRefused = "Clutter refused the Terraform it rendered for this diagram (it holds a feature Clutter does not run, such as a provisioner or a module). Change the diagram and try again."
)

// ErrCannotBuild is the user error for MsgCannotBuild.
var ErrCannotBuild = &UserError{Message: MsgCannotBuild}
