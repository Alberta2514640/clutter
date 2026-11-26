import { describe, expect, it } from "vitest";
import { failureAdvice, failureLines, runHeadline, runResult } from "./runLog";

describe("runResult", () => {
  it("reads Terraform's apply summary and drops the zero counts", () => {
    const log = "aws_vpc.main: Creating...\n\u001b[0m\u001b[1m\u001b[32mApply complete! Resources: 24 added, 0 changed, 0 destroyed.\u001b[0m\n";
    expect(runResult(log, "completed")).toEqual({ tone: "ok", text: "Apply complete: 24 added" });
  });

  it("reads plan, destroy and no-change summaries", () => {
    expect(runResult("Plan: 3 to add, 1 to change, 0 to destroy.", "completed")?.text).toBe("Plan: 3 to add, 1 to change");
    expect(runResult("Destroy complete! Resources: 5 destroyed.", "completed")?.text).toBe("Destroy complete: 5 destroyed");
    expect(runResult("No changes. Your infrastructure matches the configuration.", "completed")?.text).toBe("No changes");
    expect(runResult("Apply complete! Resources: 0 added, 0 changed, 0 destroyed.", "completed")?.text).toBe("Apply complete: no changes");
  });

  it("states a plan once when B repeats Terraform's summary", () => {
    const log = "Plan: 19 to add, 0 to change, 0 to destroy.\n\nPlan: 19 to add, 0 to change, 0 to destroy.\n";
    expect(runResult(log, "completed")?.text).toBe("Plan: 19 to add");
  });

  it("gives an apply Terraform's count, not its plan or B's block count", () => {
    const log = [
      "Run r1: apply on deployment d1.",
      "Plan: 19 to add, 0 to change, 0 to destroy.",
      "aws_s3_bucket.b: Creating...",
      "Apply complete! Resources: 19 added, 0 changed, 0 destroyed.",
      "Apply complete: 6 blocks built or changed.",
    ].join("\n");
    expect(runResult(log, "completed")?.text).toBe("Apply complete: 19 added");
  });

  it("drops the plan Terraform prints inside a destroy, and an apply with nothing to do", () => {
    expect(runResult("Plan: 0 to add, 0 to change, 4 to destroy.\nDestroy complete! Resources: 4 destroyed.\nDestroy complete.\n", "completed")?.text).toBe(
      "Destroy complete: 4 destroyed",
    );
    expect(
      runResult("No changes. Your infrastructure matches the configuration.\nApply complete! Resources: 0 added, 0 changed, 0 destroyed.\nApply complete: everything is up to date.\n", "completed")?.text,
    ).toBe("Apply complete: no changes");
  });

  it("reads B's block count when no Terraform ran", () => {
    expect(runResult("Created mock-1 (vm).\nApply complete: 1 block built or changed.\n", "completed")?.text).toBe("Apply complete: 1 block built or changed");
    expect(runResult("Apply complete: everything is up to date.\n", "completed")?.text).toBe("Apply complete: everything is up to date");
  });

  it("gives each cloud of a split run its own line", () => {
    const log = [
      "== Lab (Mock cloud): apply ==",
      "Lab (Mock cloud): apply complete, everything is up to date.",
      "== Prod (AWS): apply ==",
      "Plan: 2 to add, 0 to change, 0 to destroy.",
      "Apply complete! Resources: 2 added, 0 changed, 0 destroyed.",
      "Prod (AWS): apply complete, 1 block built or changed.",
    ].join("\n");
    expect(runResult(log, "completed")?.text).toBe("Lab (Mock cloud): everything is up to date; Prod (AWS): 1 block built or changed");
  });

  it("adds up a split destroy across its clouds, once per cloud", () => {
    const log = [
      "== Prod (Amazon Web Services): destroy ==",
      "Plan: 0 to add, 0 to change, 7 to destroy.",
      "Destroy complete! Resources: 7 destroyed.",
      "Destroy complete! Resources: 7 destroyed.",
      "== Lab (OpenStack): destroy ==",
      "Plan: 0 to add, 0 to change, 10 to destroy.",
      "Destroy complete! Resources: 10 destroyed.",
    ].join("\n");
    expect(runResult(log, "completed")?.text).toBe("Destroy complete: 17 destroyed");
  });

  it("adds up a split apply when every cloud ran Terraform", () => {
    const log = [
      "== Lab (OpenStack): apply ==",
      "Plan: 10 to add, 0 to change, 0 to destroy.",
      "Apply complete! Resources: 10 added, 0 changed, 0 destroyed.",
      "Apply complete! Resources: 10 added, 0 changed, 0 destroyed.",
      "Lab (OpenStack): apply complete, 4 blocks built or changed.",
      "== Prod (Amazon Web Services): apply ==",
      "Plan: 7 to add, 1 to change, 0 to destroy.",
      "Apply complete! Resources: 7 added, 1 changed, 0 destroyed.",
      "Prod (Amazon Web Services): apply complete, 3 blocks built or changed.",
    ].join("\n");
    expect(runResult(log, "completed")?.text).toBe("Apply complete: 17 added, 1 changed");
  });

  it("gives the first error of a failed run, in plain words, with its cloud", () => {
    const log = "== Prod (AWS): apply ==\nProd (AWS): Error: nothing was built: This needs a real AWS account.\nError: second\n";
    expect(runResult(log, "failed")).toEqual({ tone: "error", text: "Prod (AWS): nothing was built: This needs a real AWS account." });
  });

  it("says nothing while a run is active, or when the log has no summary", () => {
    expect(runResult("Apply complete! Resources: 1 added, 0 changed, 0 destroyed.", "running")).toBeNull();
    expect(runResult("Initializing the backend...", "completed")).toBeNull();
    expect(runResult("no error here", "failed")).toBeNull();
  });
});

