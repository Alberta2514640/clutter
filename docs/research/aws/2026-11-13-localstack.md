# LocalStack for local AWS apply tests

Read on 2026-11-13. Every fact below was fetched that day unless it says otherwise. "Official" means LocalStack or HashiCorp. "Community" means anyone else.

## Sources

- Licensing (official): <https://docs.localstack.cloud/aws/licensing/>
- Pricing and plan table (official): <https://www.localstack.cloud/pricing>
- Road ahead blog, posted 2025-12-18, updated 2026-03-17 (official): <https://blog.localstack.cloud/the-road-ahead-for-localstack/>
- Single image next steps (official): <https://blog.localstack.cloud/localstack-single-image-next-steps/>
- Pricing changes blog, posted 2026-02-27, updated 2026-07-24 (official): <https://blog.localstack.cloud/2026-upcoming-pricing-changes/>
- Terms of service (official): <https://localstack.cloud/legal/terms-of-service>
- Auth token docs (official): <https://docs.localstack.cloud/aws/getting-started/auth-token/>
- Install docs (official): <https://docs.localstack.cloud/aws/getting-started/installation/>
- Terraform docs (official): <https://docs.localstack.cloud/aws/integrations/infrastructure-as-code/terraform/>
- IAM enforcement (official): <https://docs.localstack.cloud/aws/capabilities/security-testing/iam-policy-enforcement/>
- Service pages (official): <https://docs.localstack.cloud/aws/services/ec2/> , /iam/ , /lambda/ , /s3/ , /dynamodb/ , /apigateway/ , /ssm/ , /sts/ , /logs/ (the Logs page is at `/logs/`; `/cloudwatch-logs/` is a 404)
- Arm64 notes (official): <https://docs.localstack.cloud/aws/capabilities/config/arm64-support/>
- Docker Hub tags (official): <https://hub.docker.com/v2/repositories/localstack/localstack/tags>
- GitHub releases, repo archived (official): <https://github.com/localstack/localstack/releases>
- AWS provider custom endpoints guide, tag v5.100.0 (official): <https://raw.githubusercontent.com/hashicorp/terraform-provider-aws/v5.100.0/website/docs/guides/custom-service-endpoints.html.markdown>
- AMI parameter issue 10764 (official repo, community report): <https://github.com/localstack/localstack/issues/10764>

## Lead must read

1. There is no free open-source edition any more. The repo was archived on 2026-03-23. The `latest` image needs an auth token.
2. The free plan (called Hobby, or "Free plan" in one blog) bans commercial use. Clutter's client is a for-profit company (Alberta 2514640 Inc). Using Hobby for this work is likely a terms breach. Unconfirmed: LocalStack has no published ruling on a student project done for a company. Ask them.
3. Hobby has no IAM policy enforcement and no persistence. EC2 is always the mock manager on Hobby. So Hobby cannot test our permissions boundary or the narrowed customer role.
4. The old no-token route is a pinned tag, `localstack/localstack:4.14.0`. It is frozen, with no fixes. It keeps working without a token. I did not read its coverage page, so service coverage in 4.14.0 is unconfirmed.
5. Docker Hub `latest` is 2026.10.7 (pushed 2026-11-11), multi-arch with arm64.

## Findings

### 1. Licence and access

