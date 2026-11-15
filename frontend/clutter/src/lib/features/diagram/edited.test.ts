import { describe, expect, it } from "vitest";
import { editedLabel } from "./edited";

const now = new Date("2026-11-14T12:00:00Z");
const ago = (ms: number) => new Date(now.getTime() - ms).toISOString();
const MIN = 60_000;

describe("editedLabel", () => {
  it("says nothing without a readable time", () => {
    expect(editedLabel(undefined, now)).toBeNull();
    expect(editedLabel("", now)).toBeNull();
    expect(editedLabel("not a date", now)).toBeNull();
  });

  it("words recent saves in minutes, hours and days", () => {
    expect(editedLabel(ago(20_000), now)).toBe("Edited just now");
    expect(editedLabel(ago(1 * MIN), now)).toBe("Edited 1 minute ago");
    expect(editedLabel(ago(45 * MIN), now)).toBe("Edited 45 minutes ago");
    expect(editedLabel(ago(3 * 60 * MIN), now)).toBe("Edited 3 hours ago");
    expect(editedLabel(ago(2 * 24 * 60 * MIN), now)).toBe("Edited 2 days ago");
  });

  it("gives the date after a week, and treats a clock ahead of ours as just now", () => {
    expect(editedLabel("2026-11-02T12:00:00Z", now)).toBe("Edited Nov 2, 2026");
    expect(editedLabel(new Date(now.getTime() + 5 * MIN).toISOString(), now)).toBe("Edited just now");
  });
});
