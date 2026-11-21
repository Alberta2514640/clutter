// Relative import: Vitest runs this file without the "@/" alias.
import type { Example } from "../examples/types";

// orderExamplesForCloud keeps B's order but puts first the examples that run
// on cloud (the first cloud chosen in onboarding). No cloud keeps B's order.
export function orderExamplesForCloud(examples: readonly Example[], cloud: string | null | undefined): Example[] {
  if (!cloud) return [...examples];
  const fits = (e: Example) => e.clouds.includes(cloud);
  return [...examples.filter(fits), ...examples.filter((e) => !fits(e))];
}
