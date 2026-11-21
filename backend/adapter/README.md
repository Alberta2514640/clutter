# B API

B is the adapter layer beside A (`docs/architecture/system-b-adapter-layer.md`). This folder, `backend/adapter`, is its one Go service. The API is `cmd/api`, and the run task is `cmd/runtask`. The service holds the `/v1` API, the connector interface and registry, and the translator from a building-block diagram to a graph. It also holds the capability check, the job runner, the isolated run task, the shared Terraform runner, and the credential store. It shares A's Postgres and imports `backend/api/generic` for sign-in and ownership checks.

## Layout

| Package | Job |
| --- | --- |
| `connectors` | the interface every target implements, its manifest and credential types, and the registry |
| `connectors/contract` | the tests every connector must pass |
| `connectors/mock` | a connector that builds nothing, for tests and local runs |
| `internal/api` | the `/v1` routes |
| `internal/auth` | sign-in middleware over `generic.VerifyUserJWT` |
| `internal/translate` | the building-block diagram document and its translation to a graph |
| `internal/split` | a deployment with blocks placed on other clouds, as parts in the order a run takes them, with its cross-cloud edges (`docs/architecture/system-b-adapter-layer.md` § Split deployments) |
| `internal/capability` | the graph against a manifest, minus a deployment's exclusions |
| `internal/jobs` | the in-process run queue and run states |
| `internal/runtask`, `internal/terraform` | one process per run with only that run's environment; Terraform with state per deployment |
| `internal/credentials` | the local encrypted store and the Secrets Manager store |
| `internal/netguard` | the one list of network addresses B may connect to for a connection. It covers B API's own calls (the OpenStack connector's verification) and the run task's check of a run's hosts before Terraform starts |
| `internal/config`, `internal/logstore` | the settings B reads at start-up; run logs on disk or in S3 |
| `internal/pgtest` | a throwaway Postgres container for the tests |
| `internal/codestore` | uploaded function code: the upload check, and the disk and S3 stores |
| `internal/store` | B's tables, from `backend/storage-sql/adapter-tables.sql` |

## Routes

All `/v1` routes need the `Authorization` header with the JWT A's log-in issues (with or without `Bearer`). Browser calls get CORS headers only from the origins in `ADAPTER_ALLOWED_ORIGINS`, matched exactly. B answers their preflight (`OPTIONS`) itself, allowing `GET`, `POST`, `PUT`, `PATCH` and `DELETE` with `Authorization` and `Content-Type`. With `ADAPTER_ALLOWED_ORIGINS` unset, B sends no CORS headers.

