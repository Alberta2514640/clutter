# Tests

One folder per test type from the team's test plan. Each folder has a `run.sh` that runs that type and writes its results to `tests/results/` (ignored by git). `run-all.sh` runs them all in order and prints a table.

Run everything from the repo root, in Linux or WSL (the scripts are bash):

```sh
npm run dev                 # optional: the live security checks and the browser suite need the app running
bash tests/run-all.sh       # or one type: bash tests/unit/run.sh
```

## Where each type lives

The Go suites for System B use B's internal packages, which Go only lets code under `backend/adapter` import. So their test code is in `backend/adapter/tests/<suite>`, and the folder here holds the script that runs them. CI runs that code on every pull request through the checks workflow's `go -C backend/adapter test ./...`.

| Test type | Folder | Test code | Needs |
| --- | --- | --- | --- |
| Unit testing | `unit/` | `backend/adapter/tests/unit` | Go |
| Connector contract testing | `connector-contract/` | `backend/adapter/tests/contract` | Go |
| Input and capability validation testing | `input-validation/` | `backend/adapter/tests/validation` | Go; Docker for the API cases |
| State isolation and security testing | `state-isolation/` | `backend/adapter/tests/isolation` | Go, Docker |
| Performance testing | `performance/` | `backend/adapter/tests/performance` | Go, Docker |
| Security regression testing | `security-regression/` | `defects.tsv` (A's own tests), `check_role.py`, `backend/devserver/tests/crosstenant` | Go, Python 3; the running backend for the live part |
| AWS regression testing | `aws-regression/` | A's whole suite, at `BASELINE` and now | Go, Python 3, git |
| End-to-end and browser compatibility testing | `e2e-browser/` | `workflow.spec.ts` (Playwright) | Node, the running app |
| Platform and version check | `platform-version/` | `check_versions.py` | Python 3; CI runs it on every pull request |

Shared pieces: `testkit` in `backend/adapter/tests/testkit` (a stand-in for the terraform binary, a fake Keystone, a fake STS, and B API in-process on a throwaway Postgres), and `lib/` here (the results summarizer and shared settings).

## Not implementable yet

**Patrii Cloud integration testing** has no folder. Patrii publishes no customer API, OpenStack endpoint or Terraform provider (`docs/research/patricloud/`), so there is no connector and no sandbox to script against. The contract suite keeps a Patrii row that skips with that reason. When Patrii opens OpenStack endpoints it becomes a connection of `general-openstack`, and the lifecycle script is that connector's real-cloud test (`TestRealCloud`, today run against DevStack) pointed at a Patrii sandbox project.

## What each suite proves

- **Unit:** the translator, manifest handling, the capability check's rules, the connector registry's lookups, and the job runner's status tracking, one component at a time.
- **Connector contract:** the one shared suite (`connectors/contract`) passes for General's OpenStack connector, the AWS connector and the mock, side by side, through B's real job runner.
- **Input and capability validation:** the approved set of invalid inputs (V01 to V32: missing images or flavors, invalid dependencies, AWS-only resources elsewhere, unknown providers, malformed network data) is refused before anything runs, and nothing from it reaches a preview.
- **State isolation and security:** four applies at once, two organizations by two providers. Each run sees only its own credentials and state, neither organization can reach the other, and seeded secrets appear in no diagram, row, response, log, file or repository change. The per-organization run cap holds.
- **Performance:** 20 Capability Check requests and 5 deployments through B API, with the median and maximum.
- **Security regression:** each fixed System A defect has tests that still pass; the customer role has no wildcard grant; one user cannot read or change another organization's projects, diagrams or Terraform through A's real Lambdas; Postgres's `anon` role reads no tenant row.
- **AWS regression:** A's whole AWS suite at the recorded revision before B was integrated, and now, compared test by test.
- **End-to-end and browser:** sign in, create, save and reopen a diagram, connect a cloud and choose it for a deployment, deploy, watch the status and log, and destroy, in Chromium, Firefox and WebKit.
- **Platform and version:** toolchains and images are pinned, the pins agree, CI builds on them, and the machine meets them.
