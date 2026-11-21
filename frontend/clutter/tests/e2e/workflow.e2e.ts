import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

// The full user workflow in a real browser, against the running app (A2):
// sign in, create a diagram, save it, reopen it, connect a cloud, choose it
// as the provider of a deployment, deploy, watch the status and log, and
// destroy what was built. The cloud is B's mock connector, which builds
// nothing, so the run needs no account.
//
// The organization and project are made through A's API as fixtures, so the
// test never touches anything already in the local database; it deletes
// them at the end.

const API = process.env.CLUTTER_API_URL ?? "http://127.0.0.1:8787";
const ADAPTER = process.env.CLUTTER_ADAPTER_URL ?? "http://127.0.0.1:8790";

type Fixture = { token: string; orgId: string; projectId: string; connectionName: string };

async function signIn(page: Page): Promise<string> {
  await page.goto("/dev/login");
  await page.waitForURL(/\/(dashboard|onboarding)/, { timeout: 60_000 });
  const cookie = (await page.context().cookies()).find((c) => c.name === "clutter_token");
  expect(cookie, "the dev sign-in sets the clutter_token cookie").toBeTruthy();
  return decodeURIComponent(cookie!.value);
}

async function makeFixture(request: APIRequestContext, token: string, browser: string): Promise<Fixture> {
  const stamp = `${browser}-${Date.now().toString(36)}`;
  const org = await request.post(`${API}/organization`, {
    headers: { Authorization: token },
    data: { organizationName: `E2E ${stamp}`, description: "end-to-end test; safe to delete" },
  });
  expect(org.ok(), `create organization: ${org.status()}`).toBeTruthy();
  const orgId = (await org.json()).data.id as string;
  const proj = await request.post(`${API}/project`, {
    headers: { Authorization: token },
    data: { organizationId: orgId, name: `E2E ${browser}`, description: "end-to-end test" },
  });
  expect(proj.ok(), `create project: ${proj.status()}`).toBeTruthy();
  const projectId = (await proj.json()).data.id as string;
  return { token, orgId, projectId, connectionName: `e2e-${browser}`.slice(0, 32) };
}

async function cleanup(request: APIRequestContext, f: Fixture) {
  const h = { Authorization: f.token };
  const conns = await request.get(`${ADAPTER}/v1/organizations/${f.orgId}/connections`, { headers: h });
  if (conns.ok()) {
    for (const c of (await conns.json()).items ?? []) {
      await request.delete(`${ADAPTER}/v1/organizations/${f.orgId}/connections/${c.id}`, { headers: h });
    }
  }
  await request.delete(`${API}/organization/${f.orgId}`, { headers: h });
}

test("create, save, reopen, choose a provider, deploy, watch, destroy", async ({ page, request, browserName }) => {
  const token = await signIn(page);
  const f = await makeFixture(request, token, browserName);
  await page.evaluate((orgId) => {
    window.localStorage.setItem("clutter-current-organization", JSON.stringify({ state: { organizationId: orgId }, version: 0 }));
  }, f.orgId);

  try {
    await test.step("create a diagram from an example", async () => {
      await page.goto(`/projects/${f.projectId}`);
      // The header keeps a hidden copy while the empty state shows its own.
      await page.getByRole("button", { name: "New diagram" }).filter({ visible: true }).first().click();
      await expect(page.getByTestId("new-diagram-dialog")).toBeVisible();
      await page.getByTestId("new-diagram-example-single-server").click();
      await page.waitForURL(/\/diagram\/[0-9a-f-]{36}/);
      await expect(page.getByTestId("blocks-canvas")).toBeVisible();
      await expect(page.getByTestId("block-node").first()).toBeVisible();
    });
    const diagramURL = page.url();
    const before = await page.getByTestId("block-node").count();
    expect(before).toBeGreaterThanOrEqual(2);

    await test.step("change it and save", async () => {
      await page.getByTestId("palette-volume").click();
      await expect(page.getByTestId("block-node")).toHaveCount(before + 1);
      const save = page.getByRole("button", { name: "Save", exact: true });
      await expect(save).toBeEnabled();
      await save.click();
      await expect(save).toBeDisabled();
      await expect(page.getByTestId("save-status")).toContainText(/saved/i);
    });

    await test.step("reopen it and find the saved change", async () => {
      await page.goto(`/projects/${f.projectId}`);
      await page.goto(diagramURL);
      await expect(page.getByTestId("blocks-canvas")).toBeVisible();
      await expect(page.getByTestId("block-node")).toHaveCount(before + 1);
    });

    await test.step("connect a cloud on the Connections page", async () => {
      await page.goto("/settings/organization/connections");
      await page.getByRole("radio", { name: /Mock cloud/ }).click();
      const form = page.getByRole("region", { name: /Connect Mock cloud/ }).or(page.locator('section[aria-label="Connect Mock cloud"]'));
      await form.getByPlaceholder("for example lab").fill(f.connectionName);
      await form.locator('input[type="password"]').first().fill("mock-valid-token");
      await form.getByRole("button", { name: "Check and connect" }).click();
      await expect(page.getByTestId("connection-row").filter({ hasText: f.connectionName })).toBeVisible();
    });

    await test.step("choose that provider for a deployment", async () => {
      await page.goto(diagramURL);
      await page.getByRole("tab", { name: "Deploy" }).click();
      await page.getByRole("button", { name: "Add deployment" }).first().click();
      await page.locator("label").filter({ hasText: f.connectionName }).click();
      await page.getByRole("button", { name: `Add deployment to ${f.connectionName}` }).click();
      await expect(page.getByTestId("deployment-row")).toHaveCount(1);
      await expect(page.getByTestId("compatibility")).toBeVisible();
    });

    const row = page.getByTestId("deployment-row");
    await test.step("deploy and watch the status and log", async () => {
      await row.getByRole("button", { name: "Deploy", exact: true }).click();
      const confirm = page.getByTestId("deploy-confirm");
      await expect(confirm).toBeVisible();
      await confirm.getByRole("button", { name: "Deploy", exact: true }).click();
      await expect(row.getByTestId("run-status")).toHaveAttribute("data-status", "completed", { timeout: 60_000 });
      await expect(row.getByTestId("run-status")).toContainText("Deploy");
      await row.getByRole("button", { name: "Show log" }).click();
      await expect(row.getByTestId("run-log")).toContainText(/\S/);
      await expect(row.getByTestId("deployment-state")).toHaveText("Live");
      // With nothing changed since the deploy, the next action is off.
      await expect(row.getByRole("button", { name: "Up to date" })).toHaveAttribute("aria-disabled", "true");
    });

    await test.step("destroy what was built", async () => {
      await row.getByRole("button", { name: "Destroy", exact: true }).click();
      await page.getByRole("alertdialog").getByRole("button", { name: "Destroy for good" }).click();
      await expect(row.getByTestId("run-status")).toContainText("Destroy", { timeout: 60_000 });
      await expect(row.getByTestId("run-status")).toHaveAttribute("data-status", "completed", { timeout: 60_000 });
      await expect(row.getByTestId("deployment-state")).toHaveText("Destroyed");
    });

    await test.step("remove the deployment from the diagram", async () => {
      await row.getByRole("button", { name: "Remove from diagram" }).click();
      await page.getByRole("alertdialog").getByRole("button", { name: "Remove it" }).click();
      await expect(page.getByTestId("deployment-row")).toHaveCount(0);
    });
  } finally {
    await cleanup(request, f);
  }
});
