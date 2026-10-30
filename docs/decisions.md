# Decisions

One entry per decision: the date, the status, what was decided, why, and what would change it. A superseded entry stays, marked with what replaced it.

## ADR-001: Capability-based provider adapters, not a lowest common denominator (2026-10-26, proposed)

Each cloud provider declares the resource types it supports and the canvas shows only those. Why: Patrii has no serverless, API gateway or managed NoSQL table, and a common-denominator model would remove those from AWS users too. Prior art that works this way: Terraform providers, Crossplane compositions, Cluster API providers. Would change if: the client wants one abstract diagram that deploys unchanged to several clouds.

## ADR-002: Terraform stays the engine for every provider (2026-10-26, proposed)

Adapters emit HCL for an existing Terraform provider; Clutter does not call cloud APIs to create resources itself. Why: plan, apply, state and drift handling are already solved there, and the runner and logs already speak Terraform. Would change if: a target cloud has no Terraform provider and no OpenStack-compatible API.

## ADR-003: Patrii is targeted through public OpenStack APIs first, its own provider second, and never its console's private API (2026-10-26, proposed)

Why: the standard OpenStack Terraform provider and Keystone application credentials are documented and portable to other OpenStack clouds; Patrii's own provider is not released; the console API is undocumented and changes with each build. Contested: the second research run on the same date recommends coding against the console's REST API now (Path C in `provider-adapter.md`), since it is the only surface that works today and Patrii's homepage never names OpenStack services. Would change if: Patrii states it will not expose OpenStack endpoints and no provider is planned, or publishes an OpenAPI file and calls its REST API the supported surface.

## ADR-005: The central standard is a versioned inbound API over Clutter's own model (2026-10-26, proposed)

Other products, and people, call one `/v1` API with six objects (organization, workspace, cloud account, desired state, run, catalog). It sits in front of the existing handlers and translates; target adapters sit behind it. Why: the existing org, project and diagram path with its membership check is sound enough to wrap, while the internal routes are inconsistent (ids in paths and queries, mixed names, no error envelope, no idempotency). Freezing them as the public contract would freeze those gaps. Would change if: the only consumers are Clutter's own frontend, in which case the outbound adapters alone are enough.

## ADR-004: Clutter stays SaaS on AWS while the adapter is built; the runner contract is kept BYOC-ready (2026-10-26, proposed)

Why: moving the platform off Lambda and API Gateway is a separate project, and nothing about Patrii requires it before a Patrii adapter works. Keeping credentials and state behind `RunnerEnv` and `StateBackend` leaves room to run the deployer inside a customer's cloud later. Would change if: the client needs Canadian-only hosting of Clutter itself (data residency, CLOUD Act exposure), in which case hosting on Patrii or BYOC moves before phase 3.
