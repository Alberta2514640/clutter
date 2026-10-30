# Patrii Cloud public API contract

Checked 26 October 2026. The name that resolves in public sources is **Patrii Cloud** (`patrii.cloud`, GitHub login `PatriiCloud`), a Montreal OpenStack cloud sold by Patrii Québec inc. on technology owned by Patrii Cloud inc. Spellings PatriCloud, Patri Cloud, Patricloud, patriicloud, patriot cloud, and `patri.cloud` do not identify a second product with its own API. Near-miss products are listed under the last question so they are not coded against by mistake.

Two surfaces disagree, and they must not be merged:

- **Marketing / “coming soon” pages** on `patrii.cloud` show illustrative curls. They label the REST API, OpenAPI spec, SDKs, webhooks, rate limits, and Terraform provider as not yet available.
- **The live host and the production console** (`https://api.patrii.cloud`, `https://console.patrii.cloud`) are a real JSON API. The console bundle is the only detailed contract found. It is a de facto client, not an official reference. Paths below that come from that bundle are quoted from `https://console.patrii.cloud/assets/index-Duo5u5i-.js` as fetched on 26 October 2026 (the hashed filename will change on the next build).

No public OpenAPI, AsyncAPI, Postman collection, SDK, or published Terraform provider was found. Do not invent request bodies beyond fields the console actually sends.

## What auth does the API use, and what is the tenant model?

### Takeaway

The live API accepts a bearer session token and a separate `X-API-Key` header. The production console logs in with email/password (optional MFA), stores a bearer token plus a session id, refreshes that session, and can redirect to an OIDC `authorize_url`. The console’s own API-key screen tells the user to send the key as `Authorization: Bearer`. No scopes, mTLS, or signed-request scheme were found. The tenant boundary in the client is a **project**, not an organization.

### Cited Findings

