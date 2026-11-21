import { pyramid, PYRAMID_MAX, type IsoBlockSpec } from "@/components/brand/iso";
import IsoBlock from "@/components/brand/IsoBlock";
import IsoStack from "@/components/brand/IsoStack";
import CloudLogo from "@/components/common/CloudLogo";
import type { Experience } from "@/lib/features/preferences/uiStore";
import { HelpCircle } from "lucide-react";

// One small block drawing above each onboarding question, in StepFrame's
// illustration slot. All decorative; the question and its cards say it all.

const SIZE = 26;

// A stack of n blocks, outline, with the newest in the logo's colours. It
// drops in again when n changes.
function Growing({ count, accent }: { count: number; accent: boolean }) {
  const blocks = pyramid(count).map((b, i, all) => ({ ...b, accent: accent && i === all.length - 1 }));
  return <IsoStack key={count} blocks={blocks} accent="brand" size={SIZE} assemble />;
}

const EXPERIENCE_BLOCKS: Record<Experience, number> = { new: 1, console: 2, terraform: 3 };

// Step 1: one block for a beginner, two for a console user, the whole logo
// for someone who writes Terraform.
export function ExperienceArt({ value }: { value: Experience | null }) {
  return <Growing count={value ? EXPERIENCE_BLOCKS[value] : 1} accent={!!value} />;
}

// Step 2: two blocks side by side, a team.
export function TeamArt() {
  return <IsoStack preset="pair" accent="brand" size={SIZE} />;
}

// Step 3: Clutter's block beside the chosen cloud's logo, or a question mark
// before a cloud is chosen or for "Not sure yet".
export function CloudArt({ connector }: { connector: string | null }) {
  return (
    <>
      <IsoBlock variant="brand" size={SIZE} />
      <span className="bg-input mb-3.5 h-px w-6" />
      <span key={connector ?? "none"} className="surface-card animate-fade-in mb-0.5 flex size-10 items-center justify-center rounded-lg">
        {connector ? <CloudLogo connector={connector} size={22} /> : <HelpCircle className="text-muted-foreground size-5" />}
      </span>
    </>
  );
}

// kindsOf lists the block kinds of a building-block document, each once, in
// the order they first appear.
function kindsOf(document: unknown): string[] {
  const nodes = (document as { nodes?: unknown } | null)?.nodes;
  if (!Array.isArray(nodes)) return [];
  const kinds: string[] = [];
  for (const n of nodes) {
    const kind = (n as { type?: unknown } | null)?.type;
    if (typeof kind === "string" && !kinds.includes(kind)) kinds.push(kind);
  }
  return kinds;
}

// Step 4: the chosen example as a small stack, one block per kind it uses in
// that kind's colours; one outline block for Blank; the logo's outline
// before a choice.
export function StartArt({ document, blank }: { document: unknown; blank: boolean }) {
  if (blank) return <IsoBlock size={SIZE} />;
  const kinds = kindsOf(document).slice(0, PYRAMID_MAX);
  if (kinds.length === 0) return <IsoStack preset="cluster" size={SIZE} />;
  const blocks: IsoBlockSpec[] = pyramid(kinds.length).map((b, i) => ({ ...b, variant: "solid", kind: kinds[i] }));
  return <IsoStack key={kinds.join(" ")} blocks={blocks} size={SIZE} assemble />;
}