1. Community and Pro merged into one image, `localstack/localstack`, from 2026-03-23. The unified image "will require authentication via an auth token". Official. <https://blog.localstack.cloud/localstack-single-image-next-steps/> and <https://blog.localstack.cloud/the-road-ahead-for-localstack/> (read 2026-11-13).
2. The GitHub repo "was archived by the owner on Mar 23, 2026". Latest tag there is v4.14.0 (26 Feb). Official. <https://github.com/localstack/localstack/releases>
3. Old tags run without a token, locally and in CI. Pinning means no updates and no security patches. The blog uses 4.12 only as an example. Official. <https://blog.localstack.cloud/the-road-ahead-for-localstack/>
4. Docker Hub shows tag `4.14.0` active, last updated 2026-02-26, with amd64 and arm64 images. Official. <https://hub.docker.com/v2/repositories/localstack/localstack/tags/4.14.0>
5. The licence file of the archived repo was not checked. Whether the 4.14.0 image carries Apache 2.0 terms is unconfirmed.
6. The token is mandatory on the current image. Without it the container exits. Developers use personal tokens. CI uses a separate CI token held in CI secrets. Offline fallback needs re-activation every 24 hours. Official. <https://docs.localstack.cloud/aws/getting-started/auth-token/>
7. Plans as of 2026-03-23: Hobby (non-commercial), Base, Ultimate, Enterprise. Student and OSS-sponsorship subscriptions exist. Official. <https://docs.localstack.cloud/aws/licensing/>
8. Hobby terms: "intended for personal, educational, and non-commercial development use only". Commercial use needs consent from LocalStack. Telemetry is enforced. Official. <https://localstack.cloud/legal/terms-of-service> and <https://www.localstack.cloud/pricing>
9. CI is allowed on Hobby with a token, under a fair-usage policy. Commercial or shared team pipelines need a paid plan. Official. <https://www.localstack.cloud/pricing>
10. OSS sponsorship needs a project not owned by a for-profit entity. Clutter's upstream is the client's repo, so it likely does not qualify. Official. <https://www.localstack.cloud/pricing> (via search snippet; not re-fetched).
11. Student plan needs a verified GitHub Education account. Official. <https://docs.localstack.cloud/aws/licensing/>
12. Price of Base: unconfirmed. One third-party page says $39 a month. I could not verify it.

### 2. Service coverage on Hobby

Plan table is official: <https://www.localstack.cloud/pricing> and <https://docs.localstack.cloud/aws/licensing/> . One fetch said the Yes/No came from icon names, but the second fetch of the licensing page agrees.

| Service | Hobby | Notes (official service pages) |
| --- | --- | --- |
| EC2 | Yes | Hobby always uses the mock manager (CRUD in memory). Docker-backed instances are Base and up. VPC, subnet, IGW, route table, route, ingress rule, volume and attach calls are listed as implemented. Only the `default` security group is supported by the Docker manager. Real AWS AMIs cannot be downloaded. Mock AMIs fail with `InvalidAMIID.NotFound` on Docker launch. Instance profile association calls implemented. |
| IAM | Yes | 151 of 180 operations. Roles, policies, instance profiles and permissions boundary calls implemented. The page says nothing on PassRole or AWS managed policy availability. Enforcement is not on Hobby. |
| Lambda | Yes | Runs real code in Docker containers. Needs the Docker socket mounted. Zip upload works. The role ARN format is checked but the role need not exist. Arm64 and x86_64 supported. |
| S3 | Yes | Buckets, versioning and public access block calls implemented. Virtual-host style is default. Signature validation is off by default. Authorisation needs IAM enforcement. |
| DynamoDB | Yes | LocalStack's own emulator. TTL needs `DYNAMODB_REMOVE_EXPIRED_ITEMS=1`. The worker runs every 60 minutes. |
| API Gateway REST v1 | Yes | 113 of 124 operations. Resources, methods, Lambda proxy, deployments, stages work. Resource policies are stored but not evaluated. HTTP, WebSocket and Management APIs are Base and up. The legacy implementation was removed in 2026.10.0. |
| SSM | Yes (Parameter Store only listed) | `GetParameter` implemented. Run Command (`AWS-RunShellScript`) implemented but needs the Docker EC2 manager, so not on Hobby. Public `/aws/service/ami-amazon-linux-latest/...` parameters: the page does not mention them. Issue 10764 (closed 2025-02 as stale, never fixed) shows the CDK-resolved AMI id did not match the AMI LocalStack tagged. Treat public AMI parameters as unconfirmed. |
| STS | Yes | `AssumeRole` implemented. The page does not mention ExternalId. Enforcement of ExternalId or trust conditions is unconfirmed. |
| CloudWatch Logs | Yes | 55 of 118 operations. Log groups, retention calls, Lambda logs work. |

### 3. Terraform AWS provider against LocalStack

