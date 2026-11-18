# Bug: the deploy panel can keep saying "Not deployed" after a deploy succeeds

Found by this suite on 2026-11-18. Status: **open**. The full test report is `docs/reports/2026-11-18-test-plan-results.pdf` (Finding 1).

## What the user sees

After a successful deploy, the deployment's row in the deploy panel shows the finished run ("Deploy finished", "Built 4 blocks", and "Apply complete" in the log). Its badge still says **Not deployed**, and Destroy stays off with "Nothing is built yet." This lasts until the page is reloaded.

## How often

In 5 of 12 runs of `workflow.spec.ts`, at the step "deploy and watch the status and log":

| Browser | First run | Repeated 3 times |
| --- | --- | --- |
| Chromium | passed | 1 of 3 passed |
| Firefox | failed | 1 of 3 passed |
| WebKit | passed | 3 of 3 passed |

It happens in more than one browser, so it is a timing bug in Clutter, not a browser difference. Every step before deploying passed in every run.

## Cause

B's job runner (`backend/adapter/internal/jobs`, `Runner.run`) records the end of a run with two separate database writes:

1. `RunFinished` marks the run completed.
2. `SetDeploymentStatus` then marks the deployment live.

The deploy panel reads the deployment list (`GET .../diagrams/{id}/deployments`), which returns each deployment's status and its latest run in one answer. A read that lands between the two writes gets a completed run on a deployment that is still `new`.

The panel only keeps polling while a run is active (`useDiagramDeployments` in `frontend/clutter/src/lib/features/deployments/hooks.ts`). After that one mixed answer it sees no active run and stops asking, so the wrong badge stays.

A throwaway check against B API alone confirmed the gap: polling the list during 20 applies gave 1 mixed answer in 211 reads. The browser hits it far more often. It refreshes the list the moment the run starts, and the mock's run finishes in about 25 ms, right inside the gap. On a real cloud a run takes minutes and the panel polls every 2 seconds, so it is rarer there, but it can still happen.

## How to reproduce

1. Start the app: `npm run dev`.
2. Run the workflow several times: `cd tests/e2e-browser && npx playwright test --repeat-each=3`.
3. A failed run stops at `expect(row).toContainText("Deployed")`, and its screenshot shows the finished run beside the "Not deployed" badge.

## Fix (both parts recommended)

1. **In B:** record the run's end and the deployment's new status in one transaction, so no reader can see one without the other. A smaller change that fixes this symptom is to set the deployment status before marking the run completed.
2. **In the frontend:** in `useDiagramDeployments`, refresh the list one more time when a run changes from active to finished, instead of stopping on the first answer with no active run.

## How to confirm the fix

Run `npx playwright test --repeat-each=5` in this folder with the app running. All 15 runs (5 in each browser) should pass.
