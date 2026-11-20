# Environments: local and cloud

Clutter runs in two places, and each has one job (ADR-015). Only local exists today. A separate test environment is added when real outside users arrive; the deploy command already takes the environment as a parameter.

| | Local | Cloud |
| --- | --- | --- |
| **Job** | build and test on your laptop | the hosted product: demos now, real users later |
| **Status** | works | not set up |
| **Who uses it** | each developer alone | the team, the client, and later users |
| **Costs money** | no | yes |
| **Frontend** | `npm run dev` on port 3100, opened through A2's front door on port 3000 | a Fargate service behind B's load balancer, its image built with A's and B's addresses and pushed by `npm run deploy -- prod --site-image` |
| **Backend** | the real Lambdas, run by `backend/devserver` | real Lambdas behind API Gateway, and B's container on Fargate, in the client's AWS account in `ca-central-1` |
| **Database** | Postgres in Docker | a Supabase project |
| **File storage (S3)** | the devserver's local S3, on disk | real S3 |
| **Secrets live in** | `backend/devserver/.state/`, generated for you | `prod.tfvars` on the deployer's machine, never in git; users' cloud credentials in AWS Secrets Manager (ADR-016) |
| **Terraform state (platform)** | none | a state bucket in the client's account |
| **Terraform state (B's deployments)** | files in B's data folder | B's state bucket, one file per deployment, locked |
| **Where B's runs happen** | a child process of B | one Fargate task per run, in B's own cluster |
| **Where diagrams deploy (world 2)** | a fake AWS account; DevStack for OpenStack | our test AWS account and a DevStack VM for demos, then real users' own accounts |
| **Data** | fake, wipe it any time | test data until real users arrive |

## What each feature does where

The cloud does not exist yet, so nothing has been tested there. Its column says what is expected once it is set up.

| Feature | Local (tested) | Cloud (expected) |
| --- | --- | --- |
| Sign in | Google, through the capstone's client for now | Google, the client's own client |
| Organizations, projects, diagrams | works | works |
| Canvas: drag, connect, configure, save | works | works |
| Link an AWS account | fake account only | our test account, then real users' accounts |
| Generate Terraform on save | works (the editor calls generation after each save) | works the same way |
| Read and download Terraform in the editor | works | works, once generation runs |
| Upload Lambda code or a playbook | stored on disk | real S3 |
| Deploy and destroy from A's editor | does not run | the first real test happens here |
| Deploy and destroy a building-block diagram through B | runs on the mock cloud, on DevStack for OpenStack, and on LocalStack for AWS (`backend/devserver/README.md`) | runs in the users' own clouds; the first real AWS test happens here |
| Live deploy status, real deploy logs | does not run | runs with deploy |
| Run Ansible on a server | does not run | runs, needs a real EC2 server |
| `npm run smoke`, `npm run generated` | team tools | not available |

## The rules

- **Local and cloud share nothing:** no database, Google client, secret or state bucket.
- **Cloud data never goes to a laptop.**
- **The same code runs in both.** Only configuration differs. Code that branches on the environment name is a defect; the devserver stands in for AWS from outside the Lambdas.
- **Change moves one way:** local, then pull request (with the checks), then `main`, then the cloud. The cloud is deployed only from a clean `main`.
- **A person deploys.** No session or agent runs `terraform apply`.

## What each environment needs

**Local:** nothing beyond the install list in the root `README.md`. No env file, no accounts.

**Cloud.** Clutter in the cloud is a hosted service (SaaS): one copy, run for everyone. The accounts that run Clutter itself (world 1, `system-b-adapter-layer.md` § Names) belong to the client, who owns the product and pays for running it, and the team gets access to deploy. Users bring a Google sign-in and their own cloud accounts (world 2), which they pay for. Stages 1 to 6 are done, so this is set up when the client's accounts arrive (`implementation-plan.md`):

| Thing | Owned by | Goes into |
| --- | --- | --- |
| An AWS account for Clutter, region `ca-central-1` | the client; the team gets access through its own user logins | AWS credentials on the deployer's machine |
| A Supabase project | the client; the team sets it up | `psql_connection_string` |
| A Google OAuth client | the client; the team sets it up | the frontend, and `GOOGLE_CLIENT_ID` on the log-in Lambda |
| A domain and DNS access | the client | `site_domain` (the site's one name, such as `app.example.com`, which serves the website and B), and the Google client's allowed origin |
| A certificate for that name | the client; the team requests it in AWS Certificate Manager in `ca-central-1` and validates it through DNS | `certificate_arn` |
| Where the website runs | Clutter's own AWS account: a Fargate service behind B's load balancer (`backend/infra/modules/website`) | `site_domain` and `site_image_tag` |
| Terraform inputs | filled in from the rows above (§ Credentials day) | `backend/infra/prod.tfvars` and `prod.backend.hcl`, copied from the `.example` files beside them |

**Test accounts for world 2** (where diagrams deploy during testing and demos):

| Thing | Owned by | Why |
| --- | --- | --- |
| A second AWS account, acting as a user | the client or the team; easiest as a second account in the same AWS organization | proves a real cross-account deploy, as a user's account would be |
| An OpenStack | the team | locally DevStack on a laptop; for the cloud, a DevStack virtual machine in the test account, switched on only for demos, until Patrii opens |
| A DigitalOcean account (optional) | the team | the second General connector (`general-digitalocean`) |

## Credentials day

This section is the one owner of the go-live steps. The capstone deployment is gone: its API host (`7c1aawofvl.execute-api.us-west-2.amazonaws.com`) returns NXDOMAIN, and its platform state bucket belongs to the capstone AWS account, which we cannot reach. No step below has been run yet. What is missing is accounts and secrets, not tools. Every step marked **(person)** creates real resources, costs money or needs a person's login, so a person runs it, never a session.

`npm run deploy:plan -- prod` shows what would change, and `npm run deploy -- prod` applies it. They stop on incomplete or example config (`backend/infra/check-config.sh`), on the wrong AWS account, and on anything but a clean `main`; they ask you to type `prod` before applying, and there is no tear-down command. `prod` is the only environment they accept (ADR-015). They have not been run against real AWS yet; `backend/infra/test-deploy.sh` tests only their checks. Stage 1 (phase 0 in `.claude/findings/NOW.md`) must be on the `main` you deploy from before step 9 on any account reachable from the internet. The fork's `main` has its fixes, such as the signed-in generation endpoint. The client's `main` does not until the fork's `main` reaches it, so there the generation endpoint still needs no login.

Before Clutter is hosted, what users build (world 2) can be tested from a laptop against real AWS and OpenStack accounts: `backend/devserver/README.md` § Real clouds from this machine.

### Every value the cloud needs

The two files are ignored by git; each has a committed example beside it with a comment per key. `backend/infra/check-config.sh prod` reads the real files and names every key that is missing, still an example value, or of the wrong form, without printing any value. The capstone team shared their tfvars file on Discord; it is not in the repo and must not be. Values the platform and the deploy command can work out are not asked for: the CloudFormation template's address, every B setting (`backend/infra/modules/adapter-service`), B's database string and JWT secret in Secrets Manager (from `psql_connection_string` and `jwt_secret`), the site's origin that A and B answer CORS for (`https://` and `site_domain`), and the website's image build values that come from elsewhere (A's API address from the `api_invoke_url` output, and B's address, which is the site's own origin); the image also takes `google_client_id` and `contact_email` from `prod.tfvars`.

| Value | What it is | Where to get it | Goes in | Fake example |
| --- | --- | --- | --- | --- |
| AWS sign-in | the deployer's access to Clutter's account | AWS IAM Identity Center: `aws configure sso`, then `aws sso login` | `~/.aws` on the deployer's machine, never the repo | profile `clutter-prod` |
| `aws_region` | where everything runs (ADR-015) | fixed | `backend/infra/prod.tfvars` | `ca-central-1` |
| `clutter_account_id` | Clutter's own AWS account; the deploy command refuses any other | AWS console, account menu (top right), or `aws sts get-caller-identity --query Account --output text` | `prod.tfvars` | `123456789012` |
| `jwt_secret` | signs A's log-in tokens; B checks them | generate: `openssl rand -hex 32` | `prod.tfvars` | 64 hex characters |
| `psql_connection_string` | Supabase's session pooler (IPv4; the direct host is IPv6 only) | Supabase dashboard: the project's **Connect** button, **Session pooler**; the password from Project Settings, Database | `prod.tfvars` | `postgres://postgres.PROJECT-REF:PASSWORD@aws-0-ca-central-1.pooler.supabase.com:5432/postgres` |
| `site_domain` | the site's one DNS name: the load balancer sends its paths `/v1/*` and `/healthz` to B API and everything else to the website; A and B answer CORS for `https://` and this name alone | the domain the client picks for the site; lower case, no `https://`, no path | `prod.tfvars` | `app.example.com` |
| `ansible_runner_image_tag` | A's Ansible runner image tag; a new one for each push, never `latest` | you choose | `prod.tfvars` | `v1` |
| `terraform_deployer_image_tag` | A's Terraform deployer image tag; a new one for each push, never `latest` | you choose | `prod.tfvars` | `v1` |
| `google_client_id` | the Google OAuth client for sign-in | Google Cloud console: APIs & Services, Credentials, the Web client's **Client ID** | `prod.tfvars` | `NUMBER-ID.apps.googleusercontent.com` |
| `contact_email` | the address the website's footer, FAQ, privacy policy and terms show; the website's image is built with it | the client's contact address: `devinatkin@gmail.com` (confirm with the client before launch) | `prod.tfvars` | `hello@example.com` |
| `adapter_image_tag` | the tag of B's two images; a new one for each push | you choose | `prod.tfvars` | `v1` |
| `site_image_tag` | the tag of the website's image; a new one for each push, never `latest` | you choose | `prod.tfvars` | `v1` |
| `certificate_arn` | the HTTPS certificate for `site_domain` | AWS console: Certificate Manager in `ca-central-1`, the certificate's ARN once it shows **Issued** (step 6) | `prod.tfvars` | `arn:aws:acm:ca-central-1:123456789012:certificate/REPLACE` |
| `bucket` | the platform's own state bucket | `aws s3 ls \| grep tfstate-ca-central-1-` after step 3 | `backend/infra/prod.backend.hcl` | `tfstate-ca-central-1-REPLACE` |
| `key` | the state file's key | fixed | `prod.backend.hcl` | `clutter/infra/terraform.tfstate` |
| `region` | the state bucket's region, the same as `aws_region` | fixed | `prod.backend.hcl` | `ca-central-1` |
| `dynamodb_table` | the platform state's lock table, made by step 3 | fixed | `prod.backend.hcl` | `tflock` |
| `encrypt` | encrypts the platform state | fixed | `prod.backend.hcl` | `true` |
| Authorized JavaScript origin | the site's origin, allowed to sign in with the Google client: `https://` and `site_domain`, the same origin A and B answer CORS for | Google Cloud console: the Web client, **Authorized JavaScript origins** | Google Cloud, not a file | `https://app.example.com` |
| Certificate validation record | proves to AWS that the client owns `site_domain` | Certificate Manager shows a CNAME after the request | the DNS provider | `_a1b2.app.example.com` CNAME `_c3d4.acm-validations.aws` |
| The site's DNS record | sends `site_domain` to the load balancer, which routes B's paths to B API and the rest to the website | the `load_balancer_dns_name` output after step 9 | the DNS provider | `app.example.com` CNAME `clutter-adapter-1234.ca-central-1.elb.amazonaws.com` (an apex name needs the DNS provider's alias or ALIAS record) |
| Supabase Data API | off, or row-level security on, before any data is loaded | Supabase dashboard: Project Settings, Data API | Supabase, not a file | off |
| The test AWS account | a second account acting as a user (world 2) | AWS Organizations: **Add an AWS account** | linked in the site, step 12 | `444455556666` |
| DevStack credentials | an application credential on the demo DevStack VM | the VM's Horizon or `openstack application credential create` | entered in the site's Connections page, step 13 | `clutter-demo` |

### The steps, in order

1. **(person) The accounts.** The client's AWS account for Clutter (`ca-central-1`), a second AWS account for tests, a Supabase project in the Canada (Central) region, a Google Cloud project, and access to the DNS of the client's domain. Done when each team member who deploys can sign in to each.
2. **(person) Sign in to Clutter's AWS account** on your machine: `aws configure sso`, then `aws sso login`. Success: `aws sts get-caller-identity` shows `clutter_account_id`. If it fails, ask the account's owner for a permission set in IAM Identity Center.
3. **(person) The platform's state bucket,** once per account: `terraform -chdir=backend/infra-state-bootstrap init`, then `terraform -chdir=backend/infra-state-bootstrap apply -var aws_region=ca-central-1`. Success: `Apply complete! Resources: 5 added`, and `aws s3 ls | grep tfstate-ca-central-1-` shows the bucket, the `bucket` value. Its own state stays in `backend/infra-state-bootstrap` (ignored by git); keep it, because it is needed only to change or remove that bucket. If it fails with access denied, the sign-in lacks rights.
4. **(person) Supabase.** Turn off the Data API (or turn on row-level security) first. Then load A's schema into the new, empty project, reading the address from the file so it stays out of your shell history: `psql "$(sed -nE 's/^psql_connection_string *= *"(.*)"/\1/p' backend/infra/prod.tfvars)" -v ON_ERROR_STOP=1 -f backend/storage-sql/application-tables.sql`, then the same with `seed-resources.sql` (after step 7 has made `prod.tfvars`). B applies its own `adapter-tables.sql` when it starts. Success: psql ends without `ERROR`. If it fails to connect, check the session pooler address and the percent-encoded password. No webhook is configured in Supabase: the editor calls the signed-in `POST /terraform-engine` after each save (ADR-018).
5. **(person) The Google OAuth client:** Google Cloud console, APIs & Services: set up the OAuth consent screen, then Credentials, **Create credentials**, **OAuth client ID**, Web application, with `https://` and `site_domain` (such as `https://app.example.com`) as an authorized JavaScript origin. Success: the console shows the client id. It goes in `google_client_id` only: the log-in Lambda reads it as `GOOGLE_CLIENT_ID`, and the deploy builds the website with it.
6. **(person) The certificate:** Certificate Manager in `ca-central-1`, **Request**, public, the one name `site_domain` (such as `app.example.com`), DNS validation; add the CNAME it shows at the DNS provider. Success: the certificate shows **Issued**, usually within minutes. If it stays pending, the CNAME is wrong or not yet published.
7. **Fill the files.** `cp backend/infra/prod.tfvars.example backend/infra/prod.tfvars`, `cp backend/infra/prod.backend.hcl.example backend/infra/prod.backend.hcl`, and fill in every value from the table. Then `backend/infra/check-config.sh prod`. Success: `config: complete`. If not, it names each key to fix. Anyone may run it; it prints no value.
8. **(person) Plan:** `npm run deploy:plan -- prod`. It checks the files and the account, builds the Lambdas (`backend/api/build.sh`, all 34), runs `terraform init` with `prod.backend.hcl`, and plans. Success: `Plan: N to add, 0 to change, 0 to destroy` on the first run, and `Plan only: nothing changed.` If `init` fails, the `bucket` or `region` in `prod.backend.hcl` is wrong.
9. **(person) The first deploy:** `npm run deploy -- prod`, from a clean `main`, then type `prod`. It creates A, B's part of the platform and the website. B's part (`backend/infra/modules/adapter-service`) is B's own VPC (public subnets for B API and the load balancer, and separate run subnets whose network ACL refuses private and link-local addresses), its own ECS cluster `clutter-adapter`, B API as one Fargate task, the load balancer (HTTPS with `certificate_arn`), the run task definition, the ECR repositories `clutter-adapter-api` and `clutter-adapter-runtask`, the run and state buckets, B's roles and B's two secrets. The website (`backend/infra/modules/website`) is its own ECS cluster `clutter-website`, one Fargate task of the Next.js server, the ECR repository `clutter-website`, a role that can only pull that image and write its log group `/ecs/clutter-website`, and a security group that only the load balancer can reach. The load balancer serves `site_domain` only and routes by path: `/v1/*` and `/healthz` to B API (health check `/healthz`), everything else to the website (health check `/`); any other name gets a 404. It also publishes A's API in the stage `prod`, and makes a new deployment of it whenever a file that defines the API changes, so no deploy needs a manual publish step. Success: `Apply complete!`, A's API address (the `api_invoke_url` output) and the load balancer's name. A's runner tasks, B API and the website wait for their images until step 11; that is expected. This first run takes no image flag, because the image repositories and A's API (whose address the website is built with) do not exist before it. If the apply stops partway, fix the error it names and run the same command again.
10. **(person) The DNS record:** at the DNS provider, point `site_domain` at `terraform -chdir=backend/infra output -raw load_balancer_dns_name` (a CNAME, or the provider's alias record for an apex name). Success: `dig +short app.example.com` shows the load balancer's addresses once DNS has spread.
11. **(person) The images, and the last apply:** `npm run deploy -- prod --a-images --adapter-images --site-image`. After you type `prod`, it builds and pushes A's two runner images with `terraform_deployer_image_tag` and `ansible_runner_image_tag` (`backend/infra/modules/fargate/ecr-scripts/*/build-and-push.sh`), B's two images with `adapter_image_tag` (`backend/adapter/images/build-and-push.sh`), and the website's image with `site_image_tag` (`frontend/build-image.sh`), then applies. The website's image is built with A's API address from the `api_invoke_url` output, B's address (the site's own origin, `https://` and `site_domain`), `google_client_id` and `contact_email`, so nobody types them. A tag ECR already holds is not rebuilt, so changing one tag ships only that image; the repositories are immutable, so a new image needs a new tag in `prod.tfvars`. Each flag also works alone, for example `--site-image` after a change to the website. Success: each image pushed, then `Apply complete!`; in the ECS console the services `clutter-adapter-api` (cluster `clutter-adapter`) and `clutter-website` (cluster `clutter-website`) each show 1 running task with a healthy target; `curl -fsS https://app.example.com/healthz` answers 200 (from B); and `https://app.example.com` loads, where **Sign in with Google** opens Google's window. If a push fails, nothing is applied. A 502 or 503 means that service's task is not running: its log group (`/ecs/clutter-adapter-api` or `/ecs/clutter-website`) names the cause. If it says A's API address changed in this apply, set a new `site_image_tag` and run `npm run deploy -- prod --site-image`.
12. **(person) Link the test AWS account:** in the site, link an AWS account, run the CloudFormation link in the test account, and paste back the role ARN; then, on the Connections page, add an AWS connection from that linked account. Users' roles trust B through the template's `AdapterRunnerRoleName`, whose default `ClutterAdapterRunnerRole` is B API's task role, so no step is needed unless that name changes, in both places. Success: the connection shows verified. If it does not, the role's external id or trust does not match. Before the first deploy with a table, open DynamoDB's settings in the test account and check that attribute-based access control is on: the role changes only tables tagged `ManagedBy = Clutter`, and with it off DynamoDB ignores tags, so creating a table fails with access denied.
13. **(person) The demo OpenStack:** start the DevStack VM in the test account and add an OpenStack connection with its application credential. Switch the VM off after the demo.
14. **(person) Test:** run the 12-step test below (§ Testing A in the cloud) and B's demo (`system-b-adapter-layer.md` § What the user sees). Any failure is a bug in A, B or the setup; fix it and repeat from that step.

### How a cloud run works

B API starts one Fargate task per run in its own cluster, from the run task image (`backend/adapter/cmd/runtask`, Terraform 1.14.4). For each run, B assumes `ClutterAdapterRunFilesRole` with a session policy that allows only that run's objects and that deployment's state, for at most one hour. It uploads the run's files and environment to `runs/<run>/` in the run bucket, and puts those keys in a file ECS reads into the task, so neither the keys nor the user's credentials appear in the task's description. The task has no role, no database and no secret store. It runs `terraform init` with the state bucket's S3 backend (state at `state/<deployment>/terraform.tfstate`, locked by S3's lock file), then plan, apply or destroy, and uploads its log every few seconds; B copies the log into the run's log, removing credential values. When the task stops, B reads its exit code and deletes the run's objects; the bucket removes any left after a day. A run lasts at most 55 minutes (`ADAPTER_RUN_TIMEOUT`), under the one-hour limit of its keys.

## What must change before the cloud can exist

Nothing in A's files blocks the cloud now. Each of these was a seam or a decision in A's files (ADR-006, `implementation-plan.md` § Stage 1), and each comes from configuration:

| Value | Comes from |
| --- | --- |
| The platform state's backend | `prod.backend.hcl`, at `terraform init` |
| The website's addresses | build arguments of its image: A's `api_invoke_url` output, and for B the site's own origin (`https://` and `site_domain`), passed by `npm run deploy -- prod --site-image` |
| The site's origin for CORS | `https://` and `site_domain`, for A's Lambdas (`CORS_ALLOWED_ORIGIN`) and B (`ADAPTER_ALLOWED_ORIGINS`) |
| The Google client id | `google_client_id`, for the log-in Lambda's `GOOGLE_CLIENT_ID` and the website's build argument `NEXT_PUBLIC_GOOGLE_CLIENT_ID`, with no id written into the code |
| The contact address | `contact_email`, for the website's build argument `NEXT_PUBLIC_CONTACT_EMAIL`, with no address written into the code |
| A's API address | the platform Terraform's stage `prod`, redeployed when a file that defines the API changes; the `api_invoke_url` output |
| A's runner images | `terraform_deployer_image_tag` and `ansible_runner_image_tag`, version tags that the platform Terraform refuses as `latest` |
| The website's image | `site_image_tag`, a version tag that the platform Terraform refuses as `latest` |

B's part is done in code: the platform Terraform, both images, the run task, the cloud launcher (`ADAPTER_RUN_LAUNCHER=ecs`, which hands the whole run to one Fargate task with S3 state and a lock file) and run logs in S3 (`ADAPTER_LOG_STORE=s3`); the platform Terraform sets both. The website's hosting is done in code too: its Fargate service behind the same load balancer (`backend/infra/modules/website`), and its image built and pushed by the deploy command. Only a real deploy can confirm either (stage 7).

## Testing A in the cloud

Nothing here has been run yet. A is proven when every step passes, in this order. Use the test AWS account, never a real user's.

1. Sign in with Google, using the client's Google client.
2. Create an organization, a project and a diagram.
3. Link the test account through the real CloudFormation link, and paste back the role ARN.
4. Draw a Lambda connected to an S3 bucket, and save.
5. Press **Preview Terraform**: the files appear, with no warning.
6. Press **Deploy**: live status runs, and the run ends in success.
7. Check the AWS console of the test account: the Lambda, the bucket and the IAM policy exist.
8. Open the log files tab: the run's logs are there.
9. Upload Lambda code, deploy again, and invoke the function.
10. Run an Ansible playbook on an EC2 server from the diagram.
11. Sign in as a second user in another organization: they cannot see any of this.
12. Press **Destroy**: everything is gone from the test account.

Any failure is a bug in A or in the setup; fix it and repeat from that step. B's own end-to-end check is the demo in `system-b-adapter-layer.md` § What the user sees.

## Before real users

Before outside users arrive: add a separate test environment, backups for Postgres and Terraform state, monitoring and alerts, and the items under "Not planned yet" in `.claude/findings/NOW.md`.
