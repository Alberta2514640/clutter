# Security regression testing

Tests for each fixed System A defect (`defects.tsv`), a wildcard scan of the customer role, and a live cross-tenant and row-level security test against A's real Lambdas.

Run it with `bash tests/security-regression/run.sh`. What it covers, what it needs and where its code lives: [tests/README.md](../README.md).
