# OpenStack connector facts: Keystone check, provider 3.4 arguments, form lists, Lima (2026-11-13)

Facts for `general-openstack` (stage 4). The DevStack test and block mapping are settled in `2026-11-13-devstack-and-block-mapping.md` and are not repeated. Every source below was read on 2026-11-13. "Official" means the OpenStack, Terraform provider, gophercloud or Lima project's own docs or source. Nothing here is community.

## Sources

- Keystone Identity API v3 reference (official): <https://docs.openstack.org/api-ref/identity/v3/index.html>
- Keystone application credentials guide (official): <https://docs.openstack.org/keystone/latest/user/application_credentials.html>
- Keystone source, `auth/plugins/application_credential.py`, `application_credential/backends/sql.py`, `auth/plugins/core.py` (official, master): <https://github.com/openstack/keystone>
- Gophercloud v2.16.0 source and docs (official): <https://pkg.go.dev/github.com/gophercloud/gophercloud/v2> and <https://github.com/gophercloud/gophercloud/tree/v2.16.0>
- Terraform provider v3.4.0 docs, read from the tagged source (official): <https://github.com/terraform-provider-openstack/terraform-provider-openstack/tree/v3.4.0/docs>
- Provider 3.0 upgrade guide (official): <https://github.com/terraform-provider-openstack/terraform-provider-openstack/blob/v3.4.0/docs/guides/upgrade-guide-version-3.md>
- Nova API reference (official): <https://docs.openstack.org/api-ref/compute/>
- Glance API reference (official): <https://docs.openstack.org/api-ref/image/v2/index.html>
- Neutron API reference (official): <https://docs.openstack.org/api-ref/network/v2/index.html>
- DevStack configuration doc and source (official): <https://docs.openstack.org/devstack/latest/configuration.html>, <https://github.com/openstack/devstack>
- Lima network, port forwarding and default template (official): <https://lima-vm.io/docs/config/network/>, <https://lima-vm.io/docs/config/port/>, <https://github.com/lima-vm/lima/blob/master/templates/default.yaml>

## Lead must read

- The 3.x provider has no `openstack_compute_floatingip_associate_v2`. Use `openstack_networking_floatingip_v2` plus `openstack_networking_floatingip_associate_v2` with the VM's Neutron port from a `openstack_networking_port_v2` data source. The block mapping must change if it names the old resource.
- Application credential auth sends no scope. The credential carries its own project. A user cannot choose a project with it.
- DevStack's catalog URLs use `SERVICE_HOST`, which defaults to `HOST_IP`. If `HOST_IP` is an address the Mac cannot reach, every catalog URL fails even when Keystone was reached. Set `HOST_IP` to the VM's vzNAT address before `stack.sh`, or rewrite catalog hosts in B.
- Not confirmed: the exact HTTP status for an unknown application credential id (see finding 1.6).

## 1. Credential verification without Terraform

1.1 **Endpoint.** `POST {auth_url}/auth/tokens`, where `auth_url` ends in `/v3` (DevStack: `http://HOST_IP/identity/v3`). Official, Keystone api-ref.
1.2 **Success is `201 Created`.** The token is in the `X-Subject-Token` response header, not the body. The body holds a `token` object with `catalog`, `expires_at`, `issued_at`, `methods`, `project`, `roles`, `user`. Official, Keystone api-ref.
1.3 **Application credential body (by id).** Official, Keystone api-ref.

```json
{"auth": {"identity": {"methods": ["application_credential"],
  "application_credential": {"id": "<id>", "secret": "<secret>"}}}}
```

By name, add `"name"` instead of `"id"` and a `"user": {"id": "<user id>"}` object. Do not send a `scope`: "You are not allowed to request a scope, as the scope is retrieved from the application credential." The response `token.application_credential` has `id`, `name` and `restricted`.

1.4 **Password body.** Official, Keystone api-ref.

```json
{"auth": {"identity": {"methods": ["password"],
  "password": {"user": {"name": "admin", "domain": {"name": "Default"}, "password": "<pw>"}}},
  "scope": {"project": {"name": "demo", "domain": {"name": "Default"}}}}}
```