1. LocalStack now recommends `lstk terraform` (alias `lstk tf`). It writes a temporary override file. It "supersedes the older `tflocal` wrapper", which is deprecated. Official. <https://docs.localstack.cloud/aws/integrations/infrastructure-as-code/terraform/> and <https://docs.localstack.cloud/aws/connecting/infrastructure-as-code/deprecated-wrapper-scripts/>
2. Manual config from LocalStack: `access_key = "test"`, `secret_key = "test"`, `skip_credentials_validation = true`, `skip_metadata_api_check = true`, and an `endpoints {}` block per service pointing at `<http://localhost:4566`.> Same page.
3. The AWS provider at v5.100.0 supports the global `AWS_ENDPOINT_URL` and per-service `AWS_ENDPOINT_URL_<SERVICE>`. The `endpoints` block wins over both. Custom endpoints are "best effort"; HashiCorp tests only real AWS. Official. The v5.100.0 doc link above.
4. For S3 set `AWS_ENDPOINT_URL_S3` as well, or virtual-host requests go to `<bucket>.localhost.localstack.cloud`. Use `s3_use_path_style = true` if that name does not resolve. Official. <https://docs.localstack.cloud/aws/services/s3/>
5. `skip_requesting_account_id`: not in the LocalStack page I read. Unconfirmed as a requirement. The default account id is `000000000000`.
6. Known gaps with provider v5.100: none found in official docs. Not checked in issue trackers. Unconfirmed.
7. Clutter point: generated code that hard-codes the `provider "aws"` block would need an override. An `AWS_ENDPOINT_URL` env var needs no code change unless the block sets its own `endpoints`.

### 4. IAM enforcement

1. Off by default. Enable with `ENFORCE_IAM=1`. Official. <https://docs.localstack.cloud/aws/capabilities/security-testing/iam-policy-enforcement/>
2. Listed for Base and Ultimate. Hobby is explicitly without it (pricing page).
3. It evaluates identity policies, resource policies, permissions boundaries and SCPs. Official, same page.
4. PassRole enforcement: unconfirmed. Not stated.
5. For Clutter: boundary and narrowed-role tests need Base or higher, which is commercial.

### 5. Running it on Apple Silicon

1. The image is multi-arch. `docker pull localstack/localstack:<tag>` picks arm64 on a Mac. Official. <https://docs.localstack.cloud/aws/capabilities/config/arm64-support/>
2. Ports: 4566 gateway, 4510-4559 external services, 443 HTTPS gateway. Mount `/var/run/docker.sock` for Lambda. `PERSISTENCE=1` plus a volume at `/var/lib/localstack` keeps state, but persistence is not on Hobby. Official. <https://docs.localstack.cloud/aws/getting-started/installation/> and the pricing page.
3. Health: `curl <http://localhost:4566/_localstack/health`.> Official, install page.
4. Lambda arch must match the function arch, or set `LAMBDA_IGNORE_ARCHITECTURE=1`. On a Mac the host cannot reach EC2 containers over the bridge network. Official arm64 and EC2 pages.

Compose sketch for the pinned no-token image (not run here):

```yaml
services:
  localstack:
    image: localstack/localstack:4.14.0
    ports:
      - "127.0.0.1:4566:4566"
    environment:
      - DEBUG=0
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
```

For the current image add `LOCALSTACK_AUTH_TOKEN=${LOCALSTACK_AUTH_TOKEN}` and use a tag such as `2026.10.7`.

## Contradictions and absences

- The first blog said free accounts have no CI credits. A later update says "no more limitations on CI credits for any plan". The pricing page says CI is included on Hobby with fair use. I follow the newer statements.
- The blogs say "Free plan". The pricing and licensing pages say "Hobby". I treat them as one plan.
- Absent: PassRole enforcement, AWS managed policy availability, ExternalId handling, public AMI parameters, `skip_requesting_account_id`, 4.14.0 per-service coverage, the licence of the archived repo, any ruling on student work for a company.
- Community claims only, not checked: free alternatives named Floci and LocalEmu, both new. Source: <https://dev.to/peytongreen_dev/localstack-killed-its-free-tier-heres-how-to-test-aws-in-python-for-free-in-2026-12me> and search results. Check them independently before any use.

## Confidence and gaps

Medium-high on licensing and plan facts (official, several pages agree). Medium on service behaviour: I read doc summaries, not behaviour on a real apply. Nothing was run.

## What this means for Clutter

