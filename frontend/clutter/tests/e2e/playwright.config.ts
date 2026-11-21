import { defineConfig, devices } from "@playwright/test";

// The whole user workflow, once per browser engine. The app must already be
// running (npm run dev); tests/scripts/e2e-browser.sh checks that and points
// the test at it.
export default defineConfig({
  testDir: ".",
  // .e2e.ts, not .spec.ts, so Vitest's default pattern never picks it up.
  testMatch: /.*\.e2e\.ts/,
  timeout: 180_000,
  expect: { timeout: 30_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["list"], ["json", { outputFile: process.env.E2E_JSON ?? "results.json" }]],
  use: {
    baseURL: process.env.CLUTTER_WEB_URL ?? "http://localhost:3000",
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
    viewport: { width: 1440, height: 900 },
  },
  outputDir: process.env.E2E_ARTIFACTS ?? "test-results",
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 900 } } },
    { name: "firefox", use: { ...devices["Desktop Firefox"], viewport: { width: 1440, height: 900 } } },
    { name: "webkit", use: { ...devices["Desktop Safari"], viewport: { width: 1440, height: 900 } } },
  ],
});
