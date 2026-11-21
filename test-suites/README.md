# Test suites

The team's test plan has ten test types. Each suite sits beside the code it tests, in that code's `tests/` folder, with its scripts in a `scripts/` subfolder and a README that says what it covers and how to run it. This folder holds what the suites share and the one test that covers the whole repo.

## Run them

From the repo root, on macOS, Linux or WSL (the scripts are bash, and run on macOS's own bash 3.2):

```sh
npm run dev                       # optional: the live security checks and the browser suite need the app running
bash test-suites/scripts/run-all.sh     # every suite, then a table of results
```

Each suite writes its results to `test-suites/results/` (ignored by git), or to `RESULTS_DIR` if it is set. `run-all.sh` keeps going past a failed suite and exits 1 if any failed. A suite that exits 2 is shown as skipped and does not fail the run; the browser suite does that when the app is not running. The security regression suite's live part instead prints that it skipped and still passes, and it finds the app only when it was started from the same checkout. CI runs the Go tests and the platform check; the scripts and the browser suite run by hand.

## Where each suite lives

| Test type | Tests | Script | README |
| --- | --- | --- | --- |
| Unit testing | `backend/adapter/tests/unit` | `backend/adapter/tests/scripts/unit.sh` | [backend/adapter/tests](../backend/adapter/tests/README.md) |
| Connector contract testing | `backend/adapter/tests/contract` | `backend/adapter/tests/scripts/connector-contract.sh` | [backend/adapter/tests](../backend/adapter/tests/README.md) |
| Input and capability validation testing | `backend/adapter/tests/validation` | `backend/adapter/tests/scripts/input-validation.sh` | [backend/adapter/tests](../backend/adapter/tests/README.md) |
| State isolation and security testing | `backend/adapter/tests/isolation` | `backend/adapter/tests/scripts/state-isolation.sh` | [backend/adapter/tests](../backend/adapter/tests/README.md) |
| Performance testing | `backend/adapter/tests/performance` | `backend/adapter/tests/scripts/performance.sh` | [backend/adapter/tests](../backend/adapter/tests/README.md) |
| Security regression testing | A's own tests, listed in `defects.tsv`, and `backend/devserver/tests/crosstenant` | `backend/api/tests/scripts/security-regression/run.sh` | [backend/api/tests](../backend/api/tests/README.md) |
| AWS regression testing | A's whole suite, at `BASELINE` and now | `backend/api/tests/scripts/aws-regression/run.sh` | [backend/api/tests](../backend/api/tests/README.md) |
| End-to-end and browser compatibility testing | `frontend/clutter/tests/e2e` | `frontend/clutter/tests/scripts/e2e-browser.sh` | [frontend/clutter/tests](../frontend/clutter/tests/README.md) |
| Platform and version check | the whole repo's pins | `test-suites/scripts/platform-version/run.sh` | below |
| Patrii Cloud integration testing | not built yet | | below |

## What is in this folder

- `scripts/run-all.sh` runs every suite in order.
- `scripts/lib/common.sh` holds the settings every suite's script shares: where the repo and the results are, and `run_go_suite`, which runs one Go package with `-json`.
- `scripts/lib/gotest_summary.py` turns `go test -json` output into one results file per suite.
- `scripts/platform-version/` is the **platform and version check**. It checks that the toolchains and images are pinned, that the pins agree (Go in all three modules, Node, Postgres, Terraform, the OpenStack and AWS providers), that CI builds on them, and that this machine meets them. Each check is PASS, WARN (pinned loosely) or FAIL. CI runs it on every pull request (the checks workflow's `platform` job). Run it with `bash test-suites/scripts/platform-version/run.sh`.

## Not built yet: Patrii Cloud integration testing

Patrii publishes no customer API, OpenStack endpoint or Terraform provider (`docs/research/patricloud/`), so there is no connector and no sandbox to script against. The contract suite keeps a Patrii row that skips with that reason. When Patrii opens OpenStack endpoints it becomes a connection of `general-openstack`, and the lifecycle script is that connector's real-cloud test (`TestRealCloud`, run today against DevStack) pointed at a Patrii sandbox project.

## Results

The results of the 2026-11-18 run, with what each one means, are in `docs/reports/2026-11-18-test-plan-results.pdf`. The bug that run found (the deploy panel could keep saying "Not deployed" after a deploy) is fixed in B: a run's end and its deployment's status are now saved in one transaction. On 2026-11-20 every suite passed on this branch merged with `main`, on a Mac: the browser workflow 10 of 10 (Chromium and WebKit) and the live security checks.
