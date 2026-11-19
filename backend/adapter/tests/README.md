# System B's test-plan suites

Five of the team's test types test System B (the adapter layer). Their Go tests are here, beside the code, because they use B's `internal/` packages, which Go only lets code under `backend/adapter` import. CI runs them on every pull request through the checks workflow's `go -C backend/adapter test ./...`. The index of every suite is [tests/README.md](../../../tests/README.md).

## Run them

From the repo root, in Linux or WSL. Each script writes `tests/results/<suite>.json`:

```sh
bash backend/adapter/tests/scripts/unit.sh
bash backend/adapter/tests/scripts/connector-contract.sh
bash backend/adapter/tests/scripts/input-validation.sh
bash backend/adapter/tests/scripts/state-isolation.sh
bash backend/adapter/tests/scripts/performance.sh
```

Or run the Go tests directly: `go -C backend/adapter test ./tests/...`. The validation, isolation and performance suites need Docker for a throwaway Postgres; without it, the tests that need it skip. Nothing here touches a real cloud.

## The suites

| Folder | Test type | What it proves |
| --- | --- | --- |
| `unit/` | Unit testing | Each component on its own: the Translator (reads a blocks diagram, refuses 12 kinds of broken document), manifest handling (every shipped manifest is well formed, 8 kinds of broken manifest are refused, value checks per field type), the Capability Check's 13 rules, the connector registry's lookups, and the job runner's status tracking (queued, running, completed or failed; the deployment's status; safe failure reasons; credentials removed from logs; queue limits). |
| `contract/` | Connector contract testing | The one shared contract suite (`connectors/contract`) passes for General's OpenStack connector, the AWS connector and the mock, side by side, through B's real job runner. Patrii has a row that skips: it has no connector yet. |
| `validation/` | Input and capability validation testing | The approved set of invalid inputs, V01 to V32 (missing images or flavors, invalid dependencies, AWS-only resources on other clouds, malformed network data), is refused before anything runs. Unknown providers are refused by the registry and by B API, and B API refuses to preview or run an unready deployment. |
| `isolation/` | State isolation and security testing | Four applies at once, two organizations by two providers. Each run sees only its own credentials and writes only its own state, neither organization can read or change the other's deployments, and seeded secrets appear in no diagram, row, response, log, file or repository change. The cap of 3 runs per organization holds. |
| `performance/` | Performance testing | 20 Capability Check requests and 5 deployments through B API's routes, with the median and maximum. The numbers are B's own cost; a real cloud adds its own time. Set `PERF_RESULTS` to also write them as JSON. |
| `testkit/` | (shared) | A stand-in for the terraform binary, a fake Keystone, a fake STS, the three connectors wired to them, and B API in-process on a throwaway Postgres. |
| `scripts/` | (scripts) | One script per suite, as above. |