Identify the user by `id` or by `name` plus domain. Scope may be a project (by `id`, or by `name` plus domain), a domain or the system. Without a scope the token is unscoped and the catalog is empty (the api-ref does not state this; see gaps). A `clouds.yaml` entry maps to this: `username` or `user_id`, `password`, `user_domain_name`, `project_name` or `project_id`, `project_domain_name`.
1.5 **Catalog shape.** `token.catalog` is an array. Each entry has `type`, `name`, `id` and `endpoints`. Each endpoint has `id`, `interface` (`public`, `internal`, `admin`), `region`, `region_id` and `url`. The region list is the distinct `region_id` values across endpoints. Official, Keystone api-ref. `GET /v3/auth/catalog` with `X-Auth-Token` returns the same catalog later (since v3.3). Adding `?nocatalog` to the auth request leaves the catalog out (Keystone source, `api/auth.py`).
1.6 **Errors.** The api-ref lists `401` (must authenticate), `403` (policy) and `404` (not found) for the call. Keystone source for application credentials: a wrong secret and an expired credential both raise `AssertionError("Invalid application credential ID or secret")`, which the plugin turns into `exception.Unauthorized`, so both are `401`. `_check_expired` compares now with `expires_at`. Official source, not the api-ref. Not confirmed: the status for an unknown id. The plugin first calls `get_application_credential(id)`, which raises `ApplicationCredentialNotFound` before the secret check. I did not read how that maps to HTTP, so treat `401` and `404` both as "bad credential".
1.7 **Restricted credential.** "Restricted" only means it cannot create or delete other application credentials or trusts. It still authenticates. Access rules (`path`, `method`, `service`) limit later service calls, so the failure appears as `403` from Nova or Neutron, not from Keystone. Removing the owner's role on the project invalidates the credential. Official, Keystone application credentials guide. A credential with fewer roles than Clutter needs would pass auth and fail on the first create. A cheap extra check is a read call such as `GET /v2.0/networks?limit=1`.
1.8 **Token lifetime.** `token.expires_at` is the token expiry, not the credential expiry. The credential's own `expires_at` is not in the token response I read. Unconfirmed whether a read endpoint exposes it to the credential holder.

### Gophercloud

1.9 Current is **v2.16.0** (pkg.go.dev, listed 2026-11-13; `git ls-remote` tags confirm v2.16.0 is the newest v2 tag). Module path `github.com/gophercloud/gophercloud/v2`. `go.mod` requires Go 1.25.0. Official.
1.10 `gophercloud.AuthOptions` has `IdentityEndpoint`, `ApplicationCredentialID`, `ApplicationCredentialName`, `ApplicationCredentialSecret` (and `UserID`, `Username`, `DomainName` for the name form). Call `openstack.AuthenticatedClient(ctx, opts)`. It returns a `*ProviderClient` after posting to `/auth/tokens`. With an id, it sends the id and secret only. A missing secret returns `ErrAppCredMissingSecret`. Official, `auth_options.go` and `openstack/client.go` at v2.16.0.
1.11 Catalog access: use `tokens3.Create(ctx, identityClient, opts)` from `openstack/identity/v3/tokens`. Then `ExtractServiceCatalog()` gives the catalog and `ExtractToken()` gives `ID` from `X-Subject-Token` and `ExpiresAt`. Service clients come from `openstack.NewComputeV2(provider, gophercloud.EndpointOpts{Region: r})`. `EndpointOpts` has `Type`, `Name`, `Region`, `Availability`. The docs page did not show a catalog-walking example; the function names were read in source.
1.12 **Is it lighter than raw HTTP?** For the check alone, a raw `net/http` POST is about 30 lines and adds no dependency. Gophercloud pays off once B also lists flavors, images, networks and zones, because it picks the endpoint from the catalog by region and type and handles pagination and re-auth. Recommendation: use gophercloud for the connector, since stage 4 needs the list calls too. This is a judgement, not a sourced fact.
1.13 Password `clouds.yaml` parsing is not in the core module. It lives in a separate module, `github.com/gophercloud/utils` (`openstack/clientconfig`). I did not read its current state. B can instead store the fields in its own record and build `AuthOptions` itself.