// B's lines after an apply: its count, then one "Built:" line per block the
// deployment holds.
const built = (changed: number, ids: string[]) =>
  [`Apply complete: ${changed} ${changed === 1 ? "block" : "blocks"} built or changed.`, ...ids.map((id) => `Built: ${id} vm ${id}`)].join("\n");

describe("runHeadline", () => {
  it("counts the blocks from the run's own Built lines, with the cloud resources behind them", () => {
    const four = built(4, ["a", "b", "c", "d"]);
    expect(runHeadline(`Apply complete! Resources: 18 added, 0 changed, 0 destroyed.\n${four}`, "completed", "apply")).toEqual({
      tone: "ok",
      text: "Built 4 blocks (18 cloud resources)",
    });
    expect(runHeadline(`Apply complete! Resources: 1 added, 2 changed, 1 destroyed.\n${built(1, ["a"])}`, "completed", "apply")?.text).toBe(
      "Built 1 block (1 cloud resource added, 2 changed, 1 removed)",
    );
    expect(
      runHeadline(`Apply complete! Resources: 0 added, 0 changed, 0 destroyed.\n${built(0, ["a", "b", "c"]).replace("0 blocks built or changed", "everything is up to date")}`, "completed", "apply")
        ?.text,
    ).toBe("Built 3 blocks. Nothing needed to change.");
  });

  it("says what B changed when no Terraform ran (the mock)", () => {
    expect(runHeadline(built(3, ["a", "b", "c"]), "completed", "apply")?.text).toBe("Built 3 blocks");
    expect(runHeadline(built(1, ["a", "b", "c"]), "completed", "apply")?.text).toBe("Built 3 blocks (1 new or changed)");
    expect(runHeadline("Apply complete: everything is up to date.\nBuilt: a vm a", "completed", "apply")?.text).toBe("Built 1 block. Nothing needed to change.");
  });

  it("falls back to the run's summary for a deploy from before B named its blocks", () => {
    expect(runHeadline("Apply complete! Resources: 2 added, 0 changed, 0 destroyed.", "completed", "apply")?.text).toBe("Apply complete: 2 added");
  });

  it("adds up each cloud of a split deploy and counts each block once", () => {
    const log = [
      "== lab (OpenStack): apply ==",
      "Apply complete! Resources: 12 added, 0 changed, 0 destroyed.",
      "lab (OpenStack): apply complete, 3 blocks built or changed.",
      "Built: a vm a",
      "Built: b network b",
      "Built: c volume c",
      "== Local AWS (Amazon Web Services): apply ==",
      "Apply complete! Resources: 7 added, 0 changed, 0 destroyed.",
      "Local AWS (Amazon Web Services): apply complete, 1 block built or changed.",
      "Built: d bucket d",
    ].join("\n");
    expect(runHeadline(log, "completed", "apply")?.text).toBe("Built 4 blocks (19 cloud resources)");
  });

  it("leads a destroy with what it removed", () => {
    expect(runHeadline("Destroyed web (VM).\nDestroy complete: 3 blocks removed.\n", "completed", "destroy")?.text).toBe("Destroyed 3 blocks");
    expect(runHeadline("Destroy complete! Resources: 5 destroyed.\nDestroy complete.\n", "completed", "destroy")?.text).toBe("Destroyed 5 cloud resources");
    expect(runHeadline("Destroy complete! Resources: 0 destroyed.\nDestroy complete.\n", "completed", "destroy")?.text).toBe("Nothing was left to destroy.");
    expect(runHeadline("Destroy complete.\n", "completed", "destroy")?.text).toBe("Destroyed everything this deployment built.");
  });

  it("leads a failed run with its error in plain words, and says nothing while a run is active", () => {
    expect(runHeadline("Deploy on Mock cloud (lab), started 10:42:03 UTC.\nError: boom\n", "failed", "apply")).toEqual({ tone: "error", text: "boom" });
    expect(runHeadline("", "running", "apply")).toBeNull();
  });
});

