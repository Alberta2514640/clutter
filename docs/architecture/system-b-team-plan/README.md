# The team's adapter-layer plan

The team's plan, first written on 2026-10-30 and updated on 2026-11-13 to match the decisions made since (ADR-013 to ADR-017): B0 first, users' own clouds, one diagram with many deployments, and the staged build order. The figures were redrawn on 2026-11-14 to match the code as built. They are in `figures/` (also bound together in `diagrams.pdf`). The design that owns the details is `docs/architecture/system-b-adapter-layer.md`; the overall picture is `docs/architecture/figures/clutter-worlds.png`.

![Legend](figures/00-legend.png)

## The context

**Clutter today** is a drag-and-drop canvas. A user places cloud resources on a diagram, Clutter turns that diagram into Terraform (plus Ansible for server setup) and deploys it into the user's own AWS account. It only works with AWS. It is built and tested, and today it runs locally (its AWS hosting was torn down).

**The goal** is to expand that same codebase so it can deploy to targets other than AWS. First General, which reaches many clouds through approved Terraform providers (OpenStack first), then AWS through B, then PatriCloud, TierHive and whatever comes later. You're doing this by building an **adapter layer**: one standard way for Clutter to talk to any target, with a separate **connector** for each target. The people who come after you should be able to add new connectors without reworking what already exists.

## The names

The names the figures use. Every name is defined once in `docs/architecture/system-b-adapter-layer.md` § Names.

