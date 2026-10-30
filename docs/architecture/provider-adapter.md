# Provider adapter layer

Status: proposed, 2026-10-26. Nothing here is built yet. The decisions it rests on are ADR-001 to ADR-005 in `docs/decisions.md`; the Patrii facts are in `docs/research/2026-10-26-patrii-cloud.md`; the current code is `system.md`.

## The goal

A central, standard layer with two faces:

- **Inbound, the standard API:** a versioned contract that people and other products call to use Clutter, without learning React Flow, Supabase webhooks or AWS role ARNs.
- **Outbound, target adapters:** one adapter per cloud behind a provider-neutral core. Adding a cloud means writing an adapter, not editing the core.

Clutter owns intent (what the customer wants); clouds own resources. Callers talk to Clutter; Clutter talks to clouds. Patrii is a target cloud, not a host for the standard and not a caller: it publishes no inbound plugin and has no diagram, plan or apply resource. A Patrii-side integration that only wants inventory (as Patrii's Lynx DCIM connector sketches) is a read through the same target adapter.

This half-split and the five standard objects come from a second, independent research run on the same date (`reports/PatriCloud Clutter adapter layer.md`, with notes in `research_notes/`), checked against the code by the lead.

## Part 1: the standard API (inbound)

### Principles

1. **Clutter's model is the contract.** Callers send intent; adapters turn it into each cloud's resources. A caller never sends an ARN, a Terraform block or a Patrii path as the standard's vocabulary.
2. **Every caller resolves to a member.** A service credential is a new way to mint an identity that still passes the one membership check (`CheckOrganizationMembershipPSQL`), never a bypass of it. Google sign-in becomes one client of the same rule.
3. **Compile and run are one authenticated sequence.** A public webhook that compiles, and a diagram save that does not, cannot be the contract.
4. **The standard is the only versioned surface.** `/v1`, one error envelope, one id style, an idempotency key on every create. Neither Clutter nor Patrii has idempotency today, and Patrii's own changelog records a retried create starting duplicate billable servers.
5. **Today's trust boundary is not standardized.** The admin-sized customer role and the open compiler are fixed before any other product is invited in.

### The standard objects

| Object | What it is | Clutter today |
|---|---|---|
| Organization | the tenant | organization CRUD; the creator is the only member; no invite route (the frontend members API is a mock) |
| Workspace | Clutter's "project", renamed so it does not collide with a cloud's project (Patrii and OpenStack call their tenant a project) | project CRUD; name column 32 characters while the API schema allows 100 |
| Cloud account | a credentialed target, with a provider field | the AWS-named role table and its two-step CloudFormation link |
| Desired state | the diagram, as a neutral graph the standard translates to the canvas document | diagram CRUD; save does not compile |
| Run | apply or destroy, one resource to poll, events to subscribe to | Terraform runs in `diagram_deployment_logs`, Ansible in `jobs`: two tables, polled logs, no events |
| Catalog | the resource types a provider accepts | the seed table, the generator switch and the config panel: three lists that drift |

Not in the standard until a handler exists: member invites and roles, token refresh and revoke, `plan` (the frontend type offers it; the backend enum is apply and destroy only), and diagram history (written, never exposed).

### Where it runs

The standard is one process (the HTTP API, the identity mint, the Postgres model and the compiler); the runner is a second process that sits where it can reach the target cloud and the state. Hosting moves custody, not the contract: whoever runs the runner holds the target credentials and the state. See ADR-004 for the default and Part 2 for state placement.

## Part 2: target adapters (outbound)

### Principles

1. **Capability-based, not lowest common denominator.** Each provider declares the resource types it supports, with their fields, defaults and allowed values. The canvas shows only those types, and validation refuses an unsupported node before generation. A lowest-common-denominator layer would remove Lambda, API Gateway and DynamoDB from every user to make room for a cloud that lacks them.
2. **Provider-native nodes first.** A node is "Patrii VPS" or "AWS EC2", not an abstract "compute". Generic nodes that map to each provider can come later as a layer on top, once two providers exist to learn from.
3. **Terraform stays the engine.** Each adapter emits HCL for an existing Terraform provider. Clutter does not call cloud APIs to create resources itself. Plan, apply and state stay in Terraform, as in Terraform's own core and provider split.
4. **One owner per resource definition.** The provider's resource manifest, in Go, is the single source. The seed table, the palette, the config panel options and the request schemas are read from it (through the API), not maintained beside it.
5. **The core never names a provider.** It asks the registry for the diagram's provider and calls the interface. A `switch provider` in the core is a defect.
6. **Credentials are the provider's shape, stored the core's way.** Each provider declares its credential schema and how to verify it. The core stores a reference to an encrypted secret, never the secret in a plain column, and never logs it.
7. **State lives with the customer's cloud where the provider allows it.** For Patrii, the Terraform state for a customer's diagram belongs in that customer's own object storage, with locking, not in Clutter's US bucket.

### The interface (first cut)

```go
// Package providers. Each cloud implements Provider and registers itself.
type Provider interface {
    ID() string                                   // "aws", "patrii"
    Resources() []ResourceSpec                    // the manifest: types, fields, defaults, allowed values, icon
    Validate(d Diagram) []GenerationError         // refuses unsupported nodes and bad values before rendering
    Render(d Diagram) (Files, error)              // resources, edges (IAM or security rules), provider block, outputs
    Regions() (list []string, def string)
    Credentials() CredentialSpec                  // what the customer supplies, and how the UI asks for it
    VerifyAccount(ctx context.Context, c Credential) error // proves a link works before it is saved
    RunnerEnv(c Credential, region string) (map[string]string, error) // what the deployer needs, scoped to one run
    StateBackend(ref DiagramRef) (Files, error)   // the backend block, with locking
}
```

Remote command execution (today SSM for Ansible) is left out of the first cut. Patrii has no SSM equivalent; the options are cloud-init at boot, SSH through a bastion or VPN, or a pull agent. It becomes a separate optional interface once one of those is chosen.

### What each method replaces today

| Method | Replaces |
|---|---|
| `Resources` | the type constants, the label switch, the seed rows, the palette list and the config panel option lists |
| `Validate` | per-generator `ValidateVariables`, plus the input validation that is missing today |
| `Render` | the generator files, the template prefix, IAM from edges, the fixed `provider "aws"` block in the writer |
| `Regions` | the region list in `generic`, the schema enum and the frontend's own list |
| `Credentials`, `VerifyAccount` | the CloudFormation link and role-ARN handlers, which today store the ARN unverified |
| `RunnerEnv` | the role ARN and external id passed to the deployer, and the STS step in its script |
| `StateBackend` | the local state file synced to S3 without a lock |

### What Patrii needs, and the blocker

Patrii does not yet publish a documented customer API: its REST API docs and its own Terraform provider are both "coming soon", and no public OpenStack endpoint was found. `api.patrii.cloud` is live and serves the console (bearer session tokens, project-scoped API keys, unversioned paths such as `/compute/servers`), but with no OpenAPI file and no version to pin. The homepage promises "full API compatibility" with Terraform and Ansible but never names Nova, Neutron or Keystone. There are three paths, and the choice is open (ADR-003):

- **Path A (preferred):** Patrii exposes public OpenStack endpoints (Keystone v3 with application credentials, Nova, Neutron, Cinder, Octavia, Designate) and an S3 endpoint. The adapter emits `terraform-provider-openstack` HCL and uses a Keystone application credential with access rules and an expiry as the linked-account credential. This is the most standard path and also works for any other OpenStack cloud.
- **Path B:** Patrii ships its `patrii/patrii` Terraform provider. The adapter emits that provider's HCL and uses a Patrii project API key.

- **Path C:** a Patrii adapter that calls the console's REST API directly (the second research run's recommendation). It works today, but the write schemas are not in the console bundle and would have to be captured from a real console session, and any console release can break it. If chosen, it needs idempotency on Clutter's side and a contract test suite run against a Patrii test project, and Clutter loses Terraform's plan, state and drift handling for Patrii (ADR-002).

