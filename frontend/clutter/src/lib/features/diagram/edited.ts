// editedLabel words when a diagram was last saved, for its card: "Edited just
// now", "Edited 5 minutes ago", "Edited 3 hours ago", "Edited 2 days ago",
// then the date ("Edited Nov 2, 2026"). null when the time is missing or
// unreadable, so the card shows nothing rather than a wrong time.
export function editedLabel(iso: string | null | undefined, now: Date): string | null {
  if (!iso) return null;
  const then = new Date(iso);
  if (Number.isNaN(then.getTime())) return null;
  const minutes = Math.floor((now.getTime() - then.getTime()) / 60_000);
  const plural = (n: number, unit: string) => `${n} ${unit}${n === 1 ? "" : "s"} ago`;
  if (minutes < 1) return "Edited just now";
  if (minutes < 60) return `Edited ${plural(minutes, "minute")}`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `Edited ${plural(hours, "hour")}`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `Edited ${plural(days, "day")}`;
  return `Edited ${then.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })}`;
}