- Do not use Hobby for client-linked work without written consent from LocalStack.
- The pinned `4.14.0` image is the practical no-token option. Pin the digest too. Verify its terms and coverage first.
- It can prove: generated Terraform parses, the provider wiring works, resource ordering and dependencies apply, and S3, DynamoDB, IAM objects, Lambda from zip, and REST API v1 create.
- It cannot prove: IAM enforcement (boundary, narrowed role, PassRole), real EC2 behaviour, public AMI lookups, SSM Run Command on Hobby, ExternalId on AssumeRole, or anything about the real AWS accounts.

## Questions for the vendor or client

1. LocalStack: may a university capstone project built for a for-profit client use the free plan?
2. Client: will they pay for a Base plan for a developer seat, or prefer a frozen community image?
3. Team: are IAM enforcement tests needed before the client accounts exist, or can a throwaway real AWS account cover them?

## Checked by the lead, 2026-11-13

- The `v4.14.0` tag's `LICENSE.txt` is the Apache License 2.0 (<https://raw.githubusercontent.com/localstack/localstack/v4.14.0/LICENSE.txt>), so the frozen image may be used for client and commercial work without a token. GitHub's API reports the archived repository's licence as "NOASSERTION"; the tag's file is what applies to that release.
- Docker Hub's `localstack/localstack:4.14.0` was last updated 2026-02-26 and has `amd64` and `arm64` images (<https://hub.docker.com/v2/repositories/localstack/localstack/tags/4.14.0>). Pin it by digest.
- Decision: Clutter uses only that frozen image, never the token image, as a wiring check for AWS output. IAM enforcement, real EC2 and the external id stay on a real AWS test account (stage 7).

## What we observed, 2026-11-13

Run on an Apple Silicon Mac with Docker 29.9.2, Terraform 1.15.6 and the AWS provider 5.100.0, against `localstack/localstack:4.14.0@sha256:3ebc37595918b8accb852f8048fef2aff047d465167edd655528065b07bc364a` (the multi-arch index digest on Docker Hub; arm64 `sha256:8f29a093...`, amd64 `sha256:75ba7a4b...`). How to run it and the tests: `backend/devserver/README.md` § AWS on LocalStack.

- The image starts with no auth token and reports `"edition": "community", "version": "4.14.0"`.
- The public AMI parameter `/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64` exists in `ca-central-1` and resolves to an AMI that `RunInstances` accepts. Nothing had to be created first. The AWS managed policies `AWSLambdaBasicExecutionRole` and `AmazonSSMManagedInstanceCore` exist too.
- Every resource type A's and B's AWS templates render applies, reads back and destroys: 30 types in B's test (VPC, subnet, internet gateway, route table and association, security group and its ingress and egress rules, instance, EBS volume and attachment, IAM role, inline policy, policy attachment and instance profile, S3 bucket with versioning and public access block, DynamoDB table, Lambda function and permission, and API Gateway REST API, resource, method, integration, method and integration responses, request validator, deployment and stage).
- A Lambda function from a zip reaches the active state without the function being invoked.
- API Gateway refuses to deploy a REST API that has no method (`BadRequestException: The REST API doesn't contain any methods`), as AWS does. A's API template with CORS off hits this.
- API Gateway refuses an integration response created before its integration (`NotFoundException: Invalid Response status code specified`). A's API template with CORS on hits this some of the time.
- CloudFormation deploys Clutter's role stack (`client_side_terraform_deployer_role.yaml`), outputs included, but it drops `Path: /clutter/` from the managed policy: the boundary is created at `/`.
- STS `AssumeRole` succeeds for a role that does not exist, in any account, with any external id or a wrong one. The session it returns is in the role's account.
- IAM `CreateRole` accepts a permissions boundary ARN that does not exist.
- EC2 is the mock: an instance with `associate_public_ip_address = false` comes back with a public IP address, and instances' security group lists read back differently from what was set, so a plan right after apply is not empty.
- Two Terraform processes that start the AWS provider from one plugin cache at the same moment sometimes fail with "Failed to read any lines from plugin's stdout". Run the tests one at a time.

Not observed: IAM enforcement (not in this image), any Lambda invocation, and SSM Run Command.
