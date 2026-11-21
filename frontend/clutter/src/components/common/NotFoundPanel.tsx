import AssembleInView from "@/components/brand/AssembleInView";
import IsoStack from "@/components/brand/IsoStack";
import { Button } from "@/components/ui/button";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import { cn } from "@/lib/utils";
import Link from "next/link";
import type { ReactNode } from "react";
import { DotGrid, SelectionBox } from "./CanvasMotif";

type Props = {
  title?: string;
  description?: ReactNode;
  // Ways back. The default goes to Projects.
  actions?: ReactNode;
  className?: string;
};

// NotFoundPanel is the way back from an address that matches nothing: the
// 404 page, a project id that is not the viewer's, a deleted diagram.
export default function NotFoundPanel({
  title = "Page not found",
  description = "This address does not match any page. Check the link, or go back to your projects.",
  actions,
  className,
}: Props) {
  return (
    <div className={cn("relative isolate flex flex-col items-center px-4 py-16 text-center animate-fade-in", className)}>
      <DotGrid className="-z-10" />
      {/* A knocked-over stack, selected on the canvas, with the code beside the cursor. The fallen block still rocks. */}
      <AssembleInView className="flex">
        <SelectionBox cursorLabel="404" className="px-5 py-4">
          <IsoStack preset="toppled" accent="vm" size={40} assemble idle="rock" />
        </SelectionBox>
      </AssembleInView>
      <h1 className="text-foreground mt-10 text-2xl font-semibold tracking-tight">{title}</h1>
      <p className="text-muted-foreground mt-2 max-w-md text-sm">{description}</p>
      <div className="mt-8 flex flex-col gap-3 sm:flex-row">
        {actions ?? (
          <Button asChild>
            <Link href={PROJECTS_PATH}>Go to Projects</Link>
          </Button>
        )}
      </div>
    </div>
  );
}
