// Mirrors backend/adapter/internal/examples (Example, Placement) and the
// examples route (GET /v1/examples).

// examples.Levels, from easy to complex.
export type ExampleLevel = "starter" | "intermediate" | "advanced";

const LEVEL_LABELS: Record<ExampleLevel, string> = {
  starter: "Starter",
  intermediate: "Intermediate",
  advanced: "Advanced",
};

// One tone per level, the same on every screen (LevelBadge): green, blue and
// violet, three hues apart, each a Badge variant whose text passes 4.5:1 on
// its fill in light and dark.
export type LevelTone = "success" | "info" | "violet" | "neutral";

const LEVEL_TONES: Record<ExampleLevel, LevelTone> = {
  starter: "success",
  intermediate: "info",
  advanced: "violet",
};

// levelLook is a level's label and tone. A level B adds later shows its own
// name, in the neutral tone, until it is listed here.
export function levelLook(level: string): { label: string; tone: LevelTone } {
  const known = level as ExampleLevel;
  return { label: LEVEL_LABELS[known] ?? level, tone: LEVEL_TONES[known] ?? "neutral" };
}

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
