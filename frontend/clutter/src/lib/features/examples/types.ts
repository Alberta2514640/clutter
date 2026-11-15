// Mirrors backend/adapter/internal/examples (Example, Placement) and the
// examples route (GET /v1/examples).

// examples.Levels, from easy to complex.
export type ExampleLevel = "starter" | "intermediate" | "advanced";

export const LEVEL_LABELS: Record<ExampleLevel, string> = {
  starter: "Starter",
  intermediate: "Intermediate",
  advanced: "Advanced",
};

export type ExamplePlacement = {
  connector: string;
  node_ids: string[];
};

export type Example = {
  id: string;
  name: string;
  level: ExampleLevel;
  description: string;
  // Connector ids. Without placements, each one alone builds the example;
  // with placements, the example is one deployment over all of them.
  clouds: string[];
  placements?: ExamplePlacement[];
  // A building-block document (diagram/blocksDocument.ts).
  document: unknown;
};

export type ExamplesResponse = { items: Example[] | null };
