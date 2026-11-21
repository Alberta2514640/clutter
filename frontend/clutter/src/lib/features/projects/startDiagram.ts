import { fromBlocksDocument, toBlocksDocument } from "@/lib/features/diagram/blocksDocument";
import { BLANK_DIAGRAM_NAME, uniqueDiagramName } from "@/lib/features/diagram/diagramName";
import { useCreateDiagram } from "@/lib/features/diagram/hooks";
import type { Example } from "@/lib/features/examples/types";

export type StartDiagramInput = {
  organizationId: string;
  projectId: string;
  // The names of the project's diagrams, so the new one gets a free name.
  takenNames: readonly string[];
  // B's example to start from, or null for a blank canvas.
  example: Example | null;
  // The name the person typed; empty uses the example's name, or the blank default.
  name?: string;
};

// useStartDiagram creates a building-block diagram, blank or from one of B's
// examples, with A's create route and the editor's marker (useCreateDiagram).
// It is the one way "New diagram" and onboarding's first diagram are made.
export function useStartDiagram(token?: string | null) {
  const createM = useCreateDiagram(token);

  const start = (input: StartDiagramInput) => {
    const doc = input.example ? fromBlocksDocument(input.example.document) : null;
    const base = input.name?.trim() || input.example?.name || BLANK_DIAGRAM_NAME;
    return createM.mutateAsync({
      organizationId: input.organizationId,
      projectId: input.projectId,
      name: uniqueDiagramName(base, [...input.takenNames]),
      document: doc ? toBlocksDocument(doc.nodes, doc.edges) : undefined,
    });
  };

  return { start, isPending: createM.isPending };
}
