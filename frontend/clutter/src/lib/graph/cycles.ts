export function wouldCreateCycle(edges: { from: string; to: string }[], candidate: { from: string; to: string }) {
  const adj = new Map<string, string[]>();
  for (const e of edges) adj.set(e.from, [...(adj.get(e.from) || []), e.to]);
  const seen = new Set<string>();
  const stack = [candidate.to];
  while (stack.length) {
    const n = stack.pop()!;
    if (n === candidate.from) return true;
    if (seen.has(n)) continue;
    seen.add(n);
    (adj.get(n) || []).forEach((v) => stack.push(v));
  }
  return false;
}
