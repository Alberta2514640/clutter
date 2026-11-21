import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { LANDING_EXAMPLES } from "./exampleList";
import { contactEmail } from "./site";

// The example files B embeds, found from this file's own place, so the test
// runs the same from frontend/clutter (the README) and in CI. Only Vitest reads
// them; the website's build never does.
const examplesDir = fileURLToPath(new URL("../../../../../backend/adapter/internal/examples/", import.meta.url));

type ExampleFile = { id: string; name: string; level: string; description: string; clouds: string[]; placements?: unknown[] };

describe("LANDING_EXAMPLES", () => {
  it("matches B's example files, in their order", () => {
    const files = readdirSync(examplesDir)
      .filter((f) => f.endsWith(".json"))
      .sort((a, b) => a.localeCompare(b, "en", { numeric: true }));
    const fromB = files.map((f) => {
      const e = JSON.parse(readFileSync(examplesDir + f, "utf8")) as ExampleFile;
      return { id: e.id, name: e.name, level: e.level, description: e.description, clouds: e.clouds, split: (e.placements?.length ?? 0) > 0 };
    });
    expect(fromB.length).toBeGreaterThan(0);
    expect(LANDING_EXAMPLES).toEqual(fromB);
  });
});

describe("contactEmail", () => {
  it("keeps a real address", () => {
    expect(contactEmail(" team@clutter.cloud ")).toBe("team@clutter.cloud");
  });
  it("treats a missing, malformed or example address as none", () => {
    expect(contactEmail(undefined)).toBeNull();
    expect(contactEmail("hello at clutter")).toBeNull();
    expect(contactEmail("hello@example.com")).toBeNull();
    expect(contactEmail("hello@clutter.test")).toBeNull();
    expect(contactEmail("a@mail.example")).toBeNull();
  });
});
