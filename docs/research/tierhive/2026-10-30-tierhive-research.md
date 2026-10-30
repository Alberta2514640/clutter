# TierHive as a deployment target

Date: 2026-10-30. Researched by a `researcher` lane; the homepage facts were re-checked by the lead the same day.

Sources:

- TierHive homepage, <https://tierhive.com> (fetched 2026-10-30)
- Operator's forum threads on LowEndSpirit (account "AnthonySmith"): <https://lowendspirit.com/discussion/10336> (December 2025) and <https://lowendspirit.com/discussion/10814> (April 2026)
- Terraform provisioner docs, <https://developer.hashicorp.com/terraform/language/resources/provisioners/syntax>

## What it is

- An hourly NAT VPS host: "© 2026 TierHive by KernelKeepers LTD". Company registration, jurisdiction and head office are not stated.
- Alpha. Version 0.2.2 Alpha was announced in December 2025. The operator calls it "not enterprise grade" and says a small team of three runs it (forum, operator's own posts).
- Plans from $0.10 a month (128 MB RAM) to $3.37 a month (4 GB RAM), a storage plan at $5.44 a month, and a custom calculator up to 8 GB RAM. Billing is prepaid tokens, hourly, with a one-hour minimum.
- Features: a private /24 per customer, IPv6 /64 per location, a WireGuard mesh between locations, built-in HAProxy with Let's Encrypt, snapshots, backups, a noVNC console, one-click recipes, static hosting. OS images: Ubuntu, Debian, AlmaLinux, FreeBSD, OpenBSD.
- Not offered: object storage, block volume API, managed databases, DNS, GPU, serverless.
- Locations: the homepage lists the USA, Germany, France, UK, Canada and Singapore; the HDD option adds Australia. Forum posts claim up to 19 locations. The list is not stable.
- Orders from Brazil, China, Vietnam, Russia and Iran are refused (operator, December 2025).

## Automation

- No public API today. The operator (April 2026): the API exists and the UI is built on it, but the user-facing API is held until beta, "later in 2026 or early 2027", with a full CLI.
- The homepage mentions no API, CLI, Terraform or Ansible. `tierhive.com/api` returned 403; `docs.tierhive.com` does not resolve.
- No Terraform provider was found. The registry search page could not be read, so this is not proven absent.
- No partner or reseller program was found, only a referral program. No link to Alberta 2514640 Inc, Devin Atkin or Patrii was found.

## What "General / custom" usually means

- Terraform with a provider the customer names, plus its credentials and variables.
- SSH to hosts the customer already has, through Ansible with an SSH inventory. HashiCorp advises against Terraform provisioners for this, so Ansible is the usual tool.
- A generic REST connector (Terraform http or restapi providers) is possible but is a community pattern, not a standard one.

## What it means for Clutter

A TierHive connector cannot be built until TierHive publishes its API. Until then, a TierHive VPS a customer created by hand is reachable through the General connector as an existing host (SSH and Ansible). TierHive's feature set maps to compute VM (as-is), config runner (over SSH), and load balancer (new); object storage and serverless are removed.

## Questions

- For TierHive: when do the beta and API ship? Is there an auth model, an OpenAPI file, a Terraform provider or an Ansible collection planned?
- For the client: why TierHive, and is there a relationship with KernelKeepers LTD?
