import AssembleInView from "@/components/brand/AssembleInView";
import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { DotGrid, SelectionBox } from "./CanvasMotif";

type Props = {
  icon?: LucideIcon;
  // A small block drawing (an IsoStack preset in outline with one coloured
  // block, with `assemble` and `idle`) in place of the icon tile. It
  // assembles as it comes into view and pauses its idle loop off screen.
  illustration?: ReactNode;
  // What this place is, in a few words ("No projects yet").
  title: string;
  // One sentence: what it will hold, or why it is empty.
  description?: ReactNode;
  // The one thing to do next, usually a primary Button.
  action?: ReactNode;
  // h2 on a page under PageHeader's h1; h3 inside a section with its own heading.
  headingLevel?: "h2" | "h3";
  className?: string;
};

// EmptyState fills a list or panel that has nothing yet: the canvas motif
// around an icon or a block drawing, what this is, and one action.
export default function EmptyState({ icon: Icon, illustration, title, description, action, headingLevel = "h2", className }: Props) {
  const Heading = headingLevel;
  const hasArt = !!illustration || !!Icon;
  return (
    <div
      className={cn(
        "surface-card relative isolate overflow-hidden border-dashed px-6 py-12 text-center animate-fade-in",
        className,
      )}
    >
      <DotGrid fade="top" className="-z-10" />
      {illustration ? (
        <AssembleInView className="flex justify-center [--iso-paper:var(--card-surface)]">{illustration}</AssembleInView>
      ) : (
        Icon && (
          <SelectionBox className="mx-auto">
            <span className="bg-background flex size-11 items-center justify-center rounded-lg border">
              <Icon className="text-link size-5" aria-hidden="true" />
            </span>
          </SelectionBox>
        )
      )}
      <Heading className={cn("text-foreground text-base font-semibold", hasArt && "mt-5")}>{title}</Heading>
      {description && <p className="text-muted-foreground mx-auto mt-1.5 max-w-md text-sm">{description}</p>}
      {action && <div className="mt-6 flex flex-wrap items-center justify-center gap-2">{action}</div>}
    </div>
  );
}
