// startRegion is the region a new connection starts on: the region the cloud's
// link flow opens in (preferred), when the cloud offers it, else its first.
export function startRegion(regions: string[], preferred?: string): string {
  if (preferred && regions.includes(preferred)) return preferred;
  return regions[0] ?? "";
}
