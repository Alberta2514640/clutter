import { describe, expect, it } from "vitest";
import { runHeadline, runResult } from "./runLog";

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

  it("gives the first error of a failed run", () => {
    const log = "== Prod (AWS): apply ==\nProd (AWS): Error: nothing was built: This needs a real AWS account.\nError: second\n";
    expect(runResult(log, "failed")).toEqual({ tone: "error", text: "Error: nothing was built: This needs a real AWS account." });
  });

  it("says nothing while a run is active, or when the log has no summary", () => {
    expect(runResult("Apply complete! Resources: 1 added, 0 changed, 0 destroyed.", "running")).toBeNull();
    expect(runResult("Initializing the backend...", "completed")).toBeNull();
    expect(runResult("no error here", "failed")).toBeNull();
  });
});

describe("runHeadline", () => {
  it("leads a finished deploy with its blocks and the cloud resources behind them", () => {
    expect(runHeadline("Apply complete! Resources: 18 added, 0 changed, 0 destroyed.", "completed", "apply", 4)).toEqual({
      tone: "ok",
      text: "Built 4 blocks (18 cloud resources)",
    });
    expect(runHeadline("Apply complete! Resources: 1 added, 2 changed, 1 destroyed.", "completed", "apply", 1)?.text).toBe(
      "Built 1 block (1 cloud resource added, 2 changed, 1 removed)",
    );
    expect(runHeadline("Apply complete! Resources: 0 added, 0 changed, 0 destroyed.", "completed", "apply", 3)?.text).toBe("Built 3 blocks. Nothing needed to change.");
    expect(runHeadline("Apply complete: 3 blocks built or changed.", "completed", "apply", 3)?.text).toBe("Built 3 blocks");
  });

  it("adds up each cloud of a split deploy", () => {
    const log = [
      "== lab (OpenStack): apply ==",
      "Apply complete! Resources: 12 added, 0 changed, 0 destroyed.",
      "== Local AWS (Amazon Web Services): apply ==",
      "Apply complete! Resources: 7 added, 0 changed, 0 destroyed.",
    ].join("\n");
    expect(runHeadline(log, "completed", "apply", 4)?.text).toBe("Built 4 blocks (19 cloud resources)");
  });

  it("is runResult's line for anything but a finished deploy", () => {
    expect(runHeadline("Destroy complete! Resources: 5 destroyed.", "completed", "destroy", 4)?.text).toBe("Destroy complete: 5 destroyed");
    expect(runHeadline("Error: boom", "failed", "apply", 4)).toEqual({ tone: "error", text: "Error: boom" });
    expect(runHeadline("", "running", "apply", 4)).toBeNull();
  });
});