- Unauthenticated `GET https://api.patrii.cloud/v1/instances` on 26 October 2026 returned HTTP 401 `{"detail":"Token required"}` and no `WWW-Authenticate` header. The same body is returned for paths that are not in the console (`/healthz`, `/swagger.json`), so 401 does not prove a route exists. — live probe of `https://api.patrii.cloud`
- `Authorization: Bearer not-a-real-token` on `GET /v1/instances` returned HTTP 401 `{"detail":"Token has expired. Please login again."}`. — live probe of `https://api.patrii.cloud/v1/instances`
- `X-API-Key: not-a-real-token` and `X-Api-Key: not-a-real-token` returned HTTP 401 `{"detail":"Invalid API key"}`. A bare `Authorization` value with no scheme, and an `api-key` header, still returned `{"detail":"Token required"}`. — live probe of `https://api.patrii.cloud/v1/instances`
- The console creates an axios client with `baseURL` `https://api.patrii.cloud`, `Content-Type: application/json`, and `withCredentials: true`. On requests other than login/register/verify/invitation it sets `Authorization: Bearer ${authToken}` from `sessionStorage` and `X-Session-Id` from `localStorage` or `sessionStorage`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- `POST /auth/login` response fields the client reads: `mfa_required`, `token`, `user`, `remember_me`, `session_id`. The request body is the object passed in by the form; field names were not recovered from the bundle. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Session refresh is `POST /auth/refresh-token` with header `X-Session-Id`. The client reads `token` and `session_id` from the response. A 401 on other calls triggers that refresh once, then redirects to `/auth/login`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- SSO: `GET /auth/oidc/login` and `GET /auth/oidc/setup-mfa`, optional query `return_to`. The client expects `authorize_url` and navigates the browser there. Completion is another `POST /auth/refresh-token` with `X-Session-Id`. This is a user-login redirect, not a documented client-credentials grant. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Other auth routes called by the console: `POST /auth/logout`, `POST /auth/register`, `GET /auth/registration-status`, `GET /auth/email/registration-status`, `POST /auth/email/resend`, `POST /auth/email/verify`, `POST /auth/forgot-password`, `POST /auth/reset-password`, `GET /auth/verify-reset-token/${token}`, `POST /session/check-and-renew`, `POST /session/renew`, `GET /session/status`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Mutating requests (`POST`, `PUT`, `PATCH`, `DELETE`) also send `X-CSRF-Token`, taken from a `csrf_token` cookie, a `csrf-token` meta tag, or the `x-csrf-token` response header. Unauthenticated `GET /` set `Set-Cookie: csrf_token=…; Max-Age=3600; Path=/; SameSite=strict; Secure` and returned the same value in `x-csrf-token`. — live probe of `https://api.patrii.cloud/`; [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- API keys are a console feature at route `/app/api-keys` (old path `/app/projects/app-credentials` redirects there). UI copy: “No API keys in this project yet”, “Send it in the Authorization header: Bearer followed by the key.”, the secret is shown once, revoke is immediate, and a duplicate name returns “An API key with that name already exists”. The HTTP paths that list, create, and revoke those keys are not string literals in the bundle. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- `X-API-Key` is attached by the console only on announcement admin calls: `POST /announcements`, `PUT /announcements/${id}`, `DELETE /announcements/${id}`, `GET /announcements?active_only=`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Project switch: `POST /auth/switch-project` with body `{project_id, session_id}`. If `status` is `reauthentication_required` the client throws `message`. On success it stores `token`, `project`, and `user`. Related calls: `GET /auth/get-default-project`, `POST /auth/set-default-project`, `POST /auth/user-projects`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Project CRUD and membership: `GET /projects`, `POST /projects`, `GET/PUT/DELETE /projects/${id}`, `POST /projects/${id}/check-empty`, `GET /projects/${id}/members`, `POST /projects/${id}/members`, `DELETE /projects/${id}/members/${memberId}`, `PUT /projects/${id}/members/role`, `PUT /projects/${id}/members/roles`, `GET /projects/${id}/quotas`, `GET /projects/roles`, `GET /projects/search`, `POST /projects/${id}/support-access/grant`, `POST /projects/${id}/support-access/revoke`. No `X-Project-Id` header is set. `currentProjectId` is stored locally and used in a client-side cache key, and the user object may carry `project_id` and `project`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Marketing auth sample, labeled as not-yet-shipped documentation: `Authorization: Bearer YOUR_API_KEY` against `https://api.patrii.cloud/v1/instances`. — [API page](https://patrii.cloud/api), [documentation page](https://patrii.cloud/documentation)
- Terms (updated 7 September 2026) require an account and say the customer is responsible for credentials. They do not define API scopes, API keys, or organizations. Services are billed on quota allocated to the account, in CAD. Data is stored in Canadian data centers. Sellers: Patrii Québec inc., using technology of Patrii Cloud inc., servers in Montreal. Legal contact `legal@patrii.ca`, address 927 Dalhousie, St-Jerome, QC. — [Terms](https://patrii.cloud/legal/terms)
- Security page: RBAC and MFA, AES-256 at rest, TLS 1.3 in transit. Patrii’s own SOC 2 Type II and ISO 27001 are marked planned. Disclosure address `security@patrii.ca`. No mTLS and no customer-VPC deployment model. — [Security](https://patrii.cloud/security)
- Roadmap item “Single Sign-On (SSO)” is listed under Q3 2026, and “Identity as a Service” under Q1 2027, on a page that also says the roadmap can change. The console already calls `/auth/oidc/login`. — [Services](https://patrii.cloud/services); [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)

### Inferences

- A third-party tool should plan on `Authorization: Bearer` plus, for browser-like session flows, `X-Session-Id` and `X-CSRF-Token`. Static API keys are a product feature, but the only path that is both documented in UI copy and confirmed by a distinct server error is the bearer header. `X-API-Key` is a second scheme the server understands; the console uses it for announcement administration, not as the documented customer-key header.
- Project is the scope switch. An adapter that creates infrastructure must obtain a project id and call `POST /auth/switch-project` (or whatever server-side default the token already carries). There is no published scope string such as `compute:write`.

### Gaps

- Request JSON for `POST /auth/login` and `POST /auth/register` (email, password, MFA code names).
- HTTP paths for creating, listing, and revoking project API keys.
- Whether a bearer API key and a login JWT are distinguished server-side beyond the two 401 strings above. The fake bearer string produced “Token has expired”, which may be a generic message.
- Role names returned by `GET /projects/roles`, and whether billing profiles are a second tenant.
- OIDC issuer, client registration, and token endpoint. Only `authorize_url` is visible.
- No evidence of mTLS or AWS-style signed requests on `api.patrii.cloud`.

## What are the resource types and the main operations?

### Takeaway

The production client is REST against `https://api.patrii.cloud` with **no `/v1` prefix**. Resources are compute servers, images, flavors, keypairs, networks, load balancers, volumes, S3 buckets, DNS zones, Kubernetes clusters, databases, PaaS (Coolify) instances, projects, and billing. The marketing `/v1/instances` example is a different, explicitly unfinished sketch and should not be implemented.

### Cited Findings

- `GET https://api.patrii.cloud/` on 26 October 2026 returned HTTP 200 `{"message":"Patrii Cloud API is running","version":"1.0.0","docs":"/docs","redoc":"/redoc"}`. `GET /docs`, `GET /redoc`, and `GET /openapi.json` returned HTTP 404 `{"detail":"Not Found"}`. — live probe of `https://api.patrii.cloud`
- `GET /health` returned HTTP 200 `{"status":"healthy","internal_api":"configured"}`. — live probe of `https://api.patrii.cloud/health`
- Console base URL is hard-coded: `xu=()=>"https://api.patrii.cloud"`, and Vite env `VITE_API_URL:"https://api.patrii.cloud"`. The string `/v1/` does not occur in the bundle. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Build-time strings in that same bundle: `VITE_KEYSTONE_URL:"http://10.0.0.10:5000/v3"` and `VITE_OPENSTACK_REGION:"COLOGIX-MTL-2"`. `10.0.0.10` is a private address. It is not a public Keystone endpoint. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Marketing homepage says the platform “runs on OpenStack” and “Full API compatibility ensures your automation works seamlessly” with Terraform, Ansible, and Kubernetes. That page does not name Keystone, Nova, or any OpenStack path. — [Homepage](https://patrii.cloud/)
- Services page says orchestration is “OpenStack Epoxy”, storage “Ceph”, networking “Open vSwitch with VXLAN”, virtualization “KVM”. Object storage is described as “S3-compatible API”. DNS is described as having “Full API access”. No S3 hostname or DNS path is given on that page. — [Services](https://patrii.cloud/services)

**Compute (axios calls in the console):**

- `GET /compute/feature-flags`
- `GET /compute/servers?detailed=&force_refresh=`
- `GET /compute/servers/${id}`
- `POST /compute/servers` (body is the object the UI built; fields not inlined at the call)
- `PUT /compute/servers/${id}`
- `DELETE /compute/servers/${id}`
- `POST /compute/servers/${id}/action` with bodies the client sends: `{action:"start"}`, `{action:"stop"}`, `{action:"reboot"}`, `{action:"hard_reboot"}`, `{action:"pause"}`, `{action:"unpause"}`, `{action:"suspend"}`, `{action:"resume"}`, `{action:"resize", flavor_id}`, `{action:"rescue"}` plus optional `rescue_image_id`, `{action:"unrescue"}`, `{action:"rebuild", image_id}`
- `GET /compute/servers/${id}/console` and `GET /compute/servers/${id}/console?console_type=` (client default `novnc`)
- `POST /compute/servers/${id}/password` with `{private_key, passphrase}`
- `GET /compute/servers/${id}/diagnostics`
- `GET /compute/servers/${id}/metrics`
- `GET /compute/servers/${id}/logs`
- `POST /compute/servers/${id}/interfaces` and `DELETE /compute/servers/${id}/interfaces/${id}`
- `POST/DELETE /compute/servers/${id}/security-groups/${id}`
- `GET /compute/servers/${id}/resize/check` query `target_flavor_id`
- `POST /compute/servers/${id}/resize/request` with `{target_flavor_id, auto_confirm, use_credits}`
- `GET /compute/flavors` query `type`, `force_refresh`
- `GET /compute/allocated-flavors`
- `GET /compute/images` query `force_refresh`
- `GET /compute/networks`
- `GET /compute/security-groups`
- `GET /compute/regions`
- `GET /compute/keypairs?force_refresh=`
- `POST /compute/keypairs`
- `POST /compute/keypairs/import`
- `DELETE /compute/keypairs/${id}`
- `POST /compute/servers/${id}/snapshots` with `{name, include_volumes}`
- `GET /compute/servers/${id}/snapshots`
- `GET /compute/servers/${id}/snapshot-quota-estimate`
- `DELETE /compute/snapshots/${id}`
- `PATCH /compute/snapshots/${id}/protect` with `{protect}`
- `GET/POST/DELETE /compute/servers/${id}/snapshot-schedule`
- `POST /compute/servers/${id}/backups` with `{name}`
- `GET /compute/servers/${id}/backups`
- `DELETE /compute/backups/${id}`
- `GET /compute/metrics-test`

Source for every bullet in that list: [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js).

**PaaS (Coolify) instances:** base path `/paas/instances` with `GET/POST`, `GET/DELETE /paas/instances/${id}`, `POST /paas/instances/${id}/action?action=`, domains, floating IPs, interfaces, security groups, secrets, volumes, snapshots, resize, monitoring, network, provisioning-status, and `POST /paas/instances/${id}/security-upgrade`. Fleet job: `POST /paas/fleet/security-upgrade` with `{instance_ids, reboot_if_required: true}` and `GET /paas/fleet/upgrade-jobs/${id}`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)

**Databases:** client prefix `Fo="/databases"`, then `GET/POST ${Fo}`, `GET/DELETE ${Fo}/${id}`, backups, datastores, flavors, databases, and users under the instance. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)

**Load balancers:** client prefix `bn="/loadbalancers"` (not the `/networks/load-balancers` strings below). Operations include list/get/create/update/delete, `POST …/failover`, listeners, pools, members, healthmonitors, L7 policies and rules, stats, quotas, flavors. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)

**Kubernetes:** client `baseURL="/k8s"`. `GET/POST /k8s/clusters`, `GET/PUT/DELETE /k8s/clusters/${id}`, plus access, kubeconfig, nodegroups, nodes, scale preflight, security-groups, upgrade, users, cost, `POST …/force-delete`, and `GET /k8s/versions`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)

**Support:** `basePath="/support"` with plans, subscription, tickets, messages, and livechat config/token. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)

**Billing (representative, all called as `/billing/…`):** dashboard, flavors, invoices (download, view, apply-credits, mark-paid), profiles, company search/verify, credits topup/redeem/gift-card, quota requests, promotions, subscription cycles, support plans. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)

**Audit-route map** in the same bundle. A function matches the outgoing request URL to these patterns and assigns an event name. These are evidence of write routes the client expects to call. Several of them (`/networks`, `/storage`, `/dns`, `/secrets`, `/certificates`) did not also appear as literal `oe.get("/networks…")` calls, so the HTTP verb coverage below is what the map itself lists, not a full CRUD reading:

- `/compute/servers` POST `VPS_Create`, DELETE `VPS_Delete`; `/compute/servers/*/action` POST `VPS_Action`; `/compute/servers/*/snapshot` POST `VPS_Snapshot_Create`; `/compute/keypairs` POST/DELETE; `/compute/images` POST `Image_Upload`, DELETE `Image_Delete`
- `/networks` POST/DELETE; `/networks/subnets` POST/DELETE; `/networks/routers` POST/DELETE; `/networks/routers/*/add-external-gateway` POST; `/networks/routers/*/remove-external-gateway` DELETE; `/networks/floating-ips` POST/DELETE; `/networks/security-groups` POST/DELETE; `/networks/security-groups/*/rules` POST/DELETE; `/networks/load-balancers` POST/DELETE; listeners, pools, members, health-monitors, failover under `/networks/load-balancers/*`
- `/storage/volumes` POST/DELETE; `/storage/volumes/*/snapshots` POST; `/storage/s3` POST/DELETE; `/storage/s3/*/keys` POST/DELETE; `/storage/s3/*/upload` POST
- `/dns/zones` POST/DELETE; `/dns/zones/*/records` POST/DELETE
- `/kubernetes/clusters` POST/DELETE
- `/databases` POST/DELETE
- `/secrets` POST/DELETE; `/certificates` POST/DELETE
- `/billing/quota-request` POST; `/projects` POST/DELETE

— [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)

**Marketing sample, not the console contract.** The API page says “API Documentation Coming Soon” and “currently in development”, then lists:

- `GET /v1/instances`, `POST /v1/instances`, `PUT /v1/instances/:id`, `DELETE /v1/instances/:id`
- Example body: `name`, `type` (`vps.medium`), `region` (`montreal-1`), `image` (`ubuntu-22.04`), `ssh_keys`
- Example response: `id` `inst_1234567890`, `status` `creating`, `ip_address`, `created_at` `2025-07-21T10:00:00Z`

The services page, under the heading “Coming Soon: RESTful API”, uses a different body (`flavor` `shared-1cpu-1gb`, no `region`) and a different response (`id` `i-1234567890`, `status` `building`, `ip`). The documentation page’s quick start uses `type` `vps.small`. — [API page](https://patrii.cloud/api), [Services](https://patrii.cloud/services), [Documentation](https://patrii.cloud/documentation)

- Product nouns on the services page, without paths: shared VPS, block storage, object storage, networking, floating IPs, snapshots, custom images, databases (PostgreSQL, MySQL, Valkey, FerretDB), load balancers, DNS, domain registration, TLS certificates, PaaS (Coolify), NFS, Kubernetes, Lynx ERP. — [Services](https://patrii.cloud/services)
- No GraphQL or gRPC strings were found in the console bundle or on the API page. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js); [API page](https://patrii.cloud/api)

### Inferences

- Code against `/compute/servers` and the other unversioned paths the console calls. Treat `GET/POST /v1/instances` as a placeholder that happens to sit behind the global auth middleware.
- The two load-balancer prefixes (`/loadbalancers` in axios vs `/networks/load-balancers` in the audit map) mean one of those catalogs is stale or they are different resources. An adapter should not assume they are aliases until a response is observed.
- `COLOGIX-MTL-2` is the region string baked into the console. `montreal-1` appears only in the coming-soon samples.

### Gaps

- JSON schemas for create-server, create-network, create-volume, and DNS records. The console passes through UI objects that were not inlined as literals.
- Whether `/networks/*`, `/storage/*`, `/dns/*`, `/secrets`, and `/certificates` are live routes or only audit patterns.
- Public S3 endpoint hostname. The UI has strings for “S3 Public Endpoint URL” but the URL is fetched, not hard-coded.
- OpenStack service catalog URLs (Nova, Neutron, Cinder, Glance, Designate, Magnum). Only a private Keystone URL is embedded.
- Query parameter lists other than `detailed`, `force_refresh`, `type`, `console_type`, `target_flavor_id`, and `active_only`.

## Are there webhooks, event streams, or async jobs?

### Takeaway

No webhook subscription API or payload was found. The console opens a WebSocket on the API host and parses JSON messages with a `type` field. Long-running work is represented as server actions, PaaS upgrade jobs, and provisioning status, not as a public jobs resource.

### Cited Findings

- The API page lists “Webhook support for events” as something to expect while documentation is “coming soon”. It does not give a registration URL, signature header, or payload. The word `webhook` does not appear in the console bundle. — [API page](https://patrii.cloud/api); [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- WebSocket URL builder: `wss://api.patrii.cloud` plus the path passed in. The client connects to `` `/ws?token=${encodeURIComponent(token)}` `` using the session token. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Message handling: `JSON.parse` the frame, read `type`. `type === "ping"` is answered with `{type:"pong"}`. `type === "connected"` is ignored. Every other message is dispatched to handlers. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Types the client branches on, and the fields it reads:
  - `server_update`, `server_creating`: `data.server_id`, `data.status` (`ACTIVE`, `ERROR`, `FAILED`), `data.task_state`, `data.progress`
  - `lb_update`, `lb_ready`, `lb_failed`: `data.lb_id`, `data.status`, `data.progress`
  - `cluster_update`, `cluster_ready`, `cluster_failed`, `cluster_deleted`: handler only refetches clusters
  - `announcement_created`, `announcement_updated`, `announcement_deleted`: `data.id`, and for delete `data` is matched on `is_dismissed`
- — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Async-looking HTTP calls, not a generic jobs API: `POST /compute/servers/${id}/action`, `POST /compute/servers/${id}/resize/request`, `GET /paas/instances/${id}/provisioning-status`, `POST /paas/fleet/security-upgrade`, `GET /paas/fleet/upgrade-jobs/${id}`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- SLA text says Patrii provides “API for programmatic status checks” and names `status.patrii.ca`. On 26 October 2026 `status.patrii.ca` did not resolve. The public status page is `https://patrii.cloud/status`, which lists an “API” component as operational but does not publish a status API path. — [SLA](https://patrii.cloud/legal/sla); [Status](https://patrii.cloud/status); DNS lookup of `status.patrii.ca`

### Inferences

- There is no documented way for Patrii to call out to Clutter. The event channel that exists is a console WebSocket authenticated with the user token. Treating it as a stable third-party stream would be ahead of any published contract.
- Provisioning is asynchronous (`task_state`, `provisioning-status`, upgrade job id). An adapter would poll those resources or listen on `/ws`.

### Gaps

- Full WebSocket message schema, auth failure behavior, and whether API keys (as opposed to session tokens) can open `/ws`.
- Webhook signing, retries, and event catalog. None published.
- Schema of `GET /paas/fleet/upgrade-jobs/${id}`.
- The status-check API named in the SLA. No path was published, and the hostname in the SLA did not resolve.

## Is there an official OpenAPI spec, SDK, or Terraform provider?

### Takeaway

No. The marketing pages promise an OpenAPI 3.0 file, six language SDKs, and a Terraform provider. The live OpenAPI URLs 404, the GitHub provider repo is empty, and the Terraform Registry has no `patrii/patrii` provider.

### Cited Findings

- API page, under “API Documentation Coming Soon”: “Complete REST API for all services”, “OpenAPI 3.0 specification”, client libraries, webhooks, and rate limiting are listed as future work. SDKs named as not yet available: Python, Node.js/TypeScript, Go, Ruby, PHP, Java. Terraform sample is commented “Coming Soon” with `source = "patrii/patrii"`, `version = "~> 1.0"`, and `resource "patrii_instance"` arguments `name`, `type`, `region`. — [API page](https://patrii.cloud/api)
- French API page text returned in search says “Infrastructure as Code support coming in Q1 2026” for that provider. The English page fetched the same day says only “Infrastructure as Code support is on the way”. Q1 2026 is before 26 October 2026. — [French API page](https://patrii.cloud/fr/api); [English API page](https://patrii.cloud/api)
- `GET https://api.patrii.cloud/openapi.json` returned HTTP 404 `{"detail":"Not Found"}`. Root JSON still points at `/docs` and `/redoc`, which also 404. — live probe of `https://api.patrii.cloud`
- `GET https://registry.terraform.io/v1/providers/patrii/patrii` returned HTTP 404 `{"errors":["provider not found"]}`. — [Terraform Registry API](https://registry.terraform.io/v1/providers/patrii/patrii)
- GitHub user `PatriiCloud` (joined 2025-07-25, location Greater Montreal, blog `https://patrii.cloud`) has two public repos. `PatriiCloud/terraform` was created 2025-12-14, `size` 0, description “Patrii Cloud Terraform Provider. A Canadian Sovereign Cloud Platform”. Contents API: `{"message":"This repository is empty."}`. — [PatriiCloud user repos](https://api.github.com/users/PatriiCloud/repos?per_page=100); [terraform repo](https://github.com/PatriiCloud/terraform)
- `PatriiCloud/magnum-cluster-api` is a fork of `vexxhost/magnum-cluster-api` (“Cluster API driver for OpenStack Magnum”, Apache-2.0). It is not a Patrii REST SDK. — [repo metadata](https://api.github.com/repos/PatriiCloud/magnum-cluster-api)
- Community page: “GitHub — Coming soon”. — [Community](https://patrii.cloud/community)
- Documentation page: “Documentation Coming Soon”, and the quick-start curl is the `/v1/instances` sample. — [Documentation](https://patrii.cloud/documentation)
- A dltHub page “Load Patrii Cloud data in Python” (updated 19 May 2026) shows a generated `dlt` source with `base_url` `https://api.patrii.cloud/v1/` and resources `servers` and `instances`, plus a sample pipeline log. It tells the reader to have an AI toolkit invent the pipeline. It is not an official SDK and its resource names do not match the console. — [dltHub](https://dlthub.com/context/source/patrii-cloud)
- No AsyncAPI file, Postman collection, or changelog of API versions was found on `patrii.cloud` or in the `PatriiCloud` GitHub repos. The console contains an i18n `changelog` namespace, which is product UI copy, not an API changelog.

### Inferences

- There is nothing to vendor-pin. A Clutter adapter would be a handwritten client of the console’s routes, and it would break when those routes change, because there is no spec version to track.
- The empty `patrii/patrii` provider and the `patrii_instance` example are not an integration target.

### Gaps

- Private or authenticated OpenAPI. `/v1/openapi.json` returned 401 `{"detail":"Token required"}`, which is the default for non-public paths, so it is not evidence that a spec exists behind auth.
- Ansible collections, Pulumi providers, and official CLI. None found under the PatriiCloud GitHub user.

## Is there an adapter, connector, or plugin interface?

### Takeaway

Patrii does not publish an inbound plugin interface that another product implements in order to receive diagrams, plans, or applies. The closest artifact is Patrii’s own Lynx DCIM module, which claims to pull inventory from `https://api.patrii.cloud`. That module page required sign-in when fetched, and it does not describe an interface Clutter would implement.

### Cited Findings

- Search indexes of `https://lynx.patrii.ca/modules/lynx_dcim_ipam_cloud_patrii` and `https://dev.lynx.patrii.ca/modules/lynx_dcim_ipam_cloud_patrii` describe module `lynx_dcim_ipam_cloud_patrii` v19.0.1.0.0: “Implements the generic cloud-connector framework against the Patrii Cloud public API” for servers, networks, subnets, security groups, floating IPs, volumes, keypairs, flavours, and images. “Inventory pulls from https://api.patrii.cloud”. It says any tenant with Patrii credentials can sync “their cloud account”. Fetching `https://dev.lynx.patrii.ca/modules/lynx_dcim_ipam_cloud_patrii` on 26 October 2026 returned a Patrii sign-in page, so the body was not re-read from the origin. — indexed text of [lynx.patrii.ca module](https://lynx.patrii.ca/modules/lynx_dcim_ipam_cloud_patrii) and [dev.lynx.patrii.ca module](https://dev.lynx.patrii.ca/modules/lynx_dcim_ipam_cloud_patrii)
- The generic module index text for `lynx_dcim_ipam_cloud` v19.0.1.0.0 says it is “vendor-neutral”, with `lynx.cloud.provider` and `lynx.cloud.account`, mirror models (server, network, subnet, volume, snapshot, flavor, image, security group and rule, floating IP, keypair), “Read-only in v1”, an action queue `lynx.cloud.action.job` “for future write-through”, and a default sync interval of 30 minutes. The same fetch host required sign-in, so this is indexed text, not a page body retrieved in full. — indexed text of [dev.lynx.patrii.ca generic connector](https://dev.lynx.patrii.ca/modules/lynx_dcim_ipam_cloud)
- Services page lists “Lynx ERP/CRM/GRC” as “Our business platform, hosted in Canada” at `lynx.patrii.ca`. — [Services](https://patrii.cloud/services)
- Partner page `https://patrii.cloud/partners` is a reseller/MSP marketing page. The fetch did not contain an API, plugin manifest, or inbound callback contract. — [Partners](https://patrii.cloud/partners)
- No “plugin”, “adapter”, or inbound webhook receiver for external IaC tools appears on the API page or in the console’s route list. — [API page](https://patrii.cloud/api); [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)

### Inferences

- Direction that actually exists: a connector **calls Patrii**. Lynx’s overlay is a pull of cloud inventory into DCIM. It is not a hook Patrii uses to invoke an external diagram or Terraform service.
- If the indexed Lynx text is accurate, a read-only inventory adapter would need to mirror those resource types and poll on an interval (they cite 30 minutes). Write-through is described as future. That is a DCIM contract, not a Clutter run contract.
- A central Clutter adapter would be a new outbound client. Patrii does not currently define the methods that client must implement.

### Gaps

- The Lynx connector’s real HTTP paths, auth header, and pagination. The module pages did not render without a Patrii login, and they do not quote paths in the indexed summary.
- Whether `lynx.cloud.action.job` has a public schema. Only the name is in the index text.
- Any partner program API. Not on the partners page that was fetched.

## How do they model cloud accounts, IaC, diagrams, runs, or organizations?

### Takeaway

They model a user account, projects, billing profiles, and OpenStack-style infrastructure (servers, networks, volumes, clusters). They do not model diagrams, Terraform runs, plans, or applies. “Organization” does not appear as an API resource. Hosting is Patrii’s Montreal cloud, not a customer VPC.

### Cited Findings

- Terms: an account is required; services include “API access and integration services”; billing is quota allocated to the account, not hour-metered; deletion mid-term does not prorate. — [Terms](https://patrii.cloud/legal/terms)
- Console project and billing resources are listed in the auth and operations sections above (`/projects`, `/billing/profiles`, `/billing/company/*`, `/auth/switch-project`). — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Homepage: “Your data stays in Canada”, stack named as OpenStack, Ceph, FreeIPA, Montreal data centers, beta “since June 18, 2026”. Enterprise line mentions “Private cloud options available” without an API for bringing a customer VPC. — [Homepage](https://patrii.cloud/)
- SLA (last updated 24 July 2025) targets 99.9% for “API Endpoints” and excludes “features or services labeled as alpha, beta, or preview”. The site banner says the platform has been in beta since 18 June 2026. Maintenance window Sundays 02:00–06:00 EST. — [SLA](https://patrii.cloud/legal/sla); [Homepage](https://patrii.cloud/)
- No console route or API path containing `diagram`, `terraform`, `ansible`, `plan`, or `apply` was found. The only Terraform artifact is the empty repo and the coming-soon HCL sample. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js); [terraform repo](https://github.com/PatriiCloud/terraform)
- Region string embedded in the client is `COLOGIX-MTL-2`. Marketing samples say `montreal-1`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js); [API page](https://patrii.cloud/api)

### Inferences

- Patrii is a cloud for Clutter to target, in the same role as an AWS account, not a peer that already speaks diagram-to-Terraform. The adapter’s Patrii side is “project + bearer token + `/compute/servers` and related resources”.
- “Cloud account” in the Lynx index means a credentialed tenant whose inventory is synced. In the console, that tenant is a project under a user.

### Gaps

- Whether one user can belong to multiple billing companies that are not projects.
- What “private cloud options” means operationally. No API.
- No diagram, run, or workspace resource exists to map onto Clutter’s run model.

## What are versioning, idempotency, the error model, and pagination?

### Takeaway

The live root reports `version` `1.0.0`, but routes are unversioned. Errors observed and parsed by the console are HTTP status plus a JSON `detail` string. No idempotency key is sent. No cursor or page query was found on list calls. Rate-limit headers are present on responses even though the docs say rate limiting is still “coming”.

### Cited Findings

- `GET /` body includes `"version":"1.0.0"`. Console paths have no version segment. Marketing samples use `/v1/` while saying the API is in development. — live probe of `https://api.patrii.cloud/`; [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js); [API page](https://patrii.cloud/api)
- Observed error bodies, all `application/json`: `{"detail":"Token required"}`, `{"detail":"Token has expired. Please login again."}`, `{"detail":"Invalid API key"}`, `{"detail":"Not Found"}`. — live probe of `https://api.patrii.cloud`
- The console logs `response.data.detail` or `Error.message`, and surfaces `error.response.data.detail` for cluster failures. 401 starts a single refresh via `POST /auth/refresh-token` unless the URL is already that refresh or the user is on `/auth/`. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- The string `idempoten` does not occur in the console bundle. No `Idempotency-Key` header is set. A French changelog string in the bundle says a slow OpenStack response could retry a create and start the resource twice, so one click sometimes produced several identical billable servers. That is UI copy about a bug, not an idempotency contract. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- List calls use `force_refresh` and `detailed`. No `page`, `cursor`, `offset`, or `limit` query was found on those API calls. `page_size` hits in the bundle are Ant Design pagination widgets. The client deduplicates in-flight GETs (window constant `3e4` milliseconds) and keeps a short-lived GET cache; that is client behavior, not a server pagination protocol. — [console bundle](https://console.patrii.cloud/assets/index-Duo5u5i-.js)
- Response headers on `GET /` (26 October 2026): `x-ratelimit-limit: 1000`, `x-ratelimit-remaining: 998`, `x-ratelimit-reset: 1793043559`. The same response’s `x-request-id` was `1793043499.4900405`, 60 seconds before that reset value. `GET /health` returned `x-ratelimit-limit: 10000` and `x-ratelimit-remaining: 9998` with the same reset timestamp. No `Retry-After` was present. The API page still lists “Comprehensive rate limiting” as future work, so these headers are observed behavior, not a documented policy. — live probe of `https://api.patrii.cloud/` and `https://api.patrii.cloud/health`; [API page](https://patrii.cloud/api)
- Other response headers on `GET /`: `Server: nginx/1.29.3`, `strict-transport-security: max-age=63072000; includeSubDomains; preload`, `x-request-id`, `x-frame-options: SAMEORIGIN`. — live probe of `https://api.patrii.cloud/`

### Inferences

- Do not put `/v1` on console routes, and do not assume the marketing `/v1` resources exist, just because the root version string is `1.0.0`.
- Creates are not advertised as idempotent. The changelog copy says duplicate creates have happened. An adapter should generate its own idempotency, because the server does not document a key.
- When limited, the client-visible signal is `x-ratelimit-*`, and the limit differs by route (1000 vs 10000 on the two public GETs). The window looked like 60 seconds on that one sample.

### Gaps

- Whether `detail` is always a string. FastAPI also uses a list of validation errors. No 422 body was captured.
- Error code field. None was present in the four bodies above.
- Published rate-limit numbers, per-token vs per-IP, and which routes use 1000 vs 10000.
- Pagination once a project has more resources than one response. Not described.

## What was checked, including names that are not this product?

### Takeaway

A public JSON API is running at `https://api.patrii.cloud`. A written developer contract is not. The pages that look like docs say the contract is coming soon, and they disagree with the console. Several similarly named products are different companies.

### Cited Findings

- Checked and relevant: `https://patrii.cloud/`, `/api`, `/fr/api` (via search extract), `/documentation`, `/services`, `/security`, `/status`, `/legal/terms`, `/legal/sla`, `/community`, `/partners`, `/support` (search snippet only). Live host paths `/`, `/health`, `/docs`, `/redoc`, `/openapi.json`, `/v1/instances`, `/v1/openapi.json`. Console `https://console.patrii.cloud/` and its JS bundle. GitHub `PatriiCloud/terraform` and `PatriiCloud/magnum-cluster-api`. Terraform Registry provider `patrii/patrii`. Lynx module URLs above (login wall on direct fetch). — URLs cited in earlier sections
- `docs.patrii.cloud` does not resolve. `patri.cloud` and `patricloud.com` do not resolve. `app.`, `dashboard.`, `portal.`, `cloud.`, `my.`, `auth.`, `id.`, `s3.`, and `identity.patrii.cloud` do not resolve. `console.patrii.cloud` does. — DNS lookups on 26 October 2026
- **Partiri** (`partiri.cloud`, GitHub `Partiri-Cloud/harbor-mcp`) is a different PaaS. Its docs describe a CLI (`partiri auth login`, API key file `~/.config/partiri/key`), base URL env `PARTIRI_API_URL` default `https://api.partiri.cloud`, and an MCP server at `https://mcp.partiri.cloud/mcp` using OAuth 2.1 plus PKCE, with legacy `x-api-key`. That is not Patrii Cloud and not the contract above. — [Partiri CLI](https://partiri.cloud/documentation/cli), [Partiri account](https://partiri.cloud/documentation/account), [Partiri MCP](https://partiri.cloud/documentation/mcp)
- **pCloud** (`docs.pcloud.com`) is a file-storage API (OAuth 2.0, `api.pcloud.com` / `eapi.pcloud.com`). Unrelated. — [pCloud docs](https://docs.pcloud.com/)
- **Patrimóvel Cloud** (`br.com.lopes.patricloud`, `patrimovelcloud.com.br`) is a Brazilian real-estate CRM. No cloud-infrastructure API. — [Google Play listing](https://play.google.com/store/apps/details?hl=pt_BR&id=br.com.lopes.patricloud)
- Docker Hub `patrcloud/api` has no overview text. Nothing tied it to `patrii.cloud`. It was not used as a contract source. — [Docker Hub](https://hub.docker.com/r/patrcloud/api)

### Inferences

- For a Clutter adapter, the implementable surface today is the console’s REST client: bearer auth, project switch, and the unversioned resource paths quoted above. Everything on `/api` that says “coming soon” is marketing, including `/v1/instances`, the SDK list, webhooks, and `patrii_instance`.
- Partiri’s MCP/OAuth surface must not be copied onto Patrii. The names are close and the APIs are not.

### Gaps

- Authenticated OpenAPI, if one exists behind a real token. Not tested with credentials.
- API-key management routes.
- Whether Patrii will keep the console paths stable once the promised OpenAPI 3.0 document ships. No deprecation policy was found.
