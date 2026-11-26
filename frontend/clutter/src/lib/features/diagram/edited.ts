// editedLabel words when a diagram was last saved, for its card: "Edited just
// now", "Edited 5 minutes ago", ..., then the date ("Edited Nov 2, 2026").
// The wording of the time is timeAgo's; null when the time is missing or
// unreadable, so the card shows nothing rather than a wrong time.

// Relative import: Vitest runs this file without the "@/" alias.
import { timeAgo } from "../projects/timeAgo";

export function editedLabel(iso: string | null | undefined, now: Date): string | null {
  const ago = timeAgo(iso, now);
  return ago ? `Edited ${ago}` : null;
}
