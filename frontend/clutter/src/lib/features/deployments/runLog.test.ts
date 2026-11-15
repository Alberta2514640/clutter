import { describe, expect, it } from "vitest";
import { runResult } from "./runLog";

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
      "Apply complete: 6 blocks built.",
    ].join("\n");
    expect(runResult(log, "completed")?.text).toBe("Apply complete: 19 added");
  });

  it("drops the plan Terraform prints inside a destroy, and an apply with nothing to do", () => {
    expect(runResult("Plan: 0 to add, 0 to change, 4 to destroy.\nDestroy complete! Resources: 4 destroyed.\nDestroy complete.\n", "completed")?.text).toBe(
      "Destroy complete: 4 destroyed",
    );
    expect(
      runResult("No changes. Your infrastructure matches the configuration.\nApply complete! Resources: 0 added, 0 changed, 0 destroyed.\nApply complete: 3 blocks built.\n", "completed")?.text,
    ).toBe("Apply complete: no changes");
  });

  it("reads B's block count when no Terraform ran", () => {
    expect(runResult("Created mock-1 (vm).\nApply complete: 1 block built.\n", "completed")?.text).toBe("Apply complete: 1 block built");
  });

  it("gives each cloud of a split run its own line", () => {
    const log = [
      "== Lab (Mock cloud): apply ==",
      "Lab (Mock cloud): apply complete, 4 blocks built.",
      "== Prod (AWS): apply ==",
      "Plan: 2 to add, 0 to change, 0 to destroy.",
      "Apply complete! Resources: 2 added, 0 changed, 0 destroyed.",
      "Prod (AWS): apply complete, 1 block built.",
    ].join("\n");
    expect(runResult(log, "completed")?.text).toBe("Lab (Mock cloud): 4 blocks built; Prod (AWS): 1 block built");
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