| Route | Does |
| --- | --- |
| `GET /healthz` | 200 when the database answers |
| `GET /v1/catalog` | every connector's manifest and credential spec |
| `GET /v1/examples` | the built-in example diagrams (`internal/examples`, one embedded JSON file each): id, name, level, one-line description, the clouds it fits (only those this build offers) and, for a split example, which blocks go on which cloud, with the building-block document |
| `GET`, `POST /v1/organizations/{org}/connections` | list, or verify and create from `{"connector", "name", "region"}` plus either `"credentials"` or `"linked_id"` (a linked account of the caller's organization). B reads the linked account's secret itself, so it never passes through the browser. A connector whose credential spec is `linked_only` (the catalog says so; AWS is one) takes only `"linked_id"`. Pasted credentials are refused with 400 `linked_only`, because an AWS external id is in the stack link every member of the organization sees (ADR-019) |
| `POST /v1/organizations/{org}/connections/{id}/verify` | check the credentials again |
| `GET /v1/organizations/{org}/connections/{id}/options` | `{"settings": {"<setting>": [{"value", "label"}]}, "suggested": {"<setting>": "<value>"}}`, the values the connection's cloud offers for the connector's deployment settings (`connectors.OptionLister`), so the deploy panel shows lists. Only settings the manifest declares and values it accepts are passed on. A setting the cloud could not list is missing, and an empty list means the cloud has none. `suggested`, left out when empty, names for a setting whose manifest default the cloud does not offer the value the connector recommends instead; B keeps a suggestion only when it is one of that setting's choices. No credential is in the answer. 502 `options_failed` with the cloud's reason when nothing could be listed, 501 `no_options` for a connector without lists. For OpenStack: the flavors, active images and external networks the credential's project sees in the connection's region. They are read with one Keystone token through the same address rules as verification, and cached per connection for 10 minutes. OpenStack suggests only where a DevStack default is missing. For the image: an active Ubuntu image built for x86_64 that is not a Kubernetes image, LTS first, then the newest release, then the newest upload. For each flavor setting: the smallest flavor with at least 1, 2 or 4 GB of RAM (small, medium, large), one with a local disk first, then fewer vCPUs. For the external network: the cloud's only external network, when it has exactly one |
| `DELETE /v1/organizations/{org}/connections/{id}` | delete, refused with 409 while a deployment on it, or one that places blocks on it, is live. Every answer for a connection carries `diagrams`, how many diagrams deploy to it (as its own or through a placement), whose deployments a delete removes |
| `POST /v1/organizations/{org}/diagrams/{diagram}/deployments` | bind the building-block diagram in the path to a connection: `{"connection_id", "settings"?, "exclusions"?, "placements"?}`. Each placement is `{"connection_id", "node_ids", "settings"?}`: those blocks run on another connection of the organization, with that cloud's settings (`docs/architecture/system-b-adapter-layer.md` § Split deployments). Another organization's connection, as primary or placement, is 404, and nothing is made. The answer is 422 `invalid_placements` for the primary itself, a connection named twice, a placement with no block, a block placed twice, or more than 8. It is 422 `unknown_block` for a block not in the saved diagram, and 422 `invalid_settings` for a placed cloud's setting its manifest does not take. Every answer for a deployment carries its `placements` (`connection_id`, `connection_name`, `connector`, `region`, `node_ids`, `settings`) |
| `GET /v1/organizations/{org}/connectors/{connector}/linked-accounts` | the accounts the organization already linked outside B, by name only, for a connector that offers them (`connectors.Linker`; the catalog marks it `has_linked_accounts`). 501 for any other connector. For AWS, these are the roles linked through A's CloudFormation flow |
| `GET /v1/organizations/{org}/diagrams/{diagram}/deployments` | one diagram's deployments, oldest first, each with its connection's name, its latest run (`latest_run`, or null), its latest completed apply (`last_apply`, or null) and the saved diagram's version (`diagram_version`). A `diagram_version` newer than `last_apply.diagram_version` means the diagram has changes not deployed yet |
| `GET /v1/organizations/{org}/diagrams/{diagram}/code` | the code uploaded for the diagram's function blocks: `{"items": [{"node_id", "file_name", "size", "sha256", "uploaded_at"}]}` (§ Function code) |
| `PUT /v1/organizations/{org}/diagrams/{diagram}/code/{node}?file_name=<name>.zip` | the request body is the block's new code, a zip; answered with 200 and the item. Refused with 422 `unknown_block` for a block not in the saved diagram, 422 `not_a_function` for any other block, and 422 `invalid_file_name`. Refused with 415 `not_zip` for a `Content-Type` other than `application/zip`, `application/x-zip-compressed` or `application/octet-stream`. Refused with 413 `too_large` over 50 MB, and with 422 `invalid_code` (with the reason) for content that is not a safe zip |
| `DELETE /v1/organizations/{org}/diagrams/{diagram}/code/{node}` | remove the block's code (204; 404 when it has none), so the function deploys with Clutter's sample code again |
| `PATCH /v1/organizations/{org}/deployments/{id}` | `{"settings"?, "exclusions"?, "placements"?}`: replaces whichever it holds. Settings are checked against the connector's manifest (a null value unsets one), exclusions against the diagram's block ids, and placements as on create. The change is refused with 409 while a run is queued or running. It is refused with 409 `deployment_live` for placements that move a block while the deployment may hold resources (live or failed). Placements that change only a placed cloud's settings are allowed then |
| `DELETE /v1/organizations/{org}/deployments/{id}` | remove a deployment and its runs, answered with 204, so its connection is free for a new deployment of the diagram. Only a new or destroyed deployment can go. The answer is 409 `deployment_live` while it may hold resources (live or failed), and 409 `run_active` while a run is queued or running |
| `GET /v1/organizations/{org}/deployments/{id}/compatibility` | `{"ready", "problems", "notes"}`: what this cloud cannot build. A split deployment adds `clouds`, one report per cloud with `connection_id`, `connection_name`, `connector` and `position` (its place in a run's order). Its own problems and notes then gather every cloud's, plus the split's. The split's are an edge that cannot cross clouds (`cross_cloud_edge`), clouds that hand each other values in a circle (`cross_cloud_cycle`), and a cloud whose connector cannot take part (`cloud_cannot_split`). They also include a note per cross-cloud edge (`cross_cloud_key`, "Clutter creates a key that can only read files and puts it on web. ...", and `cross_cloud_address`) |
| `GET /v1/organizations/{org}/deployments/{id}/preview` | `{"files": {"name": "text"}}`, the Terraform a run would use, rendered without credentials. For a split deployment, each file is named `<connection name>/<file>`, and every value another cloud hands a VM is a placeholder. 422 while the deployment is not ready, 501 for a connector without a preview (`connectors.Previewer`) |
| `POST /v1/organizations/{org}/deployments/{id}/runs` | `{"action": "plan" \| "apply" \| "destroy"}`, answered with 202. The answer is 409 while the deployment has a run queued or running, and 429 `too_many_runs` while the organization has `ADAPTER_MAX_RUNS_PER_ORG` runs queued or running |
| `GET /v1/organizations/{org}/runs/{id}` and `.../logs` | a run's status, and its log as text. A split run's `parts` lists each cloud in the order the run takes them. Each entry has `connection_id`, `name`, `step` (`plan`, `apply`, `destroy` or `rollback`) and `status` (`queued`, `running`, `completed`, `failed`, `skipped` or `rolled_back`). `parts` is null for a run on one cloud |

Errors use one body: `{"error": {"code", "message", "problems"?}}`.

## Running it

Locally, `npm run dev` (or `backend/devserver/dev.sh --backend-only`) builds B and starts it on `127.0.0.1:8790` beside A, with the mock, OpenStack and AWS connectors, a local credential key and A2's JWT secret and Postgres. `npm run stop` stops it, and `npm run smoke` checks its main paths (`backend/devserver/README.md` § B API locally).

To run it by hand, from the repo root, with Postgres holding A's schema:

```sh
go -C backend/adapter build -o /tmp/adapter-api ./cmd/api
ADAPTER_LISTEN_ADDR=127.0.0.1:8790 \
ADAPTER_DATABASE_URL=postgres://... \
ADAPTER_SCHEMA_FILE="$PWD/backend/storage-sql/adapter-tables.sql" \
ADAPTER_DATA_DIR=/absolute/data/dir \
JWT_SECRET=... \
ADAPTER_CREDENTIAL_STORE=local \
ADAPTER_CREDENTIAL_KEY="$(head -c 32 /dev/urandom | base64)" \
ADAPTER_CONNECTORS=mock \
/tmp/adapter-api
```

`ADAPTER_CONNECTORS` lists the connector ids to offer: `mock` (tests and A2 only), `general-openstack` and `aws`. It has no default, so the hosted service never offers the mock by accident. A2 sets `mock,general-openstack,aws`. The `aws` connector also needs exactly one of `ADAPTER_AWS_TEMPLATE_DIR` (the repo's `backend/infra/modules/s3/uploads/aws`, locally) or `ADAPTER_AWS_TEMPLATE_BUCKET` (the platform's template bucket, in the cloud). It also needs `ADAPTER_AWS_STS`, which takes one of three values. `sts` is for the cloud, and for real AWS from a laptop (`npm run real-aws`, `backend/devserver/README.md` § Real clouds from this machine), where B takes the runner profile named by `AWS_PROFILE` from `~/.aws`. `local` (A2 only, refused unless `ADAPTER_CREDENTIAL_STORE=local`) verifies a well-formed role and stops every run with "needs a real AWS account". `localstack` (A2 only, refused unless `ADAPTER_CREDENTIAL_STORE=local`) goes with `ADAPTER_AWS_ENDPOINT` (a loopback URL such as `http://127.0.0.1:4566`). It gets each run's keys from LocalStack's STS and sends every AWS call of the run to that endpoint (`backend/devserver/README.md` § AWS on LocalStack). `ADAPTER_AWS_ENDPOINT` is refused with any other `ADAPTER_AWS_STS`. `JWT_SECRET` is the one A's log-in signs with. In the cloud, `ADAPTER_CREDENTIAL_STORE=secretsmanager` with `ADAPTER_SECRETS_PREFIX` replaces the local key.

The rest are optional. `ADAPTER_RUN_LAUNCHER` is `local`, the default, or `ecs` (§ In the cloud). `ADAPTER_LOG_STORE` is `local`, the default, which keeps run logs under `ADAPTER_DATA_DIR`, or `s3`, which keeps them in the bucket `ADAPTER_LOG_BUCKET`. `ADAPTER_TERRAFORM_BIN` defaults to `terraform` on the path, and only the local launcher reads it. `ADAPTER_WORKERS` defaults to 2. `ADAPTER_RUN_TIMEOUT` defaults to 1h. `ADAPTER_MAX_RUNS_PER_ORG` defaults to 3: one organization's runs queued or running at once. `ADAPTER_ALLOWED_ORIGINS` takes comma-separated browser origins such as `http://localhost:3000`. A wildcard or a path is refused. `ADAPTER_ALLOW_PRIVATE_TARGETS=1` (A2 only, refused unless `ADAPTER_CREDENTIAL_STORE=local`) lets the OpenStack connector verify an `http` auth URL at a loopback or private address, as a laptop's DevStack has. Without it, an auth URL must be `https`, and B connects to no loopback, private, shared or link-local address. A missing required value stops the start with the variable's name.

Every run's Terraform is checked before `init`. B parses each rendered `.tf` file. It refuses a provisioner, the `external` or `http` data source, a module, a `backend` or `cloud` block, and any provider source outside the connector's own list. The run also gets its own Terraform CLI configuration (`TF_CLI_CONFIG_FILE`), which installs only those providers and only from `registry.terraform.io`.

## Function code

A function block's code is one zip per organization, diagram and block, uploaded with the `PUT .../code/{node}` route, and every deployment of the diagram uses it. The upload goes to B in both places, not to a presigned S3 address, so B checks every byte before anything is stored.

| Limit | Refused |
| --- | --- |
| at most 50 MB (Lambda's limit for a direct zip upload) | 413, before the body is read when `Content-Length` says so, otherwise when the body passes the limit |
| a `.zip` file name of at most 255 characters, with no folder | 422 `invalid_file_name` |
| content that is a zip archive with at least one file | 422 `invalid_code` |
| no entry with an absolute path, a drive letter, a backslash, a control character or a `..` step. No link whose target leaves the zip's folder (a link inside it, such as `node_modules/.bin`, is fine) | 422 `invalid_code` |
| at most 20,000 entries and 250 MB unpacked (Lambda's unpacked limit). B counts by unpacking each entry, so a header that understates a file's size does not pass | 422 `invalid_code` |
| only a function block that is in the saved diagram | 422 `unknown_block` or `not_a_function` |

B holds the upload in a temporary file under `ADAPTER_DATA_DIR/uploads` while it checks it. B records its size and SHA-256 in `function_code` (`adapter-tables.sql`). The zip itself follows `ADAPTER_LOG_STORE`. Locally it is a file under `ADAPTER_DATA_DIR/code`. In the cloud it is an object under `code/` in `ADAPTER_LOG_BUCKET` (the run bucket). B API's role may put, get, delete and list those objects, and no lifecycle rule expires them. Its key names the organization, diagram, block and content hash. A zip is kept until no `function_code` row names it. When the user replaces or removes a zip, B deletes the old object at once. Deleting a diagram, project or organization through A removes the rows by cascade, and B never sees that delete. So B's code sweep finds those objects. It runs when B API starts and then every hour. It reads at most 1,000 keys a pass (continuing where the last pass stopped), and deletes each object no row names. Its log says how many objects it deleted, and names a diagram and block only when a delete fails. The sweep also removes an object whose own delete failed.

Every delete of an object, by an upload, a removal or the sweep, happens under the block's lock and only when no row names the object. An upload stores its zip before it takes that lock. Then, under the lock, it checks the zip is still there before it records it. If a delete removed the zip in between, the upload stores it again. So a recorded zip is always in the store, even when uploads, removals and the sweep overlap.

At each plan or apply, B puts the recorded key and hash on the function block (`connectors.FunctionCode`). The connector names the zip to the Terraform runner. The runner copies it from the store into the run's folder as `code-<address>.zip`. It refuses the zip unless its SHA-256 matches the record and it passes the upload's path check again. It refuses any other archive: the rendered files are only `.tf` files, and the runner alone chooses which archives a run holds. The AWS connector renders the function's `filename` as that name and its `source_code_hash` from the recorded hash. A function with no upload gets Clutter's sample code, zipped in the run with every file executable: a few lines that answer every request with status 200 and `{"message": "Hello from Clutter. Upload your code to replace this."}`, in the shape API Gateway's proxy integration reads. For a custom runtime (`provided.al2023`, the default) it is a `bootstrap` shell script that uses only `sh` and `curl`, so it serves any handler; for Python and Node.js it is a file named after the handler (`lambda_function.py`, `index.mjs`), so it serves a handler of the form `file.function`. For any other handler the zip holds one text file and the function does not answer. A function that runs the sample reports the result `code sample`, and "What was built" says "Answers with a sample until you upload your code." A zip replaced or removed after a run was queued stops that run with "The code of a function changed or was removed after this run was queued. Start the run again."

## Server setup at first boot

A VM block's `startup_script` field is a shell script the user writes, at most 16 KB (a manifest `text` field). Cloud-init runs it once, as root, when the VM first starts, on every cloud. The AWS connector renders it as the EC2 instance's `user_data_base64` with `user_data_replace_on_change`. The OpenStack connector renders it as the Nova server's `user_data`. So on both, a changed script builds a new VM. On AWS, the values a VM's links hand it (another VM's address, a bucket or table name) are also in the instance's `replace_triggered_by`, so a linked VM that is being replaced replaces this one in the plan, not halfway through the apply. That list is in A's `ec2/main.tf.tmpl`, which renders it only when B sets it, because Terraform ignores `replace_triggered_by` in an override file (ADR-026). `connectors.StartupUserData` owns the rules. Windows line ends become Unix ones. A script without a `#!` line runs with `/bin/sh`. A script with a control character other than a tab or a line break is refused. The script reaches the rendered Terraform only as base64, never as text inside an HCL string or heredoc.

At start-up B applies `adapter-tables.sql`, fails any run a previous stop left queued or running, and logs `listening`. SIGINT or SIGTERM stops it within 30 seconds.

## In the cloud

In the cloud, B API is one Fargate task in B's own ECS cluster, behind a load balancer. Each run is its own short-lived Fargate task in the same cluster. The platform Terraform creates both and sets every variable (`backend/infra/modules/adapter-service`). `docs/architecture/environments.md` says how to deploy them.

| Part | Where |
| --- | --- |
| B API's image | `images/api.Dockerfile`: the Go binary and `adapter-tables.sql`, no shell and no Terraform |
| The run task's image | `images/runtask.Dockerfile`: `cmd/runtask` and Terraform 1.14.4 |
| Building and pushing both | `images/build-and-push.sh`, with `AWS_REGION` and a new `IMAGE_TAG`; `npm run deploy -- prod --adapter-images` runs it |
| The cloud launcher | `internal/runtask`'s `ECS` |
| Run logs in the cloud | `internal/logstore`'s `S3` |

The same binary runs in both places; two settings choose what differs.

| | Locally (`local`, the default) | In the cloud |
| --- | --- | --- |
| `ADAPTER_RUN_LAUNCHER` | each Terraform command is a child process of B, with state in `ADAPTER_DATA_DIR/state` and the `terraform` binary on B's machine | `ecs`: each run is one Fargate task, with state in the state bucket. B API needs no Terraform and does not look for it |
| `ADAPTER_LOG_STORE` | run logs are files under `ADAPTER_DATA_DIR/logs` | `s3`: run logs are objects under `logs/` in `ADAPTER_LOG_BUCKET` (the run bucket), kept when B API's task stops |

With either launcher, B renders and checks the run's files in a folder under `ADAPTER_DATA_DIR/work`. It writes the state backend itself (a local file, or the S3 backend the cloud launcher names). It removes the folder when the run ends. A rendered file can never set the backend. A run counts as having built nothing when it failed before Terraform could reach the cloud. That is when it was refused, the task never started or could not prepare, or `terraform init` failed.

The S3 log store keeps a log in memory while the run writes it. It uploads the whole object every three seconds and when the run ends. B API reads a running run's log from memory and a finished run's log from the bucket. The bucket keeps run logs for 90 days, then its lifecycle rule removes them. Its one-day expiry covers `runs/` only.

A run's log is the user's. When the launcher cannot start a run (a refused upload, a task that never started or could not prepare), the run's log and its failure say only "Clutter could not start this run. Try again; if it keeps failing, contact support." The detail, which names Clutter's own buckets, roles and ECS reasons, goes to B's service log. A connector's `UserError` message still reaches the user as it is.

The cloud launcher runs a whole run in one task. For each run, it assumes the run-files role (`ADAPTER_RUN_FILES_ROLE_ARN`) for at most one hour. The session policy allows only that run's objects in the run bucket and that deployment's state in the state bucket. It uploads the run folder and the run's environment to `runs/<run>/in/`, and writes those short-lived keys into `runs/<run>/in/task.env`. It starts the task with only the run id, the bucket, the region and the address of `task.env` in its overrides. ECS reads `task.env` with the task's execution role, so no key or credential appears in the task's description. The task has no role of its own. It runs each step with the run's environment only, and gives `terraform init` the keys for the S3 backend. It uploads its log every five seconds and its result at the end. A run that hands values to another cloud (a part of a split deployment) ends with `terraform output -json`. The task uploads that step's output as `runs/<run>/out/outputs.json`, never into the log. B reads it before it deletes the run's objects. Locally the same step's output goes straight to the connector. B copies the log into the run's log as it grows, and reads the exit code when the task stops. It stops the task if the run is cancelled or times out, and deletes the run's objects. State is `state/<deployment>/terraform.tfstate` in the state bucket, encrypted and locked by S3's lock file (`use_lockfile`).

Its settings, all set by the platform Terraform: `ADAPTER_RUN_LAUNCHER=ecs`, `ADAPTER_LOG_STORE=s3` and `ADAPTER_LOG_BUCKET`, `AWS_REGION`, `ADAPTER_RUN_CLUSTER`, `ADAPTER_RUN_TASK_DEFINITION`, `ADAPTER_RUN_CONTAINER`, `ADAPTER_RUN_SUBNETS` and `ADAPTER_RUN_SECURITY_GROUPS` (comma-separated), `ADAPTER_RUN_ASSIGN_PUBLIC_IP` (`true` in B's VPC, which has no NAT), `ADAPTER_RUN_BUCKET`, `ADAPTER_STATE_BUCKET` and `ADAPTER_RUN_FILES_ROLE_ARN`. `ADAPTER_RUN_TIMEOUT` is `55m` there, under the one-hour limit of a run's keys. A missing or malformed setting stops the start with the variable's name. B API runs as one copy, because its run queue is in its own process.

## Checks and expected results

| Command | Expected |
| --- | --- |
| `gofmt -l backend/adapter` | no output |
| `go -C backend/adapter build ./...` | no output |
| `go -C backend/adapter vet ./...` | no output |
| `go -C backend/adapter test ./...` | 19 packages `ok` (checked 2026-11-15): `cmd/runtask`, `connectors`, `connectors/aws`, `connectors/generalopenstack`, `connectors/mock`, and under `internal/`: `api`, `auth`, `capability`, `codestore`, `config`, `credentials`, `examples`, `jobs`, `logstore`, `netguard`, `runtask`, `split`, `terraform` and `translate` |

The `internal/api` tests start a throwaway Postgres container (the image A2 uses) and load A's schema, the seed and `adapter-tables.sql`. Without Docker they skip. The `internal/terraform` test that runs real Terraform skips when `terraform` is not installed. The AWS connector's LocalStack test plans, applies and destroys every AWS block for real through the job runner. It skips unless `CLUTTER_LOCALSTACK=1` (`backend/devserver/README.md` § AWS on LocalStack).
