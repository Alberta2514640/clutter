# Implementation plan: from the design to a working B0

The order to build `system-b-adapter-layer.md` in, what each stage changes, how it is tested, and when it is done. The design says what B is and why; this file says how it gets built. Each stage's status is in `.claude/findings/NOW.md` § The order. The design was checked before any code on 2026-11-13: an independent review, an AWS template spike and a DevStack test (`docs/research/openstack/2026-11-13-devstack-and-block-mapping.md`).

## How the work runs

- **Stages 1 to 6 are built on one implementation branch** and reach `main` by one pull request. Each stage is reviewed as it lands (every check, the smoke test, and a fresh reviewer where the next point says). Stage 7 waits for the credentials. The fork's `main` then goes to the client's repo by pull request.
- A builder writes the stage; the person reviewing runs every check and the smoke test themselves; a fresh reviewer attacks any stage that touches sign-in, tenancy, credentials or generated code. Findings are resolved before merge.
- The person commits, pushes, merges and deploys. No session deploys.
- **The docs move with the code.** Each stage's pull request updates the owner doc of every fact it changes (`docs/README.md` says which), its row in `.claude/findings/NOW.md`, and the folder README's expected results. A stage is not done while a doc still describes the old state.
- A's 85 CRLF files (50 of them Go) keep their line endings (`git diff --stat` before committing).

## Testing without the client's credentials

Nothing waits for the client except stage 7.

| Fully testable on a laptop | How |
| --- | --- |
| A, except real deploys | A2, the smoke test, Go and frontend tests, `terraform validate` |
| B's API, sign-in, tables, connections, credential encryption | B's Go tests, and smoke checks through A2 |
| The connector contract | a mock connector and the shared contract tests, in CI |
| Real OpenStack deploys | General's OpenStack connector against DevStack |
| B's AWS output | `terraform validate`, the golden test for A's template move, and plan, apply and destroy on LocalStack |
| The whole editor flow | a browser against A2; AWS deployments apply on LocalStack when it runs, and otherwise stop at the AWS call |
| CloudFormation and IAM | `cfn-lint`, `terraform validate` |

Left for the credentials: real AWS deploys by A, hosting world 1 (B on Fargate, S3 state, the website), the client's Google sign-in, and the hosted demo. B's deploys into real AWS and OpenStack accounts, and the role link end to end, ran from a laptop on 2026-11-20 (`docs/reports/2026-11-20-real-cloud-verification.pdf`).

## Stage 0: housekeeping

The decisions are recorded (ADR-013 to ADR-017), the design was checked on 2026-11-13, and the questions for the client are in `.claude/findings/NOW.md` § What we need from the client. **Done when:** the fork's `main` has stages 1 to 6 and reaches the client's repo by pull request.

## Stage 1: make A safe