## 2. Provider 3.4.0 facts

Provider 3.4.0 was published 2025-11-11 and is the latest release on 2026-11-13 (GitHub releases API). Official.

### Provider block and environment

2.1 Arguments: `auth_url`, `region`, `application_credential_id`, `application_credential_name`, `application_credential_secret`, `cloud` (a `clouds.yaml` entry), `user_name`, `user_id`, `password`, `tenant_id` or `tenant_name`, `user_domain_name`, `project_domain_name`, `endpoint_type`, `endpoint_overrides`, `insecure`, `cacert_file`, `delayed_auth`, `allow_reauth`. Official, `docs/index.md`.
2.2 Environment variables: `OS_AUTH_URL`, `OS_REGION_NAME`, `OS_APPLICATION_CREDENTIAL_ID`, `OS_APPLICATION_CREDENTIAL_NAME`, `OS_APPLICATION_CREDENTIAL_SECRET`, `OS_USERNAME`, `OS_USER_ID`, `OS_PASSWORD`, `OS_PROJECT_ID` or `OS_TENANT_ID`, `OS_PROJECT_NAME`, `OS_USER_DOMAIN_NAME`, `OS_PROJECT_DOMAIN_NAME`, `OS_ENDPOINT_TYPE`, `OS_CLOUD`, `OS_INSECURE`, `OS_CACERT`, `OS_DEBUG`. So an empty `provider "openstack" {}` block plus `OS_AUTH_URL`, `OS_REGION_NAME`, `OS_APPLICATION_CREDENTIAL_ID` and `OS_APPLICATION_CREDENTIAL_SECRET` works with no secret in files. The name form needs `user_id` or `user_name` plus `user_domain_name`.
2.3 The provider stores all instance arguments in state, including `admin_pass`. State must stay private (already a rule).

### Resources (all official, `docs/resources/*.md` at v3.4.0)

| Resource | Required | Sensible canvas fields | Notes |
| --- | --- | --- | --- |
| `openstack_compute_instance_v2` | `name`; one of `image_id`/`image_name` unless booting from volume; one of `flavor_id`/`flavor_name` | name, flavor, image, `key_pair`, `security_groups` (names), `availability_zone`, `user_data`, `network { uuid }`, `tags` | Changing image rebuilds. Changing flavor resizes. `network`, `availability_zone`, `user_data` force a new server. Put security groups on a port when ports are used. |
| `openstack_networking_network_v2` | none | `name`, `description`, `admin_state_up`, `tags` | Leave `external`, `shared`, `segments`, `tenant_id` out (admin only). |
| `openstack_networking_subnet_v2` | `network_id` | `cidr`, `ip_version` (4 default), `name`, `enable_dhcp`, `dns_nameservers`, `gateway_ip`/`no_gateway`, `allocation_pool` | `cidr` is optional in the schema (a subnet pool can supply it); the canvas should require it. |
| `openstack_networking_router_v2` | none | `name`, `external_network_id`, `admin_state_up`, `enable_snat` | Needs the external network id for outside access. |
| `openstack_networking_router_interface_v2` | `router_id` | `subnet_id` (or `port_id`) | Docs list neither `subnet_id` nor `port_id` as required; set one. |
| `openstack_networking_secgroup_v2` | `name` | `name`, `description`, `delete_default_rules` | The docs give the `description` text as "A unique name", a copy error. |
| `openstack_networking_secgroup_rule_v2` | `direction`, `ethertype`, `security_group_id` | `direction` (ingress or egress), `ethertype` (IPv4, IPv6), `protocol` (tcp, udp, icmp, or a number 0 to 255 and more names), `port_range_min`, `port_range_max` (1 to 65535), `remote_ip_prefix` (CIDR), `remote_group_id`, `description` | A port range needs a `protocol`. Every argument forces a new rule. `remote_address_group_id` excludes the other two remotes. |
| `openstack_networking_floatingip_v2` | `pool` (external network name) | `pool`, `description`, `port_id`, `subnet_id` | `port_id` can attach it directly. |
| `openstack_blockstorage_volume_v3` | `size` (GB) | `name`, `size`, `volume_type`, `availability_zone`, `description` | `image_id`, `snapshot_id`, `source_vol_id`, `backup_id` create from a source. |
| `openstack_compute_volume_attach_v2` | `instance_id`, `volume_id` | `device` (for example `/dev/vdc`; the docs warn it may not match) | `multiattach`, `tag`, `vendor_options` exist. |

