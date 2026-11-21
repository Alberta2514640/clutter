import { describe, expect, it } from "vitest";
import { timeAgo } from "./timeAgo";

const now = new Date("2026-11-20T12:00:00Z");
const ago = (ms: number) => new Date(now.getTime() - ms).toISOString();

describe("timeAgo", () => {
  it("words recent times", () => {
    expect(timeAgo(ago(20_000), now)).toBe("just now");
    expect(timeAgo(ago(60_000), now)).toBe("1 minute ago");
    expect(timeAgo(ago(5 * 60_000), now)).toBe("5 minutes ago");
    expect(timeAgo(ago(3 * 3_600_000), now)).toBe("3 hours ago");
    expect(timeAgo(ago(2 * 86_400_000), now)).toBe("2 days ago");
  });

  it("gives the date after a week", () => {
    expect(timeAgo("2026-11-02T12:00:00Z", now)).toBe("Nov 2, 2026");
  });

  it("is null for a missing or unreadable time", () => {
    expect(timeAgo(null, now)).toBeNull();
    expect(timeAgo(undefined, now)).toBeNull();
    expect(timeAgo("not a date", now)).toBeNull();
  });
});