describe("failureAdvice", () => {
  const boxed = (line: string) => ["╷", `│ Error: ${line}`, "│ ", "│   with x.y,", "╵"].join("\n");

  it("explains each known cloud error in one plain sentence", () => {
    const cases: [string, RegExp][] = [
      [boxed("Error waiting for instance (8c2f) to become ready: unexpected state 'ERROR', wanted target 'ACTIVE'. last error: %!s(<nil>)"), /could not start the server/],
      [boxed("creating server: No valid host was found. There are not enough hosts available."), /could not start the server/],
      [boxed("Error extending volume: Bad request: resizing online is disabled"), /cannot grow a disk while it is attached/],
      [boxed("updating security_groups: Duplicate items in the list: 'web-fw'"), /refused a firewall change/],
      [boxed("creating EC2 Instance: operation error EC2: RunInstances, api error UnauthorizedOperation: You are not authorized"), /Update the account link's stack/],
      [boxed("creating S3 Bucket: AccessDenied: Access Denied"), /Update the account link's stack/],
      [boxed("creating server: Quota exceeded for cores: Requested 2, but already used 20 of 20 cores"), /reached a limit/],
      [boxed("creating EC2 VPC: VpcLimitExceeded: The maximum number of VPCs has been reached."), /reached a limit/],
      [boxed("Error waiting for network (1) to become deleted: timeout while waiting for state to become 'DELETED'"), /took too long to remove/],
    ];
    for (const [log, advice] of cases) expect(failureAdvice(log)).toMatch(advice);
  });

  it("says nothing for an error it does not know, or a creating timeout", () => {
    expect(failureAdvice(boxed("creating server: boom"))).toBeNull();
    expect(failureAdvice(boxed("timeout while waiting for state to become 'ACTIVE'"))).toBeNull();
    // A block named after a quota is not a quota error.
    expect(failureAdvice("openstack_compute_instance_v2.quota-svc: Creating...\nError: boom")).toBeNull();
    // A volume's lower-case error state is not a server that failed to start.
    expect(failureAdvice(boxed("volume to become ready: unexpected state 'error', wanted target 'available'"))).toBeNull();
  });
});

describe("a plan's replacements", () => {
  const plan = (replaced: number, destroyed: number, total: number) =>
    [
      ...Array.from({ length: replaced }, (_, i) => `  # openstack_networking_secgroup_rule_v2.r${i} must be replaced`),
      ...Array.from({ length: destroyed }, (_, i) => `  # openstack_compute_instance_v2.old${i} will be destroyed`),
      `Plan: 5 to add, 4 to change, ${total} to destroy.`,
    ].join("\n");

  it("says how many destroys are replacements", () => {
    expect(runResult(plan(4, 0, 4), "completed")?.text).toBe("Plan: 5 to add, 4 to change, 4 to destroy (all 4 are replaced: removed and built again)");
    expect(runResult(plan(1, 1, 2), "completed")?.text).toBe("Plan: 5 to add, 4 to change, 2 to destroy (1 of them is replaced: removed and built again)");
    expect(runResult(plan(1, 0, 1), "completed")?.text).toBe("Plan: 5 to add, 4 to change, 1 to destroy (it is replaced: removed and built again)");
    expect(runResult("  # aws_instance.a is tainted, so must be replaced\nPlan: 1 to add, 0 to change, 1 to destroy.", "completed")?.text).toContain("it is replaced");
  });

  it("adds nothing when no destroy is a replacement, or the log does not say", () => {
    expect(runResult(plan(0, 2, 2), "completed")?.text).toBe("Plan: 5 to add, 4 to change, 2 to destroy");
    expect(runResult("Plan: 1 to add, 0 to change, 3 to destroy.", "completed")?.text).toBe("Plan: 1 to add, 3 to destroy");
  });
});

describe("failureLines", () => {
  it("reads B's errors, Terraform's boxed errors and B's problems, once each, at most three", () => {
    const log = [
      "Plan on OpenStack (lab), started 10:42:03 UTC.",
      "╷",
      "│ Error: creating server: quota exceeded",
      "│ ",
      "│   with openstack_compute_instance_v2.web,",
      "╵",
      "Error: creating server: quota exceeded",
      "Lab (OpenStack): Problem: web: This cloud has no image named debian-12.",
      "Problem: db: A volume needs a size.",
      "Problem: one too many",
      "Run 1 ended at 10:42:09 UTC.",
    ].join("\n");
    expect(failureLines(log)).toEqual(["creating server: quota exceeded", "Lab (OpenStack): web: This cloud has no image named debian-12.", "db: A volume needs a size."]);
    expect(failureLines("Plan: 1 to add, 0 to change, 0 to destroy.")).toEqual([]);
  });
});