- **C**: the whole Clutter app, the big outer box in every diagram.
- **A**: Clutter as it exists today. It is the AWS system and already a complete, tested application.
- **A2**: the local extension that runs A, and B beside it, on a laptop with no AWS.
- **B**: the adapter layer you're building. Its code is in `backend/adapter`, and its API is B API.
- **1, 2, 3 … n**: **connectors**, one per target type (General's, one per approved Terraform provider with OpenStack first; AWS; then PatriCloud, TierHive and so on). There can be any number of them, and most new clouds arrive as General connectors.
- **B0**: the adapter run by us as a hosted service (SaaS). **Built first.**
- **B1**: the adapter shipped as a **self-hosted**, separate service that people can run and use. Later.
- **B2**: the adapter as an **internal** thing a company runs inside its own setup, not self-hosted by you. In B2 the connectors become **instance connectors**. Later, and a separate decision.
- **X**: a connector (X1, X2 … Xn).
- **I**: a **connection**, one organization's linked cloud account for a connector (I1, I2 … In).
- **Deployment**: one diagram bound to one connection. A diagram can have several.
- **a, b, c, d … n**: the **building blocks** the diagram is drawn in (VM, network, firewall, volume, bucket, function, table, HTTP API), which each connector maps onto.

## Level 1: three ways to attach B to A

### Fig 1 · Option 1: build B next to A (chosen)

![Option 1](figures/01-option-1-build-b-next-to-a.png)

A stays as it is, except stage 1's security fixes and the moves that make its template loader and JWT check shareable (ADR-006). B is a new, separate piece inside the same app that contains the connector interface, and connectors 1, 2, 3 … n plug into it. As built, B is `backend/adapter`, with three connectors: General's OpenStack connector, the AWS connector (A's five templates plus network, firewall and volume), and a mock connector for local tests only. The website is one app for A and B (ADR-024): a diagram opens in A's editor or B's editor, chosen per diagram.

The reason is testing. A is already a full working application. Changing anything in it means retesting and re-running it, so if you leave A alone, all of that stays safe. All new work and new tests live in B.

This split applies to both the backend (new services) and the frontend (new screens).

### Fig 2 · Option 2: B reuses parts of A (fallback)

![Option 2](figures/02-option-2-b-reuses-a.png)

The structure is the same as Option 1, but B reuses existing functions and code from A instead of rewriting them. That avoids copying or duplicating code and makes use of what already exists. How much gets reused depends on the code itself and how cleanly each part of A can be called from outside.

This is the fallback if Option 1 doesn't work out cleanly. The plan is to stay within Options 1 and 2.

From the repo: B already imports A's `backend/api/generic` package (the JWT check, the name check and the template loader), reads A's organization, project, diagram and linked AWS account tables, and renders A's five AWS template files. The figure marks these in teal. All of them sit at the named seams, so they are Option 1 as planned; reuse beyond them is what Option 2 would add.

The trade-off: once B depends on parts of A, a change to those parts can break B. Those shared pieces need to be stable or have their own tests.

### Fig 3 · Option 3: turn A into the adapter (later, piece by piece)

![Option 3](figures/03-option-3-turn-a-into-adapter.png)

This is what happens if you start from A and find that so much of it is nearly the same as what B needs that you convert A itself into the full adapter layer. That means editing A, removing AWS-only parts, splitting the Terraform engine and migrating the database. A gets absorbed into B, AWS becomes just one more connector, and B keeps expanding with the same connectors as the other options.

It is never the first move, because it means adjusting and retesting all existing functionality, including every AWS path, before any new connector ships. System A essentially *is* the AWS side of Clutter. It stays a later destination, reached piece by piece: once B's AWS path passes on real AWS, A's own AWS machinery is removed, and A's core (sign-in, organizations, projects, diagrams and the AWS account link) moves into B's service under `/v1`. That one system is C (ADR-021), drawn in `docs/architecture/figures/clutter-c.png`.

## Level 2: inside B

### Fig 4 · One diagram, many deployments

![One diagram, many deployments](figures/04-one-diagram-many-deployments.png)

This zooms into the overall system once B exists, as a user sees it.

- A diagram is drawn **once**, in the eight building blocks.
- Each cloud it runs on is a **deployment**: the diagram, one **connection**, that cloud's settings, and its own Terraform state and runs. The deploy panel lists them, with a line saying whether each cloud can build everything ("every block can be built here"), or which blocks it leaves out. Each row has Preview Terraform, Plan, Deploy and Destroy, and shows the run's log.
- A **split deployment** places some blocks on another of the organization's clouds. A VM can get another cloud's VM address, or read a bucket or table there with a scoped key that Clutter creates and deletes (ADR-020).
- **Connections** belong to the organization and are linked once on the Connections page: an AWS account through the CloudFormation role, or an OpenStack application credential or `clouds.yaml` file (servers over SSH later), each verified before it is saved.
- Diagrams made before B (AWS-only) keep opening in A's editor, unchanged; a diagram marked `engine: "blocks"` opens in B's editor.

There is no per-diagram "AWS or Custom" choice: deployments replace it. A diagram is not tied to one cloud, so the user never has to pick one up front.

B can be delivered three ways, and the same connectors serve all three:

- **B0, hosted adapter:** run by us as SaaS, like A. **Built first**, because it proves B with the least setup and support work.
- **B1, self-hosted adapter:** a separate service people run themselves, with access to everything Clutter offers. Its connectors (1, 2, 3, 4 … n) are **entity connectors**, one per target type, written by us. It stays possible because B accepts any Postgres and any S3-compatible store.
- **B2, internal adapter:** the same adapter used inside a company. Its connectors become **instance connectors**, because each company builds or configures its own.

The overall order of decisions is: choose among Options 1/2/3 (Option 1), then the delivery model (B0 first), and A stays as AWS either way (B's AWS connector reuses A's template files). Whether to do B2 at all is a separate decision: it needs information we do not have yet, namely how targets such as TierHive and PatriCloud work inside the companies that use them.

### Fig 5 · Building blocks → connector → connection

![Building blocks, connector, connection](figures/05-blocks-connector-connection.png)

This goes one level further into B.

- **Building blocks** (a … h) are the same for every connector: VM, network, firewall, volume, bucket, function, table, HTTP API. Their fields are owned once, in `backend/adapter/connectors/blocks.go`. The left column shows what each becomes on AWS. Server setup is the VM's startup script, run by cloud-init at first boot on every cloud.
- **X** is a connector, here General's OpenStack connector. Its **manifest** says how it treats each block. There are X2 … Xn: AWS and the mock (built), and later DigitalOcean, PatriCloud and TierHive.
- **I** is one connection for X: one organization's OpenStack, with its own credential and region. There can be I2 … In. Each **deployment** that uses it has its own settings (image, flavors), exclusions and state.

Each connector can treat each block differently:

- **As-is:** the block maps straight across. On OpenStack a VM becomes a Nova server (with a floating IP when it is public), a network becomes a network, subnet and router, a firewall a security group, and a volume a Cinder volume.
- **Extended:** the block exists with more for this connector or connection (for example, a custom image).
- **Removed:** the block does not apply (OpenStack here has no bucket, function, table or HTTP API); a deployment can leave such a block out.
- **New:** something only this connector has, outside the standard set (for example, a GPU flavor).

No connector uses "extended" or "new" yet; the figure marks those two as examples.

Shared behaviour lives in the building blocks. Differences live in the connector's manifest, never in the standard set.

## Level 3: how it actually works

### Fig 6 · System A (as built)

![System A as built](figures/06-system-a-today.png)

From the repo:

1. The **browser** runs the website (Next.js and the React Flow canvas, on Fargate in the cloud), with Google sign-in.
2. **API Gateway** checks requests with a JWT authorizer Lambda. Two routes have no authorizer on purpose: `POST /log-in` and `GET /resources` (the static catalog). Generation is signed in, and the editor calls it after each save (ADR-018).
3. **Go Lambdas** each handle one domain: log-in, user information, organization and accounts, project, diagram, resources, terraform-engine, terraform-command-runner, ansible jobs and cloudformation-stack-url. That's 32 routes and 34 Lambdas. Each handler checks that the caller is a member of the organization and that the project, diagram or run belongs to it.
4. **Data:** Postgres (Supabase, 11 tables) and S3 (templates, .tf files, state, logs, playbooks, code uploads and run status files).
5. **Deploy path:**
   - The **terraform-engine** reads the saved diagram and fills Go templates from S3 to produce `main.tf` and `outputs.tf`, generating IAM policies from the diagram's edges.
   - The **terraform-command-runner** starts one **Fargate terraform-deployer** task per run. That task assumes a role in the **customer's AWS account** (STS with an external ID, set up through a CloudFormation stack, role version 10) and runs plan, apply or destroy. It holds no database string; it writes its status and logs to S3, and the API records the status.
   - **Ansible jobs** go through SQS (with a dead-letter queue) to the `ansible/run-task` Lambda, which starts a **Fargate ansible-runner** that uses SSM Run Command to configure EC2. It also holds no database string.
   - The customer role changes only resources tagged `ManagedBy`, and every role and user it creates sits under a permissions boundary.

The amber parts in the diagram are the AWS-only path. B brings its own for new diagrams, on every cloud. Still open: the deployer's Terraform state is not locked (phase 0 item 9).

### Fig 7 · System B: how the adapter layer works (as built)

![System B as built](figures/07-system-b-architecture.png)

This is B as built, delivered as B0 in the cloud (stage 7 code; not deployed yet):

1. **One public name:** a load balancer sends `/v1/*` and `/healthz` to B API and every other path to the website (ADR-023).
2. **Website:** one app for A and B (ADR-024), on its own Fargate cluster. Older AWS diagrams open in A's editor; building-block diagrams open in B's editor, with the deploy panel and the Connections page.
3. **A stays unchanged** on its AWS path, behind its own API Gateway address. B reuses A's sign-in (the JWT check in `generic`), A's organizations, projects and diagrams, and A's linked AWS accounts.
4. **Inside B** (its own VPC and the ECS cluster `clutter-adapter`):
   - **B API:** one Go service on Fargate, with the `/v1` routes; it checks the JWT.
   - **Translator** and **capability check:** turn the diagram into a graph of building blocks, and check it against the connector's manifest and its `Validate`.
   - **Terraform preview:** renders a deployment's Terraform without any credentials.
   - **Linked accounts:** lists the AWS accounts the organization linked in A, so an AWS account is linked once.
   - **Job runner:** queues each run (at most 3 per organization), tracks its status and keeps its log with credential values removed.
   - **Credential broker:** fetches one run's credentials: short-lived AWS keys from STS, or the one connection's secret from Secrets Manager.
   - **Run launcher:** checks the rendered HCL, uploads the run's files and one-hour keys to the S3 run bucket, and starts one Fargate run task.
   - **Run task:** in its own run subnets, whose network ACL refuses private addresses; it has no role, no database and no secret store. It runs `terraform init` against the S3 state bucket (locked by an S3 lock file), then plan, apply or destroy, and writes its log, result and outputs back to S3. The job runner reads its status from there.
   - **Connector interface:** validate, plan, apply and destroy, plus the optional preview, linked accounts, option lists and cross-cloud parts.
   - **Connectors:** General's OpenStack, AWS (A's 5 templates plus network, firewall and volume) and the mock (local only); more General providers, PatriCloud and TierHive later.
5. **Shared Postgres:** A's 11 tables plus B's 5 (connections, deployments, runs, function code, placements). OpenStack credentials and B's own secrets sit in Secrets Manager.

Locally (A2), B runs on `127.0.0.1:8790`, each run is a child process, and state and logs sit in B's data folder.

The gold diamonds mark the **places B touches A**: the UI, shared sign-in, the shared database and A's linked AWS accounts. The fourth seam is the security fixes on paths B shares with A. Even in Option 1, these are the unavoidable connection points.

### Fig 8 · One deploy, step by step (as built)

![Deploy sequence](figures/08-deploy-sequence.png)

Steps 1 to 3 are A's editor (older AWS diagrams); steps 4 to 15 are B's editor (building-block diagrams).

1. The user saves the diagram.
2. The editor then calls `POST /terraform-engine` with the user's token.
3. The generator writes `main.tf` and `outputs.tf` to S3; A's own runner deploys them.
4. The user saves the diagram. It is marked `engine: "blocks"`, so A's generator skips it.
5. The deploy panel asks B API for each deployment's compatibility. B API checks that the user is a member, and that the diagram and the connection belong to the organization, then runs the translator, the capability check and the connector's `Validate`. The answer is "every block can be built here", or what cannot.
6. Preview Terraform shows the files, rendered without credentials.
7. Plan, Deploy or Destroy starts a run (status QUEUED), at most 3 per organization and 1 per deployment.
8. The job runner marks it RUNNING. The broker fetches only this run's keys; the connector verifies the connection, renders the Terraform, and B refuses anything unsafe in it.
9. B uploads the run's files and one-hour keys to the S3 run bucket.
10. B starts one Fargate run task.
11. The task runs `terraform init` (state in S3, locked), then plan or apply against the user's cloud.
12. The cloud returns the resources and their IDs.
13. The task writes its log, result and outputs to S3.
14. The job runner reads the status through S3, keeps the log with credential values removed, deletes the run's objects, and marks the run COMPLETED (or FAILED).
15. The deploy panel polls the run and its log.

A split deployment repeats steps 8 to 14 once per cloud, in dependency order, each in its own task with only that cloud's keys.

### Fig 9 · Build order and where it stands

![Build order](figures/09-build-order.png)

The status on 2026-11-20, from `.claude/findings/NOW.md` § The order, which owns it:

0. **Stage 0, housekeeping:** done. Decisions recorded, the design checked, merged into the fork's `main`; the client pull request and the message to the client are still to do.
1. **Stage 1, make A safe:** done in code. Every phase 0 item is done or decided except item 9 (a lock on A's Terraform state); A's template loader and JWT check are in `generic`. Item 7 (the customer role) was confirmed on a real AWS account on 2026-11-20, through B, and is now version 10 (ADR-027); item 10 (A's runners) waits for Clutter to be hosted.
2. **Stage 2, a test cloud on a laptop:** done. DevStack is proven on one laptop, `npm run devstack -- create` sets it up, and `npm run devstack -- route` lets the Mac open the servers built there (`backend/devserver/README.md`).
3. **Stage 3, B's core:** done. B API, tables, job runner, run task and connector interface; the mock passes the contract tests.
4. **Stage 4, General's OpenStack connector:** built. A real run on DevStack built and destroyed a VM, network, firewall and volume, from its test and from the browser.
5. **Stage 5, the AWS connector:** built. It validates, and it applied and destroyed every block on LocalStack and, on 2026-11-20, on a real AWS account (`docs/reports/2026-11-20-real-cloud-verification.pdf`).
6. **Stage 6, the frontend and the whole flow:** built. Every feature runs end to end in a browser, AWS on LocalStack and OpenStack on DevStack.
   - **Stage 6b, no manual steps:** done. Function code, cloud-init, `clouds.yaml`, split deployments, website hosting, naming, and the UX, visual and docs passes are done.
7. **Stage 7, the cloud:** done in code; hosting needs the client's accounts. Clutter on a laptop already built into a real AWS account and VEXXHOST on 2026-11-20.
8. **Stage 8 and later:** more General connectors (DigitalOcean next, then existing servers over SSH), PatriCloud and TierHive (once they have APIs), the optional Clutter sandbox connection, B1, then B2 with instance connectors once partner details exist; then A's AWS path retires and C is built (ADR-021), which reaches Option 3's end state without a big-bang retest.

Stages 1 to 6 run and are tested on a laptop. The details of each stage are in `docs/architecture/implementation-plan.md`.

## Assessment and recommendations

Overall the plan is sound. The figures form a clear chain:

- how B attaches to A (Option 1 preferred, 2 as fallback, 3 avoided)
- how users deploy (one diagram, many deployments; B0 now, B1 and B2 later, A kept for existing AWS diagrams)
- how features map onto connectors and connections

Things to settle before coding:

1. **"Don't touch A" really means "touch A only at the seams."** Shared sign-in, the frontend's Connections page and deploy panel, and new tables for connections and deployments are unavoidable. Name them up front so Option 1 doesn't drift into Option 2 without anyone deciding it.
2. **Define the connector interface first.** Something like validate → plan → apply → destroy → logs, mirroring what A's Terraform runner already does. Every connector, and B2 later, implements the same contract.
3. **The diagram engine is the hardest part.** A's Terraform engine is built around AWS resources and IAM rules. Each connector needs its own mapping from the same diagram. Decide early whether connectors output Terraform for other providers, call target APIs directly, or both.
4. **Make connectors declare their capabilities.** Each connector lists which features it supports, extends or drops (Fig 5's as-is / extended / removed / new). The UI can then hide what a target can't do instead of failing at deploy time.
5. **Pin down entity vs instance connectors.**
   - An *entity connector* is a kind of cloud, like OpenStack or TierHive, written once by us.
   - A *connection* is one company's configured setup of it.
   - In B0 and B1, a connection is configuration (credentials, region).
   - In B2, connections become code each company owns: *instance connectors*.
6. **Keep Option 3 as a possible destination, not the starting move.** If B's interface is general enough for AWS to fit, A can migrate into it gradually, whenever that's worth it.

## Open questions

- How do TierHive and PatriCloud expose their infrastructure (Terraform providers, APIs, something else)? This decides how connectors are built and whether B2 is realistic.
- What does "General / custom" mean concretely? Plain Terraform for any provider, or a generic API format?
- Does B1 run as Lambdas like A, or as a long-running container, given that self-hosters need to run it outside your AWS account?
- Where are connector credentials stored, and how are they secured for each connection?

Answers so far: TierHive and PatriCloud have no public API yet (`docs/research/`); General is a family of connectors, one per approved Terraform provider (OpenStack first), plus existing servers over SSH, and it is where new clouds are added (ADR-008, ADR-017); B runs as a container (ADR-004); credentials are borrowed roles or encrypted secrets with only a reference in Postgres (ADR-016). Details are in `docs/architecture/system-b-adapter-layer.md`, and what is still open is in `.claude/findings/NOW.md`.
