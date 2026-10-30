# Patrii Cloud: what it exposes, and what that means for a Clutter adapter

Date: 2026-10-26. Sources are linked per claim. "Bundle" means Patrii's public console JavaScript (`console.patrii.cloud/assets/index-*.js`): public but undocumented, and its file name changes with each build, so treat bundle claims as a snapshot. The two claims that decide the plan (API status, product list) were re-checked against patrii.cloud/api and patrii.cloud/services on the same date.

## The finding that decides the plan

Patrii does not publish a customer API today. Its REST API is "currently in development" with "API Documentation Coming Soon", and its Terraform provider is labelled "Coming Soon" ([patrii.cloud/api](https://patrii.cloud/api)). No public OpenStack endpoint (Keystone, Nova, Neutron and so on) was found, and the console's build config points Keystone at a private address (`http://10.0.0.10:5000/v3`, bundle). So the standard `terraform-provider-openstack` and the `openstack.cloud` Ansible collection cannot target Patrii unless Patrii opens those endpoints. The first conversation with Patrii is about API access, not about code.

## Company

- Founded 2025-07-11 in Greater Montreal; Guillaume Harvey is co-founder, CEO and CTO; the founders previously co-founded KeepSec Technologies ([about](https://patrii.cloud/about)).
- Blog posts are signed "Patrii Quebec inc."; the legal page names no entity; contacts are legal@patrii.ca and privacy@patrii.ca ([blog](https://patrii.cloud/blog/why-i-started-patrii-cloud-6), [legal](https://patrii.cloud/legal)).
- In beta since 2026-06-18 ([services](https://patrii.cloud/services)). Changelog entries in the bundle go back to 2025-10-22, so the dates on Patrii's own pages do not fully agree.
- Compliance: PIPEDA and provincial privacy compliance are "Planned" ([legal](https://patrii.cloud/legal)); ISO 27001 certification targeted Q1 2027 and SOC 2 Type II Q2 2027 ([services](https://patrii.cloud/services)).
- No link found between Patrii and Alberta 2514640 Inc or Devin Atkin ([alberta2514640.ca](https://alberta2514640.ca/) describes a Calgary consultancy). This is absence of evidence only.

## Stack and products

- OpenStack Epoxy, Ceph with 3x replication, Open vSwitch with VXLAN, KVM; monitoring with Zabbix, Prometheus and Grafana ([services](https://patrii.cloud/services)). A custom dashboard replaces Horizon and FreeIPA handles identity (community: [AlternativeTo](https://alternativeto.net/software/patrii-cloud/about/)).
- Beta products: shared VPS, block storage, S3-compatible object storage, networking, floating IPs, load balancers (Basic, Standard, HA tiers with L7 rules and TLS, bundle), DNS, domain registration, TLS certificates, NFS shares, Kubernetes (Magnum with Cluster API, bundle and [github.com/PatriiCloud](https://github.com/PatriiCloud)), databases (PostgreSQL, MySQL, Valkey, FerretDB), PaaS (managed Coolify), Lynx ERP/CRM/GRC ([services](https://patrii.cloud/services)).
- Secrets use Barbican; the console has shared and dedicated VPN (bundle). VPN Interconnect is still a Q3 2026 roadmap item on the services page.
- Not offered: serverless functions, an API gateway, a DynamoDB-style managed NoSQL table service ([services](https://patrii.cloud/services)).
- Region: Montreal only (`montreal-1` in public docs, `COLOGIX-MTL-2` in the bundle). A 2026-10-05 changelog entry adds "Multi-Region Foundations" (bundle).

## Pricing and billing

- CAD per month, pay as you go, plus tax. VPS from $5.25 (1 vCPU, 1 GB, 20 GB) to $34.99 (4 vCPU, 4 GB, 80 GB); floating IP $2.15; snapshots about $0.07/GB; bandwidth unmetered ([pricing](https://patrii.cloud/pricing)).
- Beta trial: 7 days, $10 credit, no card. Object, block, load balancer, database and Kubernetes prices are not published.
- Prepaid credits, quota requests that are approved and paid, company verification (bundle).

## Auth and API

- `api.patrii.cloud` is live and answers 401 without auth. The API page previews Bearer API keys and paths such as `GET /v1/instances` ([api](https://patrii.cloud/api)).
- The console calls Patrii's own proxy API, not raw OpenStack: `/compute/servers`, `/networks/routers`, `/networks/load-balancers`, `/dns/zones`, `/kubernetes/clusters`, `/databases`, `/secrets`, `/projects/{id}/quotas` (bundle).
- Per-project API keys: a key "acts as you, in this project only, with at most your roles here" and "cannot create, list or revoke keys"; expiry of 30, 90, 180 or 365 days, or never (bundle).
- A changelog entry mentions "App credentials and an OAuth2/OIDC bridge for programmatic API access" (bundle), which suggests Keystone application credentials behind the scenes. Unconfirmed.
- Object storage is Ceph RadosGW with an S3 API, access keys, and setup snippets for mc, rclone, s3cmd, the AWS CLI and boto3 (bundle). The endpoint hostname was not found.
- SDKs `patrii-cloud-sdk` (pip) and `@patrii/cloud-client` (npm) appear on the docs page; whether they are published was not checked ([documentation](https://patrii.cloud/documentation)). `github.com/PatriiCloud/terraform` is empty.

## Mapping Clutter's AWS resources

| Clutter today | Nearest Patrii or OpenStack equivalent |
|---|---|
| EC2 | Nova server (Patrii VPS) with keypair, security group, floating IP; cloud-init supported. |
| S3 | Ceph RGW S3 API with Patrii access keys. Terraform's S3 backend accepts a custom endpoint, but HashiCorp supports non-AWS S3 only on a best-effort basis ([docs](https://developer.hashicorp.com/terraform/language/backend/s3)). |
| Lambda | None. Options: a container on Patrii Kubernetes, a Coolify app, or a VPS. |
| API Gateway | None. Nearest is the L7 load balancer with TLS, or an ingress or reverse proxy. |
| DynamoDB | None. Nearest are Valkey (key-value), FerretDB (Mongo API) or PostgreSQL. |
| IAM role plus STS AssumeRole | No keyless cross-account trust. Either a Keystone application credential with a role subset, access rules and expiry ([Keystone docs](https://docs.openstack.org/keystone/latest/user/application_credentials.html)), if Patrii exposes it, or a Patrii project API key, which carries the user's roles with no finer scope found. Both are long-lived secrets Clutter must store. |
| SSM Run Command | None. Options: cloud-init at boot; SSH from the runner over a floating IP, bastion or Patrii VPN with a keypair Clutter injects; or a pull model (ansible-pull or an agent that calls home). |

If Patrii opens the raw APIs: `terraform-provider-openstack` takes `cloud` (clouds.yaml), `auth_url`, `application_credential_id` and secret, `token` and `endpoint_overrides` ([provider docs](https://raw.githubusercontent.com/terraform-provider-openstack/terraform-provider-openstack/main/docs/index.md)); `openstack.cloud` 2.6.0 has modules for server, network, volume, object, load balancer, DNS zone and stack ([collection](https://docs.ansible.com/ansible/latest/collections/openstack/cloud/index.html)).

## Prior art for a provider layer

- **Terraform and OpenTofu:** the core is cloud-neutral; each provider brings its own credential, region and resource schema; plan, apply and state live once in the core.
- **Crossplane:** an XRD defines an abstract API and Compositions map it to provider resources ([docs](https://docs.crossplane.io/latest/composition/composite-resource-definitions/)).
- **Cluster API:** infrastructure providers implement a small contract and the core stays generic ([contract](https://cluster-api.sigs.k8s.io/developer/providers/contracts/overview)).
- **Apache Libcloud** and **CloudBridge:** one uniform API per area across clouds ([libcloud](https://libcloud.apache.org/), [cloudbridge](https://cloudbridge.cloudve.org/en/latest/)).
- **Pulumi** has an OpenStack provider ([registry](https://www.pulumi.com/registry/packages/openstack/)). **OAM/KubeVela** separates the developer's model from the operator's ([oam.dev](https://oam.dev/)).
- Lowest common denominator (Libcloud, CloudBridge) exposes only what every cloud has, which would remove Lambda, API Gateway and DynamoDB from every canvas. Capability-based (Terraform providers, Crossplane) lets each provider declare which node types it supports, and validation rejects or substitutes the rest before generation. For a generator that emits per-provider HCL anyway, capability-based fits. This last point is our synthesis, not a sourced claim.

## Hosting models for Clutter itself

- **Multi-tenant SaaS (today):** state and customer credentials live in Clutter's AWS account. A US-owned host is reachable under the US CLOUD Act even for data stored in Canada ([Balsillie report](https://balsilliepapers.ca/wp-content/uploads/2026/03/Special-Report-Canadian-Data.pdf)). PIPEDA allows processing abroad with accountability and comparable protection ([OPC](https://www.priv.gc.ca/en/privacy-topics/airports-and-borders/gl_dab_090127/)); Alberta PIPA s.13.1 requires notice and policies for service providers outside Canada ([Alberta](https://www.alberta.ca/organization-responsibilities-for-protecting-personal-information)); for Alberta public bodies FOIP was replaced by POPA on 2025-06-11 ([OIPC](https://oipc.ab.ca/legislation/popa/)).
- **Self-hosted:** everything in the customer's environment. Best residency, hardest to support and upgrade.
- **BYOC (bring your own cloud):** our control plane, runners inside the customer's cloud with outbound-only connections, the HCP Terraform agent model ([docs](https://developer.hashicorp.com/terraform/cloud-docs/agents)). State and credentials can stay with the customer. Fits Patrii well, because its Keystone and networks are private.
- **Hosted on Patrii:** Canadian-owned residency, but Patrii has no serverless, so Clutter's backend would move from Lambda and API Gateway to containers. Patrii is in beta with no certifications yet.

## Partner program

Reseller, Technology and Consulting partner types; benefits listed as revenue share, training, co-marketing, priority support and early access; apply through the contact page ([partners](https://patrii.cloud/partners)).

## From the second research run (same date)

A separate run wrote `reports/PatriCloud Clutter adapter layer.md` and notes in `research_notes/`. It adds these Patrii facts, which the lead has not re-probed:

- Terms updated 2026-09-07: sold by Patrii Québec inc. on technology owned and operated by Patrii Cloud inc., servers in Montreal. Billing is quota allocated in CAD, not hour-metered, with no proration on deletion ([terms](https://patrii.cloud/legal/terms)).
- A live probe of `api.patrii.cloud` returned `{"message":"Patrii Cloud API is running","version":"1.0.0"}`; `/docs`, `/redoc` and `/openapi.json` returned 404. The console bundle contains no `/v1/` string, so the `/v1/instances` samples on the API page do not match the running API.
- Console auth: `POST /auth/login` returns a token and session id; `POST /auth/refresh-token`; OIDC login; `X-CSRF-Token` on writes; `POST /auth/switch-project`. The tenant boundary is the project; there is no organization resource and no scope strings.
- Server actions are asynchronous, reported over `wss://api.patrii.cloud/ws`. No webhooks. Errors are an HTTP status plus a JSON `detail`. No idempotency key; a changelog line says a retried create once started several identical billable servers. Rate-limit headers appear on public GETs (`x-ratelimit-limit: 1000`).
- The SLA (dated 2025-07-24) targets 99.9% for API endpoints and excludes beta features, while the whole platform is labelled beta.
- The homepage claims "full API compatibility" and names Terraform, Ansible and Kubernetes, but never names Nova, Neutron or Keystone.
- Name collisions to avoid: Partiri (`partiri.cloud`) and Patr (`patr.cloud`) are different products.

## Not found or unconfirmed

Raw OpenStack endpoints, clouds.yaml or openrc downloads, customer-created application credentials, the S3 endpoint hostname, quota and rate-limit numbers, most prices, hourly billing, whether Heat, Designate and Swift are exposed, managed email, the legal entity's registration, whether the SDKs are published, and release dates for the API docs and Terraform provider.

## Questions for Patrii

1. Will you expose public Keystone v3, Nova, Neutron, Octavia, Designate and Glance endpoints with a downloadable clouds.yaml, and can customers create application credentials with access rules? If not, when does the `patrii/patrii` Terraform provider ship and which resources will it cover?
2. What is the scope model of API keys? Can a customer give Clutter a restricted, revocable, expiring credential, for example deploy-only for one project?
3. What is the S3 endpoint and its feature set (versioning, object lock, path-style)? Is it fit to hold Terraform state? What are the quotas and rate limits?
4. What is the recommended way to run commands on VMs from outside (there is no SSM)? Can our runners run inside a customer's project, or reach VMs over your VPN?
5. What does a Technology Partner get (sandbox credits, roadmap access)? Will you state the legal entity and a data-residency commitment in writing?