- **Phase 0** (`.claude/findings/NOW.md` § Phase 0), in this order: item 6 (Go tests offline, and CI runs every package), 1 (authenticate generation, generic error bodies), 3 (allow-lists and HCL escaping, including `${` and `%{`, through one shared escape function that B reuses), 5 (one shared ownership helper in `generic`), 10 (A's runner gets no database string, one task per run), 2 (row-level security), 4 (the Google audience check), 6b to 6d; items 7 to 9 in code only.
- **A's template code moves out of `internal`** to `backend/api/generic/tftemplate/`, with a loader that reads S3 for A and disk for B and tests. A golden test that renders fixture diagrams from disk is written first, and its output stays byte-identical across the move.
- **JWT verification moves** from the authorizer's `internal` package to `generic`, so A's authorizer and B share one implementation.
- **Region:** the generator's fixed `us-west-2` reads the environment's region (`ca-central-1` in the cloud).
- **Manual cloud steps become code**, so stage 7 is mostly filling in config: the API Gateway stage, the Google client id from configuration, the API address passed into the frontend build, versioned runner image tags, the platform state backend chosen at `terraform init`, and the Supabase webhook replaced by item 1's save-then-generate step.
- **Done when:** every phase 0 item that can be checked locally is done, the golden test passes, smoke is at 0 fail, and CI runs all Go packages.

## Stage 2: a test cloud on the laptop

DevStack in a Lima virtual machine, as in the research note. Proven on one laptop on 2026-11-13. The team gets a short start-and-stop guide with stage 4. **Done when:** the test configuration applies and destroys a VM, network, firewall and volume.

## Stage 3: B's core

- A new Go module, `backend/adapter` (Go 1.25.3, `require` plus `replace ../api` to import `generic`): one HTTP server with a connection pool.
- Tables in a new idempotent `backend/storage-sql/adapter-tables.sql`: connections (organization, type, name, region, status, secret reference, role-template version), deployments (diagram, connection, settings, exclusions, status), runs (deployment, action, status, diagram history version, times, log key). Loaded by A2 as `30-adapter-tables.sql` and applied by B at start-up.
- Sign-in through `generic`'s JWT verification. Every deployment action checks that the caller is a member of the organization, and that the diagram and the connection belong to it.
- The `connectors` package: the interface from the design doc (taking a `Deployment`), a registry, a mock connector, and the shared contract tests.
- The translator, the capability check (with per-deployment exclusions), the job runner, the isolated run task (ADR-017), the Terraform runner (state per deployment, locked), and the credential store (a local key in A2; Secrets Manager in the cloud).
- API `/v1`: connections (create, verify, list, delete), deployments (create, compatibility), runs (plan, apply, destroy, status, logs), catalog (manifests). Deletes refuse while deployments are live.
- A2: B on `127.0.0.1:8790`, started and stopped by `npm run dev` and `npm run stop`; smoke gains B checks. A's route table and its test are untouched.
- CI: vet and test `backend/adapter`.
- **Done when:** the mock connector passes the contract tests in CI, a run moves through queued, running and completed with logs, a cross-tenant test is refused, and smoke is at 0 fail.

## Stage 4: General's OpenStack connector

The manifest (VM, network and subnet, firewall, volume; bucket, function, table and HTTP API marked removed), the credential spec (an application credential, or `clouds.yaml` fields), verification by a Keystone token request, rendering for `terraform-provider-openstack` `~> 3.4` (with `hashicorp/time` `~> 0.14`), and plan, apply, destroy and logs through the shared runner. B's runner needs a route to DevStack (Lima port forwarding, or the runner inside the same VM). **Done when:** a building-block diagram deploys to DevStack through B and destroys cleanly.

## Stage 5: the AWS connector

A's five templates (a main and an outputs file each) through `generic/tftemplate`, changed only by stage 1 (escaping, the customer role's path and boundary, the API Gateway fix); three new templates (network, firewall, volume), uploaded by the platform Terraform like the others; small override files where a template lacks an input; B's own edge-to-IAM step; the deployment id in Terraform addresses and cloud-side names; A's role table read as AWS connections; and the CloudFormation template trusting B's runner role too. A test renders A's files from disk and runs `terraform validate`, so a change to A's templates that breaks B fails in CI. **Done when:** the same diagram's AWS deployment validates locally; it deployed and destroyed on a real AWS account from a laptop on 2026-11-20, and runs again from the hosted service in stage 7.

## Stage 6: the frontend and the whole flow

- B's own editor, chosen at the page level: a diagram whose saved document has `engine: "blocks"` opens it; any other diagram opens A's editor, unchanged.
- New feature folders for connections, deployments and the catalog, with `NEXT_PUBLIC_ADAPTER_API_ENDPOINT`.
- The Connections page, the building-block palette and forms from manifests, and the deploy panel with one row per deployment (compatibility, exclusions, preview, deploy, destroy, status, logs). "New diagram" creates a building-block diagram.
- A's nodes, labels, image paths and account gate stay as they are. Locally, A2 answers B's AWS verification with a stub, as it does for A.
- **Done when:** the local demo below runs end to end.

## Stage 7: the cloud