2.4 **Floating IP association in 3.x.** Official upgrade guide: the Compute Floating IPs API was removed from Gophercloud v2, so `openstack_compute_floatingip_v2` and `openstack_compute_floatingip_associate_v2` are gone, and `docs/resources/` at v3.4.0 has no file for them. The replacement is:

```hcl
resource "openstack_networking_floatingip_v2" "ip" { pool = "public" }
data "openstack_networking_port_v2" "vm" {
  device_id  = openstack_compute_instance_v2.vm.id
  network_id = openstack_compute_instance_v2.vm.network.0.uuid
}
resource "openstack_networking_floatingip_associate_v2" "fa" {
  floating_ip = openstack_networking_floatingip_v2.ip.address
  port_id     = data.openstack_networking_port_v2.vm.id
}
```

The `floatingip_associate_v2` arguments are `floating_ip` and `port_id`, both required. Note: this guide shows `network.0.uuid` for a VM attached by network name too, since the exported `network/uuid` is filled in.
2.5 **Other 3.0 removals (official upgrade guide).** `openstack_compute_secgroup_v2` (use the Neutron secgroup and rule pair), the Compute project networks API and `OS_NOVA_NETWORK`, `openstack_blockstorage_volume_v1` and `_v2` (use `_v3`), `openstack_blockstorage_volume_attach_v2` (use `_v3`), and the `fw_*_v1` and `lb_*_v1` resources. The 8 resources in the brief that remain are all present in 3.4.0 under those names.
2.6 Not found: a 3.x argument rename or deprecation inside the eight networking, compute and volume resources. I compared argument lists only, not the CHANGELOG. Unconfirmed.

## 3. Listing choices for forms

Use the project-scoped token from finding 1 and the service URL from the catalog. All official.

3.1 **Flavors.** Nova `GET /flavors/detail` (also `GET /flavors`). It "lists all flavors accessible to your project". Errors 400, 401, 403. Query `sort_key`, `sort_dir`, `limit`, `marker`, plus `is_public` and `minDisk`/`minRam` filters (the last three are listed in the full api-ref; I did not copy their text). Response fields include `id`, `name`, `vcpus`, `ram`, `disk`. Terraform data source: `openstack_compute_flavor_v2` (looks up one flavor by `name`, `ram`, `vcpus`, `min_ram` and so on; it does not list).
3.2 **Images.** Glance `GET /v2/images`. Query `visibility` (`public`, `private`, `community`, `shared`, `all`), `status`, `name`, `os_hidden`, `limit`, `marker`, `sort_key`, `sort_dir`. Default list: public, private and accepted shared images. Hidden images excluded by default. Response `images[]` has `id`, `name`, `status`, `min_disk`, `min_ram`, `disk_format`, `visibility`; paging is by `next`. Filter `status=active` for the form. Data source: `openstack_images_image_v2` (single match with `most_recent`).
3.3 **External networks.** Neutron `GET /v2.0/networks?router:external=true`. Response `networks[]` has `id`, `name`, `status`, `router:external`, `subnets`, `shared`, `is_default`. Data source: `openstack_networking_network_v2` with `external = true`.
3.4 **Availability zones.** Nova `GET /os-availability-zone` returns `availabilityZoneInfo[]` with `zoneName` and `zoneState.available`. Error 401, 403. The `/detail` variant is admin only by default. Terraform data source: `openstack_compute_availability_zones_v2` (`names`, optional `state`). Block storage zones have a separate data source `openstack_blockstorage_availability_zones_v3`. Neutron's `GET /v2.0/availability_zones` was not in the part of the api-ref I could read; unconfirmed.
3.5 Every list depends on the credential's project and roles, so cache lists per connection, not globally.

