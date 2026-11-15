// The choices the Add connection form makes for an account linked outside B
// (AWS, through A's CloudFormation flow), kept apart from the form so they can
// be tested.

import type { LinkedAccount } from "../catalog/types";

// startRegion is the region a new connection starts on: the region the cloud's
// link flow opens in (preferred), when the cloud offers it, else its first.
export function startRegion(regions: string[], preferred?: string): string {
  if (preferred && regions.includes(preferred)) return preferred;
  return regions[0] ?? "";
}

// chosenLinkedAccount is the linked account the form uses: the one the user
// picked, else the one just linked inline, once B lists it by name.
export function chosenLinkedAccount(accounts: LinkedAccount[], picked: LinkedAccount | null, justLinked: string | null): LinkedAccount | null {
  if (picked) return picked;
  if (!justLinked) return null;
  return accounts.find((a) => a.name === justLinked) ?? null;
}