The platform Terraform holds B's part (`backend/infra/modules/adapter-service`): its own VPC, cluster, load balancer, roles, run and state buckets, and the run task each run starts (ADR-017). The go-live steps, every value they need and what success looks like are in `environments.md` § Credentials day, their one owner. **Done when:** the 12-step test in `environments.md` and the demo in `system-b-adapter-layer.md` § What the user sees both pass in the cloud.

| Credentials day, after stages 1 to 6 (`environments.md` § Credentials day) | Time |
| --- | --- |
| Steps 1 to 7: accounts, sign-in, the state bucket, Supabase, the Google client, the site's certificate, the config files | about half a day, most of it console work |
| Steps 8 to 11: plan, the first deploy (which also publishes A's API), DNS, then the images and the website | about two hours |
| Steps 12 and 13: link the test AWS account; start a DevStack VM for demos | under an hour |
| Step 14: run the tests and the demo, and fix what the first real run finds (most likely permissions and settings) | a day, maybe more |

## Stage 8 and later

Patrii as an OpenStack connection when it opens; DigitalOcean as the second General connector; General's existing-servers mode (`general-ssh`) once the config runner is chosen; TierHive when it has an API; the optional Clutter sandbox connection; scoped keys on clouds other than AWS; B1, then B2; retiring A's own AWS flow. Blocks across clouds in one deployment (ADR-020) are already built, pulled forward as stage 6b. Live collaboration is another team's work, on the stable diagram document.

### Retiring A's AWS path

A's AWS-only machinery retires once B has proved itself on real AWS; what B shares with A stays.

1. **B covers what A's AWS path does:** function code upload and server setup at first boot. Both are built (stage 6b in `.claude/findings/NOW.md`).
2. **B's AWS path passes on real AWS from the hosted service:** the stage 7 tests on the test account (a laptop run passed on 2026-11-20).
3. **Remove A's AWS machinery in one change:** A's diagram editor and its AWS nodes, generation (`terraform-engine/create`, `files`), the command runner, code upload, A's log routes, the Ansible routes and queue, A's two Fargate runners and their images, and their routes in `backend/infra/main.tf` and A2's route table. Old AWS diagrams (none exist outside test data) are deleted or left read-only; no converter is built unless the client has real ones.
4. **Keep what B uses:** log-in and the JWT, users, organizations, projects, diagram save and load, the CloudFormation account link and A's linked-role table, and the website shell.
5. **Then build C (ADR-021):** move the kept routes into the adapter service under `/v1` and rename its folder for the whole server, point the website at `/v1` only, and delete API Gateway, the remaining Lambdas and A2's Lambda emulation, so Clutter has one backend, one API and one deploy path.

![C after A's AWS path retires: one service, one website, one deploy path](figures/clutter-c.png)

## What runs locally with stages 1 to 6

`npm run dev` starts Postgres, A (through A2), B and the website; DevStack runs in its own VM. In the browser:

1. Sign in.
2. **Connections:** **Connect local AWS** (the fake account, on LocalStack) and **Connect local OpenStack** (this machine's DevStack); both show as verified.
3. **New diagram:** drag in a VM, network, firewall, volume, bucket and function; drop the VM into the network's box, and link the rest with lines. Or start from one of the seven built-in examples.
4. **Deploy panel:** "Amazon Web Services: every block can be built here"; "OpenStack: 2 blocks not supported (Bucket, Function)", which the user leaves out of that deployment.
5. **Preview Terraform** for each deployment.
6. **Deploy to OpenStack:** live status and logs; the VM, network and volume appear in DevStack; **Destroy** removes them.
7. **Deploy to AWS:** valid Terraform is rendered, and the run stops at the AWS call with a clear "needs a real AWS account" message. With LocalStack running (`npm run localstack -- start`), the run applies and destroys on LocalStack instead.
8. Old AWS diagrams open in A's editor and work as before.

From the terminal: `npm run smoke` (A and B, 0 fail), every Go and frontend test, the contract tests and the golden test.