The lead's recommendation is to ask Patrii first and build Path A or B; Path C only if Patrii confirms its REST API will be the supported surface and publishes its OpenAPI file. What would change this: Patrii stating that OpenStack endpoints will never be public and no Terraform provider is planned.

First Patrii resources, once there is an API: VPS (server with keypair, security group, floating IP, cloud-init), network and router, block volume, object storage bucket. Load balancer, DNS and managed databases follow. Lambda, API Gateway and DynamoDB have no Patrii equivalent and are simply not in the Patrii manifest.

## Phases

| Phase | Work | Done when |
|---|---|---|
| 0. Make the base safe | Authenticate `POST /terraform-engine` and make save-then-compile one authenticated sequence, validate and escape every value that reaches a template, ownership checks on every child id, narrow the customer role, verify a role before storing it, a lock on state. Make `go test ./...` green offline and add Go build, vet and test to CI. | The open defects in `.claude/findings/NOW.md` are closed with tests, and CI fails on a Go break. |
| 0b. The standard API, first slice | `/v1` in front of the existing handlers: service credentials that resolve to a member, one error envelope, idempotency keys, one run resource over both run tables, the catalog served from the generator. | Another program can create a workspace and desired state, link an AWS account and run apply with a service credential, and the existing UI still works. |
| 1. AWS behind the interface | Create `providers` with the interface and a registry; move the AWS generators, templates, IAM and provider block behind `providers/aws` with no behaviour change. Serve the manifest from the resources API and have the palette and config panel read it. | Generated HCL for a set of fixture diagrams is byte-identical before and after; the core has no AWS import outside `providers/aws`. |
| 2. Accounts and runs go provider-neutral | A `cloud_accounts` table with a provider column and an encrypted credential reference, replacing the AWS-named table; runs carry the provider; the deployer image gets its credentials and backend from `RunnerEnv` and `StateBackend`. | An AWS link and deploy work end to end through the new table. |
| 3. Patrii adapter | Path A or B, depending on Patrii's answer. Manifest, render, credential verify, state in the customer's Patrii object storage. | A diagram with a VPS, network and volume deploys to a Patrii test project and destroys cleanly. |
| 4. Hosting model | Decide SaaS on AWS, Clutter hosted on Patrii, or BYOC runners (ADR-004 holds the current default). | Decided with the client, recorded as an ADR. |

## Questions that block or shape the plan

1. Patrii: the five questions at the end of the research note (API access, key scope, S3 endpoint, remote execution, partner terms).
2. The client: is Patrii a customer target, a hosting target for Clutter itself, or both? This decides whether phase 4 comes before phase 3.
3. The team: does a diagram target exactly one provider (default), or can one diagram span clouds?
