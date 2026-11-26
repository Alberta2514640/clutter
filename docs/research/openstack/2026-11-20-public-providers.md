# Public OpenStack providers for a real-cloud test (2026-11-20)

Read 2026-11-20 unless a source says otherwise. The question: where can the team open a pay-as-you-go account this week and test the General connector's OpenStack provider for real (a few small VMs for a few hours), Canada first.

The connector needs: an `https` Keystone v3 address with application credentials for normal users; Neutron routers on an external network and floating IPs; Cinder volumes; Nova with small flavors and a Linux image with cloud-init; terraform-provider-openstack v3.x; and self-service signup with a card.

## Findings

| Provider | Regions in Canada or the US | Application credentials | Router, floating IP, volumes | Signup | Smallest VM, floating IP | Free credit |
| --- | --- | --- | --- | --- | --- | --- |
| VEXXHOST | `ca-ymq-1` Montreal, `sjc1` Santa Clara | yes, documented; auth URL `https://auth.vexxhost.net/v3` | priced (router, floating IP, NVMe block storage), so they exist; user access not confirmed from API docs | card or PayPal, no contract; the order is reviewed before activation | `v3-starter-1` (1 vCPU, 2 GB) $0.0106/h; floating IP $0.0028/h; billed per minute (currency not stated) | none stated |
| Rackspace OpenStack Flex | IAD, DFW, SJC (US only) | yes, documented (Skyline) | OVN router priced; floating IP price not found | conflicting: a self-service cart, but another page says Flex is not readily available without sales | about $0.0198 (unit not shown) | unclear |
| OVHcloud Public Cloud | BHS (Beauharnois, Canada) | not found: its docs use password users and OIDC service accounts | gateway and floating IP exist; prices not read | card after the trial | not read | CA$270 on the Canadian site |
| OpenMetal | not confirmed | not found | not checked | 8-hour trial; monthly dedicated hardware | n/a | trial |
| Leafcloud, Cleura, Infomaniak | Europe only | Leafcloud yes (third-party guide); others not checked | not checked | prepay or card | not read | Cleura $50 voucher |

Not checked: Binero; any provider's cloud-init behaviour. terraform-provider-openstack v3.4.0 (2025-11-11) is the release the connector is tested with; no provider publishes its own test of it.

## Recommendation

VEXXHOST `ca-ymq-1`: Canadian, per-minute billing, no minimum, application credentials documented. A test of a few hours costs cents. Trade-off: no free credit, and the order review may delay the account, so sign up first. If the review takes days, fall back to Rackspace Flex SJC or OVHcloud BHS, after confirming application credentials there.

A new project has an external network, Ubuntu images and quota for routers and floating IPs: see the account check below. Our account worked on the day it was registered.

## Checked in a VEXXHOST account (2026-11-20)

A new pay-as-you-go project, read with the OpenStack CLI 10.5.0 through an application credential (`auth_type: v3applicationcredential`, `https://auth.vexxhost.net/v3`), which signed in. Nothing was created.

- Regions: `ams1`, `ca-east-1`, `ca-ymq-1`, `eu-west-1`, `sjc1`. A `clouds.yaml` downloaded from the dashboard names one region; it came set to `ams1`, and changing it to `ca-ymq-1` worked.
- `ca-ymq-1` offers compute, network, `volumev3`, image, load balancer, DNS and object storage.
- External network: `public`, the same name as DevStack's and the connector's default.
- Images: plain Ubuntu 22.04 and 24.04, each built for both `arm64` and `x86_64` with the same date (for example `Ubuntu 24.04.2 LTS (x86_64) [2025-04-03]`), so a name prefix such as `Ubuntu 24.04` can match the wrong processor. No `cirros`, the connector's default image.
- Smallest suitable flavor: `v3-starter-1` (2 GB memory, 10 GB disk, 1 vCPU). The `m1.*` names are absent.
- Quotas: 10 instances, 20 cores, 50 GB memory, 10 volumes and 1000 GB, 10 routers, 100 networks, 50 floating IPs, 10 security groups and 100 rules.

## Sources

- VEXXHOST application credentials: <https://docs.vexxhost.com/identity/application-credentials.md>, <https://docs.vexxhost.com/developer/cli.md>
- VEXXHOST regions: <https://vexxhost.com/company/data-centers>
- VEXXHOST prices: <https://vexxhost.com/pricing/>
- VEXXHOST signup: <https://vexxhost.com/faq/>
- Rackspace Flex: <https://docs.rackspace.com/docs/application-credentials-in-openstack-flex>, <https://docs.rackspace.com/docs/accessing-rackspace-flex-cloud>, <https://docs.rackspace.com/docs/openstack-flex-customer-handbook>, <https://manage.rackspace.com/service-catalog/en/openstack-flex>
- OVHcloud: <https://www.ovhcloud.com/en/public-cloud/faq/>, <https://www.ovhcloud.com/en-ca/public-cloud/free-trial/>, <https://docs.ovhcloud.com/en/guides/public-cloud/cross-functional/managing-tokens> (updated 2026-10-26)
- OpenMetal: <https://openmetal.io/free-trial>
- Leafcloud: <https://docs.avisi.cloud/docs/product/tasks/cloud-accounts/leafcloud>
- Infomaniak: <https://www.infomaniak.com/en/support/faq/admin2/public-cloud>
- Cleura: <https://cleura.com/free-trial>
- terraform-provider-openstack releases: <https://github.com/terraform-provider-openstack/terraform-provider-openstack/releases>
