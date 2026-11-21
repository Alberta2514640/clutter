// timeAgo words how long ago something happened: "just now", "5 minutes
// ago", "3 hours ago", "2 days ago", then the date ("Nov 2, 2026"). null when
// the time is missing or unreadable, so a page shows nothing rather than a
// wrong time.

// Relative import: Vitest runs this file without the "@/" alias.
import { plural } from "../../text";

export function timeAgo(iso: string | null | undefined, now: Date): string | null {
  if (!iso) return null;
  const then = new Date(iso);
  if (Number.isNaN(then.getTime())) return null;
  const minutes = Math.floor((now.getTime() - then.getTime()) / 60_000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${plural(minutes, "minute", "minutes")} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${plural(hours, "hour", "hours")} ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${plural(days, "day", "days")} ago`;
  return then.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}
