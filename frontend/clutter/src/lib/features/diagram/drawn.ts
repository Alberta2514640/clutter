// Whether a team has drawn anything yet: at least one of its diagrams holds
// a block. An empty diagram, or a deployment of one, is not a drawn setup.
// The "Getting started" checklist's "Draw your setup" step reads it.

type DiagramLike = { data?: { nodes?: readonly unknown[] | null } | null };

export function hasDrawnDiagram(lists: readonly (readonly DiagramLike[] | null | undefined)[]): boolean {
  return lists.some((list) => (list ?? []).some((d) => (d.data?.nodes?.length ?? 0) > 0));
}
