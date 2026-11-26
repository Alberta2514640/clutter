# End-to-end and browser compatibility tests

The whole user workflow in a real browser, against the running app, once each in Chromium, Firefox and WebKit:

1. Sign in (through `/dev/login`).
2. Create a diagram from the "Single server" example, in the "New diagram" chooser.
3. Add a block with a click in the palette, and save.
4. Reopen the diagram and find the change.
5. Connect the Mock cloud on the Connections page.
6. Add a deployment on that connection, in the diagram's Deploy tab.
7. Deploy, confirm in the dialog, then watch the status and the log. The deployment's state says "Live", and its next action, "Up to date", is off.
8. Destroy what was built: "Destroy", then "Destroy for good" in an alert dialog. The state says "Destroyed".
9. Remove the deployment from the diagram: "Remove from diagram", then "Remove it" in an alert dialog.

The cloud is B's mock connector, which builds nothing, so no account is needed. The test makes its own organization and project through A's API and deletes them at the end, so it leaves the local data alone. The index of every suite is [test-suites/README.md](../../../test-suites/README.md).

## Run it

Start the app, then run the script from the repo root, on macOS, Linux or WSL:

```sh
npm run dev
bash frontend/clutter/tests/scripts/e2e-browser.sh
```

The script uses the frontend's own Playwright, so it needs `npm ci` in `frontend/clutter` first (it runs it if Playwright is missing). The first run installs Playwright's three browsers and, on Linux, their system libraries, which needs sudo (on macOS nothing needs sudo); set `E2E_INSTALL_DEPS=0` to skip the libraries. On macOS, Playwright's headless Firefox can fail to start (`RenderCompositorSWGL failed`), before any page loads: that is the browser build, not Clutter. So on a Mac the script runs Chromium and WebKit only, and says so; `E2E_FIREFOX=1` adds Firefox. Any extra arguments go to Playwright, for example `--project=firefox` or `--repeat-each=5`. Results go to `test-suites/results/e2e-browser.json`, with a screenshot and a trace of any failure in `test-suites/results/e2e-browser-artifacts/`.

## What is here

- `e2e/workflow.e2e.ts` is the workflow test. It ends in `.e2e.ts`, not `.spec.ts`, so the frontend's Vitest run never picks it up.
- `e2e/playwright.config.ts` sets the three browsers, the timeouts and where results go.
- `scripts/e2e-browser.sh` checks that the app is running, installs the browsers, and runs the test.
