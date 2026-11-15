# devserver

This is **A2**: the extension that runs System A on one machine, without AWS, so the team can test what exists. It wraps A, with one small exception: the local-only **Link fake account** button sits in the website. Development only.

It runs the real, unchanged Lambdas from `backend/api`. Each one is built for this machine and started in aws-lambda-go's RPC mode. A gateway on `127.0.0.1:8787` turns HTTP requests into API Gateway REST proxy events, the same way the platform's API Gateway does. It runs the real authorizer on protected routes and answers CORS preflights. Postgres runs in Docker, loaded from `backend/storage-sql`. S3 is a small local store on disk, so the generation the editor asks for after a save writes real Terraform files. B API (`backend/adapter`) runs beside the gateway on `127.0.0.1:8790` (see [B API locally](#b-api-locally)).

## Start and stop

`npm run dev` at the repo root starts everything, and Ctrl+C stops it (root `README.md` § Quick start). Its start message says where to go, how to sign in and how to add sample data. Behind it are the scripts here, with shared settings in `lib.sh`. Each has an `npm run` name and answers `--help`:

| Script | `npm run` name | Does |
| --- | --- | --- |
| `doctor.sh` | `doctor` | checks the tools, ports and disk, one line each, and prints the install command for this platform. It exits with 1 only when a required tool is missing or unusable |
| `setup.sh` | `setup` | runs the doctor, then `npm ci` in `frontend/clutter` when its lock file changed. It runs `go mod download` and one build for every Go module under `backend/`. It pulls the Postgres image and checks `docker run`, and copies `frontend/clutter/.env.example` to `.env.local` when it is missing. It is safe to run again |
| `dev.sh` | `dev`, `dev:backend`, `dev:frontend` | `dev` starts everything. `dev:backend` (`--backend-only`) starts Postgres, the gateway and B API without the website, and returns. `dev:frontend` (`--frontend-only`) runs only the website, the same way `dev` does, against that backend; Ctrl+C stops the website alone |
| `seed-diagram.py` | `seed` | adds an organization with a fake linked AWS account, a project and an AWS diagram for the `/dev/login` user |
| `smoke.sh` | `smoke` | calls every route and B's main paths (see [What works locally](#what-works-locally-and-what-only-works-in-aws)) |
| `generated.sh` | `generated` | lists and validates generated Terraform (see [Generating Terraform locally](#generating-terraform-locally)) |
| `stop.sh` | `stop` | stops the website, the backend and Postgres, even when `dev.sh` runs in another terminal or a script; `--wipe` also deletes the database, the local S3 files and B's data |
| `localstack.sh` | `localstack` | see [AWS on LocalStack](#aws-on-localstack) |
| `devstack.sh` | `devstack` | makes the Lima VM named `devstack` and installs DevStack in it, makes the credential Clutter uses, and starts, stops or shows the VM (see [OpenStack on this machine (DevStack)](#openstack-on-this-machine-devstack)) |

The doctor's required checks are Node, npm, Go, Docker, curl and openssl. Node must be the version in `.nvmrc` or newer. Go must be the newest `go` line among the `go.mod` files, or Go 1.21 or newer that may download it. Docker needs its compose plugin, and must be running and able to start a container. Recommended: python3 (for `seed`), Terraform (the version B's run image installs, for B's local deploys and `generated`), the free ports and disk space. Optional: the LocalStack image, Lima and the `devstack` VM, and Playwright's Chromium. When the `devstack` VM runs, the doctor also checks that this machine reaches DevStack. `test-scripts.sh` runs the doctor and `devstack.sh` against fake tools on PATH (a missing tool, a stopped Docker, a refused user, everything present, DevStack reachable or blocked, on macOS and Linux). It also runs every script's `--help`. CI runs it.

The gateway also serves `POST /dev/link-fake-account` for the local-only **Link fake account** button, on A's "AWS account required" popup and in the AWS link flow on Connections. It links one fake account per organization and answers with the linked account's name. A production build shows no such button.

## Why the backend runs this way

Postgres runs in Docker because it is a ready-made image. The gateway and the Lambdas run directly on your machine instead. They rebuild in seconds from the code you are editing and need only Go. The frontend runs directly too, because its hot reload is fastest outside a container. Nothing stops them moving into Docker later (a single `docker compose up`, with no Go or Node needed). The cost is slower rebuilds and reloads. There is no `go run` for one Lambda. Each is a function with no web server of its own, and the gateway is what gives them URLs. Go edits need a restart, because the gateway builds the Lambdas once at start.

## What it needs

Nothing to configure: no `.env` file, no API keys, no accounts. `dev.sh` generates a local JWT secret, a database password and B's credential key into `.state/` and passes every variable the Lambdas and B need. ECS and SQS are the parts that need real accounts (`docs/architecture/environments.md`).

## If it does not start

| You see | Do this |
| --- | --- |
| `Missing go` (or another tool), or you are not sure what is missing | `npm run doctor`; each `FAIL` line says what to install |
| `Cannot connect to the Docker daemon` | open Docker Desktop and wait until it says it is running, then run `npm run dev` again |
| `something already listens on 127.0.0.1:8787` (or 8790, 3000 or 3100) | `npm run stop`, or find the process with `lsof -nP -iTCP:3000 -sTCP:LISTEN` and stop it |
| a second clone of the repo on the same machine | only one copy of Clutter runs at a time: the ports are fixed, and every clone shares one database (the Compose project is `clutter-devserver`). B's saved credentials in it are encrypted with the first clone's key, which the second cannot read. Run `npm run stop -- --wipe` (in either copy) before the first `npm run dev` in the new one |
| `The backend is not running. Run npm run dev:backend first.` | `npm run dev:frontend` runs only the website; start the backend first with `npm run dev:backend`, or use `npm run dev` |
| `B API exited` or `did not become ready` | read the end of `backend/devserver/.state/adapter.log`; a schema error there usually means `npm run stop -- --wipe` |
| the first `npm run dev` takes a minute or two | normal: it builds the Lambdas once |
| Google sign-in fails | sign in with `http://localhost:3000/dev/login`, a local test user (see [How sign-in works locally](#how-sign-in-works-locally)) |
| the canvas says "AWS account required" | press **Link fake account** on that popup |
| odd data, or you changed `backend/storage-sql` | `npm run stop -- --wipe`, then `npm run dev` reloads the schema |
| Preview Terraform shows no files after a save | read the `terraform-engine-create` lines in `backend/devserver/.state/gateway.log`; a failed generation never fails the save, and the editor shows its reason |
| anything else | read the end of `backend/devserver/.state/gateway.log` |

## What works locally, and what only works in AWS

Checked on 2026-11-13 with `smoke.sh` (all 32 routes, 39 calls: 31 pass, 3 need AWS, 5 expected failures, 0 fail; then 32 B checks, 32 pass, 0 fail). Its checks outside the route rows include a member of another organization being refused. The smoke checks that refusal on generation, on the files route and on every route that takes a project, diagram or run id. Any fail, in A's rows or B's checks, makes the smoke exit with 1. Browser results are in `frontend/clutter/README.md`.

| Feature | Locally | Why |
| --- | --- | --- |
| Sign in | works two ways | see [How sign-in works locally](#how-sign-in-works-locally) |
| Protected routes and the authorizer | work | the real authorizer runs; no token gives 401, a bad token gives 403 |
| Organizations: create, list, get, update, delete | work | Postgres only |
| AWS account link: stack URL, list, submit role ARN, delete | work | Postgres only; the CloudFormation link is built but never opened, and the ARN is never checked against AWS (it isn't in AWS either) |
| Projects and diagrams: create, list, get, update, delete | work | Postgres only; a saved diagram reads back with its nodes and edges |
| Resource palette (`GET /resources`), user information, recent activity | work | Postgres only |
| Presigned upload and download URLs (Lambda code, deploy logs, Ansible playbooks) | work, from a browser too | the URL points at the local S3, which accepts it without checking the signature and answers CORS |
| Ansible job list and job details | work, on data already in the database | reading jobs is Postgres only; the list takes an `orgId` |
| Generate Terraform from a diagram | works, when the editor asks after a save | see [Generating Terraform locally](#generating-terraform-locally) |
| Deploy or destroy (`POST /terraform-command-runner`) | only in AWS | starts an ECS Fargate task; locally it leaves a `RUNNING` log row behind |
| Live deploy status (`logs/live`) | only in AWS | ECS `DescribeTasks` |
| Deploy logs (`logs`, `logs/url`) | routes work, but no logs exist | they read the local S3; only the runner writes deploy logs, and it needs AWS |
| Submit an Ansible job (`POST /ansible/jobs`) | only in AWS | the playbook checks pass in the local S3, then queueing on SQS is refused and the job is marked `FAILED` |
| Ansible job logs | route works, but no logs exist | it reads the local S3; only the runner writes job logs, and it needs AWS |
| Running Ansible on a server (`ansible/run-task`, the two runner images) | only in AWS | an SQS consumer starting Fargate tasks that use STS and SSM |

Anything marked "only in AWS" stops at its first call to ECS or SQS, after its database, auth and S3 steps have run. Testing those paths needs an AWS account (or an SQS and ECS emulator, which this devserver does not include).

## Generating Terraform locally

Save a diagram in the app. After the save succeeds, the editor calls `POST /terraform-engine` with the user's token and `{ "projectId", "diagramId" }`, the same as in AWS. The gateway runs the real authorizer and the real `terraform-engine-create` Lambda, which checks that the caller owns the diagram and reads it from Postgres. A save alone generates nothing, so a script that saves through `PUT /diagram` calls `POST /terraform-engine` next (`seed-diagram.py` does). A failed generation never undoes the save: the editor shows the reason, and the gateway log has the cause.

The Lambda reads templates through the local S3, so it gets the repo's real files. The templates bucket follows the `aws_s3_object` blocks in `backend/infra/modules/s3/main.tf`, and reads each file from `backend/infra/modules/s3/uploads` on every request. A template edit therefore shows up on the next save, with no copy step. A template with no `aws_s3_object` does not exist here, the same as in production. If a block cannot be resolved to a file, the gateway does not start, and `go test` fails.

The files land in `backend/devserver/.state/s3/clutter-devserver-local/<org>/<project>/<diagram>/terraform/`, as `main.tf` and `outputs.tf`. A diagram with no nodes gets `main.tf` only.

Read them in the editor with **Preview Terraform**. List them with `npm run generated`, and validate one with `npm run generated -- <diagram-id>`. The first validate downloads Terraform's AWS provider (about 680 MB) and retries if the download drops. Validation runs on a scratch copy, never in the repo tree, and deletes the copy afterwards. It runs no plan and no apply. Terraform gets a clean environment and an empty home, so no AWS credentials reach it. `terraform init` is the one step that uses the network. The first run downloads the AWS and random providers from `registry.terraform.io` into `.state/terraform-plugins`, and later runs reuse them.

`npm run generated` lists what is on disk. Deleting a diagram does not delete its files, the same as in production, so a deleted diagram stays in the list until `npm run stop -- --wipe`.

## How sign-in works locally

Real Google sign-in works here while the capstone team's Google client still exists. The log-in Lambda runs unchanged. The gateway gives it the same `GOOGLE_CLIENT_ID` the frontend signs in with, so it accepts only tokens minted for Clutter. A test fails if the two ids drift apart. Each Google account is a separate local user, which is how to test two users. `/dev/login` is for automated tests and as a fallback; people use `/login`. It creates a local user, `dev@clutter.localhost`. It signs a token with the same claims and algorithm as the log-in Lambda, using a secret that `dev.sh` generates. The real authorizer accepts that token.

The frontend reads its API token from `localStorage` on its own origin, and the gateway on port 8787 cannot write it. So the gateway also runs a front door on `127.0.0.1:3000`. The front door serves `/dev/login` and passes every other request to the Next dev server on port 3100. `dev.sh` starts that server with `NEXT_PUBLIC_API_ENDPOINT=http://localhost:8787` (A) and `NEXT_PUBLIC_ADAPTER_API_ENDPOINT=http://localhost:8790` (B). So B's screens (the building-block editor, its deploy panel and the Connections page) call B directly. `/dev/login` sets the `clutter_token` cookie and the `google_data` localStorage entry, just as a real log-in does, then opens `/dashboard`.

## B API locally

`dev.sh` builds `backend/adapter/cmd/api` into `.state/adapter-api` and starts it on `127.0.0.1:8790` with a clean environment and only B's own variables. It gets the same JWT secret as A, so A's tokens (including `/dev/login`'s) sign in to B too. It connects to the same Postgres and applies `backend/storage-sql/adapter-tables.sql` at start-up. The Postgres container also loads that file as `30-adapter-tables.sql` on a fresh volume. B offers three connectors. `mock` builds nothing. `general-openstack` needs a reachable OpenStack, such as DevStack (see [OpenStack on this machine (DevStack)](#openstack-on-this-machine-devstack)). `aws` renders and previews real Terraform from the repo's templates. An `aws` run applies against LocalStack when it is running (see [AWS on LocalStack](#aws-on-localstack)), and otherwise stops at "needs a real AWS account". Connection credentials are encrypted under `.state/adapter/secrets` with a key `dev.sh` generates once into `.state/adapter_credential_key` (mode 600). Run logs go to `.state/adapter/logs`. B answers CORS for `http://localhost:3000` and `http://127.0.0.1:3000` only. Its log is `.state/adapter.log`. `npm run stop -- --wipe` deletes `.state/adapter` with the database, and keeps the key.

The smoke's B section signs in as the smoke user and checks these, in order:

1. `/healthz`.
2. `/v1/catalog` with and without a token.
3. The CORS preflight.
4. A building-block diagram (`engine: "blocks"`) saved through A's `PUT /diagram`.
5. A mock connection created and verified (a wrong token is refused).
6. A deployment that leaves out the bucket, its compatibility, then plan and apply to `completed` with logs.
7. A member of another organization refused on every connection, deployment and run path.
8. The connection delete refused with 409 while the deployment is live.
9. Destroy.
10. The delete succeeding.

It cleans up through A's organization delete. None of these calls count in A's 32 route rows.

## AWS on LocalStack

LocalStack stands in for AWS on this machine, so AWS Terraform applies and destroys for real before any client account exists. Clutter uses only the frozen image `localstack/localstack:4.14.0`, pinned by digest in `lib.sh`. It is Apache-2.0 and needs no auth token. Newer tags need a paid token, so never move to one (`docs/research/aws/2026-11-13-localstack.md`).

| Command | What it does |
| --- | --- |
| `npm run localstack -- start` | starts LocalStack on `127.0.0.1:4566` (loopback only), with the Docker socket mounted for Lambda, and waits until `/_localstack/health` answers |
| `npm run localstack -- status` | says whether it runs, and lists its services |
| `npm run localstack -- stop` | removes the container; LocalStack keeps nothing between starts |

When LocalStack answers at start, `dev.sh` starts B with `ADAPTER_AWS_STS=localstack` and `ADAPTER_AWS_ENDPOINT=http://127.0.0.1:4566`, and says so in its start message. B then gets each run's keys from LocalStack's STS and sends every AWS call of the run to LocalStack. So plan, apply and destroy of an AWS deployment in B's editor run for real. The **Link fake account** role works as it is: LocalStack lets any role in. Start LocalStack before `npm run dev`, because `dev.sh` checks only once. If Clutter already runs, run `npm run dev` again: it restarts and checks again. Without LocalStack, the start message's AWS line says how to turn it on, and B's AWS runs stop at "needs a real AWS account". A's own Deploy button still stops at ECS, as the table above says.

### The LocalStack tests

Two gated tests apply for real, and both skip unless `CLUTTER_LOCALSTACK=1`. Run them one at a time. Two Terraform runs that start the AWS provider from one plugin cache at the same moment can fail to load it. Point `TF_PLUGIN_CACHE_DIR` at an existing folder to keep the AWS provider (about 680 MB) between runs.

```sh
npm run localstack -- start
mkdir -p "$HOME/.cache/clutter-tf-plugins"
CLUTTER_LOCALSTACK=1 TF_PLUGIN_CACHE_DIR="$HOME/.cache/clutter-tf-plugins" \
  go -C backend/api test ./terraform-engine/create/internal/generator/ -run LocalStack -v -count=1 -timeout 30m
CLUTTER_LOCALSTACK=1 TF_PLUGIN_CACHE_DIR="$HOME/.cache/clutter-tf-plugins" \
  go -C backend/adapter test ./connectors/aws/ -run LocalStack -v -count=1 -timeout 40m
```

- **A's test** (`TestRenderedTerraformAppliesOnLocalStack`) renders the golden fixtures `all-five-cors-on`, `all-five-cors-off` and `defaults`, and lays them out as A's writer uploads them. It adds `localstack_override.tf`, which points the provider at LocalStack. A's writer is unchanged. The test creates the permissions boundary the role stack would make. Then it applies each fixture and checks that every AWS resource exists. Then it destroys the fixture and checks that every one is gone.
- **B's test** (`TestEveryBlockAppliesOnLocalStack`) deploys Clutter's CloudFormation role stack into LocalStack from the repo's template. It then runs plan, apply and destroy for a diagram with every AWS block and edge, as the stack's role. The runs go through B's job runner, Terraform runner and a real `terraform`. It checks that every block's main resource is reported built, that every AWS resource exists after apply, and that every one is gone after destroy.

The provider itself reads "exists" and "gone". A refresh-only plan after apply must find every resource. A refresh-only plan of the state from before the destroy must find every one deleted. Each test builds its environment from nothing, with LocalStack's test keys, so this machine's AWS credentials never reach Terraform.

Results, checked on 2026-11-13 (B) and 2026-11-14 (A):

| Test | Result |
| --- | --- |
| B, every block and edge | passes in about 5 minutes: 51 resources of 30 types apply, exist, and are gone after destroy |
| A, `defaults` | passes: 17 resources apply, exist, and are gone after destroy |
| A, `all-five-cors-on` | passes: 21 resources. The apigateway template makes the OPTIONS integration response depend on its integration; without that, AWS can answer `NotFoundException: Invalid Response status code specified` |
| A, `all-five-cors-off` | passes: 15 resources. With CORS off, A's API has no method, and API Gateway refuses to deploy an API with no method (`The REST API doesn't contain any methods`), so the template renders no deployment or stage then. A's templates still render no route for an edge from an API to a function |

### What LocalStack cannot prove

- **IAM.** LocalStack 4.14.0 enforces no IAM policy, so the customer role's permissions, the permissions boundary and the PassRole conditions are not checked. Its STS lets in a role that does not exist, with any external id, and `CreateRole` accepts a boundary that does not exist.
- **The role stack's exact shape.** Its CloudFormation creates the stack, but it drops a managed policy's `Path`, so the boundary sits at `/` instead of `/clutter/`.
- **EC2.** Instances are records in memory: nothing boots, no SSM agent runs, and Ansible cannot reach them. The mock also stores instances differently from what it was asked. In B's test, a plan right after apply wants to replace the `worker` instance. LocalStack gave that instance a public IP address that its configuration turns off. The same plan also wants to change both instances' security group lists. Whether real AWS shows either difference is not checked here.
- **Running code.** The tests create functions and APIs but never call them.
- **Anything about a real account:** quotas, which services a region offers, the provider's retries against real AWS errors.

These stay on a real AWS test account. LocalStack is not in CI: it needs the Docker socket, takes about 10 minutes for both tests, and downloads the AWS provider. Run both tests before changing an AWS template, A's writer or B's AWS connector.

## OpenStack on this machine (DevStack)

DevStack is a small, real OpenStack. It runs in a Lima VM named `devstack`, so B's OpenStack connector builds real servers, networks, firewalls and volumes on this machine. The VM's template (`lima.yaml`) and the two scripts it runs inside the VM (`install.sh` and `credential.sh`) are in `devstack/`. They hold no secrets. DevStack's passwords are made at random inside the VM when it is installed, and stay there.

### What DevStack needs

| Need | Detail |
| --- | --- |
| A Mac | `create` needs macOS, because the VM's address comes from Apple's virtualization network (vzNAT). Apple silicon is tested (an M4 Pro). Intel Macs are not tested. On Linux, `start`, `stop`, `status` and `credential` work with a `devstack` VM you make yourself |
| Speed | On Apple M3 or newer with macOS 15 or newer, DevStack's own servers run at full speed (nested virtualization). Other Macs work, but those servers run under emulation, which is much slower. `create` picks the right setting |
| CPUs and memory | the VM takes 4 CPUs and 12 GiB of memory while it runs, so the Mac needs 16 GB or more (tested with 24 GB) |
| Disk | the VM's disk is 40 GiB, and takes space only as it fills: about 12 GB after the install |
| Lima | 2.0 or newer (`brew install lima`); tested with 2.3.0 |
| Time | 10 to 20 minutes for the first `create`: it downloads Ubuntu 24.04, then DevStack's install took about 9 minutes on an M4 Pro |
| DevStack version | `install.sh` installs DevStack's latest `master`. The last version proven with Clutter is commit `902e73e` (2026-11-10). If a fresh install fails, a newer DevStack is the first suspect |

### Make it and link it

```sh
npm run devstack -- create       # makes the VM and installs DevStack; safe to run again
npm run doctor                   # checks that this machine reaches DevStack
npm run devstack -- credential   # makes the credential and prints the values for Clutter
```

| Command | What it does |
| --- | --- |
| `npm run devstack -- create` | makes the VM from `devstack/lima.yaml`, starts it, and installs DevStack with `devstack/install.sh`. A second run does nothing when DevStack is installed. When the VM's address changed, it installs DevStack again for the new address. It refuses a VM named `devstack` that it did not make, and changes nothing in it. The install log is `/opt/stack/stack.log` inside the VM; follow it with `limactl shell devstack tail -f /opt/stack/stack.log` |
| `npm run devstack -- credential` | makes an OpenStack application credential named `clutter`, for the `demo` user in the `demo` project, and prints the values Clutter's Connections page needs. When one exists, it asks before it replaces it; the old one then stops working |
| `npm run devstack -- start` | starts the VM |
| `npm run devstack -- stop` | stops the VM; its disk, and DevStack in it, are kept |
| `npm run devstack -- status` | shows the VM's state, and exits with 1 when it is not running |

`credential` prints the secret once, to the terminal. It is not saved in a file or a log, and `credential` runs only when its output is a terminal, so it never lands in a file by mistake. Copy the secret into Clutter straight away. If you lose it, run `credential` again.

### Let this machine reach the VM (macOS)

macOS asks before an app may reach a device on a local network, and the VM's address counts as one. When `npm run doctor` says DevStack does not answer at the VM's address:

1. Open System Settings, then Privacy & Security, then Local Network.
2. Switch on the app you run the terminal or editor in (Terminal, iTerm, Cursor, VS Code).
3. Run `npm run doctor` again. It should print `ok    DevStack answers at http://<the VM's address>/identity/v3`.

`npm run dev` and the tests reach DevStack through that same app, so start them from it.

### Add the connection in Clutter

1. Start Clutter with `npm run dev` and sign in.
2. Open Organization settings, then Connections, and choose OpenStack.
3. Fill in the form with the values `credential` printed: a name of your choice, the region (`RegionOne`), the auth URL (`http://<the VM's address>/identity/v3`), the application credential ID and the application credential secret.
4. Choose Check and connect. B checks the credential with DevStack's Keystone.

Locally, B accepts an auth URL over plain `http` at a private address, because `dev.sh` starts it with `ADAPTER_ALLOW_PRIVATE_TARGETS=1`. In the cloud it does not. The connection's default settings (the `cirros` image, the `public` external network, and the `m1.tiny`, `m1.small` and `m1.medium` flavors) are DevStack's own names.

### The real-cloud test

`TestRealCloud` in `backend/adapter/connectors/generalopenstack` checks the credential with Keystone, then plans, applies and destroys a VM with a public address, a network with its router, a firewall and an attached volume, on DevStack. It skips unless all four variables below are set, and it needs Terraform on PATH. It builds real resources in the `demo` project, so a person runs it; it is not in CI.

```sh
export CLUTTER_OPENSTACK_AUTH_URL=http://<the VM's address>/identity/v3
export CLUTTER_OPENSTACK_REGION=RegionOne
export CLUTTER_OPENSTACK_APP_CRED_ID=<the application credential ID>
read -rs CLUTTER_OPENSTACK_APP_CRED_SECRET   # paste the secret, then Enter; it is not shown or kept in your history
export CLUTTER_OPENSTACK_APP_CRED_SECRET
go -C backend/adapter test ./connectors/generalopenstack/ -run TestRealCloud -v -count=1 -timeout 30m
```

Optional: `CLUTTER_OPENSTACK_IMAGE` (default `cirros`), `CLUTTER_OPENSTACK_FLAVOR` (default `m1.tiny`) and `CLUTTER_OPENSTACK_EXTERNAL_NETWORK` (default `public`). A failed apply is destroyed before the test ends.

## Without AWS

Every Lambda gets fake credentials, no AWS config files, and `AWS_ENDPOINT_URL` set to one local endpoint on a random loopback port. No AWS request leaves the machine. Two things do reach the internet: the log-in Lambda fetches Google's public keys to check a real Google token, and validating downloads Terraform providers.

That endpoint answers S3 itself and refuses everything else. It serves `GetObject`, `PutObject`, `HeadObject`, `ListObjectsV2` and `DeleteObject`, path-style, on the two local buckets. The data bucket is stored under `.state/s3/<bucket>/<key>`. The templates bucket is read-only. Each S3 call is logged with `[local-s3]`.

Any other call is refused with a 403: ECS, SQS, STS, CloudWatch, any other bucket, and any other S3 operation. A route that needs one of those stops there, and the smoke table marks it `NEEDS-AWS`. The gateway log names each refused call (`[no-aws]`).

## Differences from AWS

- API Gateway's request models (JSON Schema body validation) are not applied. They use regex lookaheads, which Go cannot run. A body that API Gateway would reject with 400 reaches the handler here.
- Each Lambda runs as one long-lived process, like one warm instance, and takes one request at a time, as a Lambda instance does. Parallel requests to the same Lambda queue rather than run side by side (in AWS they would start more instances). aws-lambda-go's RPC mode fails under overlapping calls, which broke the canvas before this rule. The smoke test sends 30 parallel requests to keep it fixed. A timeout or crash replaces the process.
- Lambdas are built for this machine (`go build`), not for `linux/arm64`.
- The local S3 checks no signature, expiry or permission. It accepts a request from any process on this machine, and CORS from any loopback origin.
- The local S3 does not keep an object's content type: it guesses from the file extension. It has no versions, no encryption and no multipart upload.
- Keys are paths on disk, so the local S3 refuses a key with an empty, `.` or `..` segment, and a key that is also a folder of other keys. S3 allows these.

## Guard rails

One devserver runs per machine. Its ports (3000, 3100, 8787, 8790, 54329) and its Docker database are fixed, so two clones on one laptop share them. Separate laptops never collide; each generates its own secrets.

The gateway and B API bind to loopback only. The gateway refuses a JWT secret that `dev.sh` did not generate (prefix `devserver-local-`). It refuses a Postgres that is not on loopback. Secrets, binaries, logs, the local S3 files and B's data live in `.state/`, which git ignores.

`go test ./...` here checks the route table, CORS methods, timeouts and environment keys against `backend/infra/main.tf`, and every `aws_s3_object` against `backend/infra/modules/s3`. It fails when they drift.