## 4. Runner on the Mac, DevStack in Lima

4.1 **What DevStack puts in the catalog.** `SERVICE_HOST` defaults to `HOST_IP`; `HOST_IP` is auto-detected from the default interface unless set (DevStack `stackrc`, `stack.sh`). Keystone's URL is `${KEYSTONE_SERVICE_PROTOCOL}://${KEYSTONE_SERVICE_HOST}/identity` with no port (`lib/keystone`). Neutron and Cinder register paths `/networking` and `/volume` on the same host through uwsgi (`lib/neutron`, `lib/cinder`). I did not confirm the Nova and Glance paths. So the catalog hosts are `HOST_IP`, one port (80), different paths.
4.2 **Lima options.** Official Lima docs:

- `vzNAT` (`networks: - vzNAT: true`, needs `vmType: vz`) gives the guest an IP reachable from the Mac but not from other guests. No admin rights, no `socket_vmnet`. The range is not settable.
- `portForwards` forwards guest ports to the Mac. The default rule is guest `127.0.0.1` (also matches `0.0.0.0`) to host `127.0.0.1`, same port number. Default template:

```yaml
portForwards:
- guestPort: 80
  hostPort: 80        # a rule can pick any host port
  # hostIP defaults to 127.0.0.1
```

Unconfirmed: whether macOS lets Lima bind host port 80 without root. Lima's template comment says `hostIP: "0.0.0.0"` "allows privileged port forwarding".
4.3 **Recommended.** Use vzNAT and set DevStack's `HOST_IP` to the guest's vzNAT address. Then the catalog URLs work from the Mac as written and no rewrite is needed. Sketch:

```yaml
vmType: vz
nestedVirtualization: true
networks:
- vzNAT: true
```

In `local.conf`: `HOST_IP=<the lima0 address>` (read it with `limactl shell <vm> ip -4 addr show lima0` after first boot) and, if autodetect picks the other interface, `HOST_IP_IFACE=lima0`. `HOST_IP_IFACE` is an argument in `stack.sh`; its use here is untested. Then the Mac uses `OS_AUTH_URL=http://<that address>/identity/v3`. Trade-off: the address comes from DHCP and can change when the VM is recreated.
4.4 **If port forwarding is preferred.** Catalog URLs still name `HOST_IP` (the VM's own address), which the Mac cannot reach. The Mac side would need a rewrite of each catalog host to `127.0.0.1`. The provider has `endpoint_overrides` (full URLs per service), and B's gophercloud client could swap the host after reading the catalog. That puts a DevStack special case in the connector, which the design forbids for core code. Prefer 4.3.
4.5 The earlier note's option (run the runner inside the VM) still works and needs no network change.

## Contradictions and absences

- The compute api-ref page and WebFetch summary did not agree on flavor filters; I read the HTML text myself for 3.1 and kept to what it showed.
- No statement in the Keystone api-ref says an unscoped password token has an empty catalog. Unconfirmed.
- The registry site is a JavaScript page and returned no text; the provider docs were read from the `v3.4.0` Git tag instead. These are the same files the registry renders.

## Confidence and gaps

High: findings 1.1 to 1.5, 1.9, 1.10, 2.1, 2.2, 2.4, 2.5, 3.1 to 3.3. Medium: 1.6, 1.7, 3.4, 4.1, 4.3 (the vzNAT plan was not run). Not checked: provider CHANGELOG for argument deprecations, Lima 2.3 behavior on binding host port 80, `gophercloud/utils` `clientconfig` state, live DevStack catalog output.

## Open questions

For the team: should `general-openstack` accept password `clouds.yaml` entries in v1, or application credentials only? Default: application credentials only, password as a later add. For the client or a cloud vendor: does the target cloud (Patrii, TierHive) allow application credentials with unrestricted roles, and does it hide flavors or images from `GET` lists for a project-scoped credential?
