---
name: ground-before-building
description: Ground new work in current outside reality before coding - fetch current docs for the Terraform provider, cloud API, Go library or Next.js API you are about to use, so the design is not built on stale memory. Fire at the start of feature work, before adding or changing a cloud resource or provider, or when an external API may have changed.
---

# Ground before building

Clutter generates infrastructure code for clouds whose APIs and Terraform providers change every few months, and its own frontend runs on Next.js 16 and React 19, where remembered APIs are often wrong. A few tool calls up front cost less than debugging a template written against a resource that no longer has that argument.

## When to run

- Starting a feature or a non-trivial change.
- Adding or changing a resource template (`*.tf.tmpl`), a provider, or a runner step.
- Touching a library or framework API you have not checked this session.
- Skip it for one-line fixes, copy edits and internal refactors with no outside API in play.

## The pass

1. **List the outside surfaces** the change touches: each Terraform provider and resource, each cloud API, each library.
2. **Pull current docs.** For libraries, use context7 (resolve the library id, then query the exact API, pinned to the version in `go.mod` or `package.json`). For Terraform, read the resource's page on the Terraform Registry for the provider version the template pins (`registry.terraform.io/providers/hashicorp/aws/latest/docs`, `registry.terraform.io/providers/terraform-provider-openstack/openstack/latest/docs`). For a cloud API, read the vendor's API reference.
3. **Check what the target cloud actually exposes.** A resource in the Terraform provider is useless if the cloud does not run that service (an OpenStack cloud without Octavia has no load balancer). The research notes in `docs/research/` hold what we know per provider; confirm anything older than a month.
4. **Reconcile with the repo.** The root and folder `CLAUDE.md` files and `engineering-conventions` win over generic web advice.
5. **Then design and build.** Fold any changed API or better current pattern into the plan first.

For a broad or contested question, the lead delegates to the `researcher` agent; a lane does its own grounding inline.
