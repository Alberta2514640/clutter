// The name a local one-click connection takes ("Local AWS", "Local
// OpenStack"): "Local" and the cloud's short name, with the lowest free
// number from 2 when the organization already has a connection by that name
// (B refuses a second one, 409 name_taken).

// B caps a connection's name at 64 characters.
const NAME_MAX = 64;

export function localConnectionName(cloudShortName: string, taken: string[]): string {
  const base = `Local ${cloudShortName}`.slice(0, NAME_MAX).trimEnd();
  const used = new Set(taken);
  if (!used.has(base)) return base;
  for (let i = 2; ; i++) {
    const suffix = ` ${i}`;
    const name = base.slice(0, NAME_MAX - suffix.length).trimEnd() + suffix;
    if (!used.has(name)) return name;
  }
}
