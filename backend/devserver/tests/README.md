# Live tests against A2

Tests that need the running backend (A's real Lambdas through A2, and its Postgres). Without it they skip.

- `crosstenant/` is the live part of the security regression suite. In the cross-tenant test, one user tries 24 reads and changes on another organization's projects, diagrams and Terraform, and every one must be refused with 403 or 404, with the owner's data left as it was. In the row-level security test, Postgres's `anon` role must read no tenant row.

Run them through the security regression suite, which starts nothing itself but finds the running backend and passes its address and secrets in:

```sh
npm run dev:backend
bash backend/api/tests/scripts/security-regression/run.sh
```

What the suite covers is in [backend/api/tests/README.md](../../api/tests/README.md); the index of every suite is [test-suites/README.md](../../../test-suites/README.md).
