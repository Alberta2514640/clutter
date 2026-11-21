# System A's tests

This folder holds Go tests of A's shared packages (`generic/`, `ansible/`, `terraform-engine/`, `user-information/`), beside the tests that sit next to each Lambda. It also holds the scripts for the two test suites that test A: security regression and AWS regression. The index of every suite is [test-suites/README.md](../../../test-suites/README.md).

## Run them

From the repo root, on macOS, Linux or WSL:

```sh
go -C backend/api test ./...                                    # all of A's tests
bash backend/api/tests/scripts/security-regression/run.sh       # security regression
bash backend/api/tests/scripts/aws-regression/run.sh            # AWS regression
```

Both scripts write their results to `test-suites/results/`. For the live part of the security regression suite, start the app first (`npm run dev:backend`, or `npm run dev`) from the same checkout: the suite reads `backend/devserver/.state/pg_password` in its own tree. Without it, or when the gateway at 127.0.0.1:8787 does not answer, the live part prints that it skipped and the script still exits 0.

## Security regression testing (`scripts/security-regression/`)

- `defects.tsv` has one row per System A defect that stage 1 fixed (D01 to D23), naming the A tests that prove it stays fixed. The script runs each row's tests. A row passes only when at least one test ran and none failed; a row whose tests no longer exist shows as MISSING.
- `check_role.py` scans the customer role template (`client_side_terraform_deployer_role.yaml`) and fails on any Allow statement whose action is `*` or a whole service, such as `iam:*`. A wildcard in a Deny statement is allowed, because a deny only takes permissions away.
- With the backend running, the script also runs the live tests in `backend/devserver/tests/crosstenant` against A's real Lambdas. In the cross-tenant test, one user tries to read or change another organization's projects, diagrams and Terraform, and every attempt must be refused with 403 or 404. In the row-level security test, Postgres's `anon` role must read no tenant row.

## AWS regression testing (`scripts/aws-regression/`)

- `BASELINE` records the source revision to compare against: `a9cc82b`, A before stage 1 and before System B was merged.
- The script checks out that revision in a temporary worktree, runs A's whole suite there and in the current tree, and compares the results test by test (`compare.py`). With `CLUTTER_TF_VALIDATE=1`, which the script sets, rendered AWS diagrams also go through `terraform validate`; the first run downloads the AWS provider.
- A test that passed at the baseline and fails, or is missing, now is a regression, and the script exits 1. A new test that fails, or a package that fails to build or fails in the current tree, also fails the run. Tests that already failed at the baseline are listed but do not fail the run.
- `renames.tsv` maps tests renamed since the baseline to the tests that replace them, each with a reason. Each row was checked by hand against the code. Add a row only when a rename is real.
