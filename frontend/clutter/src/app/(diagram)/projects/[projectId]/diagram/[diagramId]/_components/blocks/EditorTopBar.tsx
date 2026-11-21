"use client";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DIAGRAM_NAME_MAX } from "@/lib/features/diagram/diagramName";
import type { SaveStatus } from "@/lib/features/diagram/saveStatus";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import { useCurrentOrganizationStore } from "@/lib/features/organization/uiStore";
import { cn } from "@/lib/utils";
import { AlertTriangle, ArrowLeft, Check, Code2, Loader2, Pencil, Rocket, Save } from "lucide-react";
import Link from "next/link";

type Props = {
  organizationId: string | null;
  organizationName: string | null;
  projectId: string;
  projectName: string | null;
  name: string;
  onNameChange: (name: string) => void;
  status: SaveStatus;
  dirty: boolean;
  saving: boolean;
  // Why Save is off besides "nothing to save", or null.
  saveBlocked: string | null;
  // The blocks with a problem that holds Save back; clicking the count
  // selects the first.
  problemCount: number;
  onShowProblem: () => void;
  // A failed save's plain sentence, or null.
  saveError: string | null;
  onSave: () => void;
  onBack: () => void;
  codeOpen: boolean;
  onToggleCode: () => void;
  onDeploy: () => void;
};

// The editor's top bar: the way back, where the diagram sits (team, project,
// its editable name), whether it is saved, and the three things to do next:
// Save, Preview the code, Deploy. One of them is the primary button: Save
// while there are changes, else Deploy.
export default function EditorTopBar({
  organizationId,
  organizationName,
  projectId,
  projectName,
  name,
  onNameChange,
  status,
  dirty,
  saving,
  saveBlocked,
  problemCount,
  onShowProblem,
  saveError,
  onSave,
  onBack,
  codeOpen,
  onToggleCode,
  onDeploy,
}: Props) {
  const selectOrganization = useCurrentOrganizationStore((s) => s.select);
  const canSave = dirty && !saving && !saveBlocked;
  // py-0.5 and leading-5 make each link a 24px target.
  const crumb =
    "text-muted-foreground hover:text-foreground inline-block max-w-40 truncate rounded-sm py-0.5 leading-5 transition-colors outline-none focus-visible:ring-ring/50 focus-visible:ring-[3px]";

  return (
    <header className="bg-background/95 relative z-20 flex h-14 shrink-0 items-center gap-3 border-b px-3">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon-sm" onClick={onBack} aria-label="Back to the project">
            <ArrowLeft aria-hidden />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Back to the project</TooltipContent>
      </Tooltip>

      <nav aria-label="Breadcrumb" className="flex min-w-0 items-center">
        <ol className="flex min-w-0 items-center gap-1.5 text-sm">
          <li className="hidden min-w-0 xl:flex">
            <Link href={PROJECTS_PATH} onClick={() => organizationId && selectOrganization(organizationId)} className={crumb}>
              {organizationName ?? "Team"}
            </Link>
          </li>
          <li aria-hidden className="text-muted-foreground/60 hidden xl:block">
            /
          </li>
          <li className="flex min-w-0">
            <Link href={`/projects/${projectId}/diagrams`} className={crumb}>
              {projectName ?? "Project"}
            </Link>
          </li>
          <li aria-hidden className="text-muted-foreground/60">
            /
          </li>
          <li className="min-w-0">
            {/* The name looks editable: a pencil on hover and focus, a ring while typing. */}
            <label className="group relative flex items-center" title="Rename the diagram">
              <input
                aria-label="Diagram name"
                aria-current="page"
                value={name}
                maxLength={DIAGRAM_NAME_MAX}
                onChange={(e) => onNameChange(e.target.value)}
                className="text-foreground hover:border-border focus:border-ring focus:ring-ring/30 w-56 rounded-md border border-transparent bg-transparent py-1 pr-7 pl-2 text-sm font-semibold outline-none focus:ring-[3px]"
              />
              <Pencil
                aria-hidden
                className="text-muted-foreground pointer-events-none absolute right-2 size-3.5 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100"
              />
            </label>
          </li>
        </ol>
      </nav>

      <div className="ml-auto flex shrink-0 items-center gap-2">
        {saveError && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span role="alert" className="text-danger inline-flex items-center gap-1 text-xs font-medium">
                <AlertTriangle className="size-3.5" aria-hidden />
                Not saved
                <span className="sr-only">: {saveError}</span>
              </span>
            </TooltipTrigger>
            <TooltipContent>{saveError}</TooltipContent>
          </Tooltip>
        )}
        {dirty && problemCount > 0 && (
          <Button
            variant="ghost"
            size="xs"
            onClick={onShowProblem}
            className="text-warning hover:text-warning"
            title="Show the first block to fix before saving">
            <AlertTriangle aria-hidden />
            {problemCount === 1 ? "1 block to fix" : `${problemCount} blocks to fix`}
          </Button>
        )}
        <span
          role="status"
          data-testid="save-status"
          data-tone={status.tone}
          className={cn(
            "flex min-w-[7.5rem] items-center justify-end gap-1.5 text-xs whitespace-nowrap",
            status.tone === "saved" ? "text-muted-foreground" : status.tone === "unsaved" ? "text-warning" : "text-muted-foreground",
          )}>
          {/* The key replays the fade, so the status cross-fades as it changes. */}
          <span key={status.text} className="inline-flex items-center gap-1.5 animate-fade-in">
            {status.tone === "saved" ? (
              <Check className="text-success size-3.5" aria-hidden />
            ) : status.tone === "busy" ? (
              <Loader2 className="size-3.5 animate-spin" aria-hidden />
            ) : (
              <span aria-hidden className="bg-warning size-1.5 rounded-full" />
            )}
            {status.text}
          </span>
        </span>
        <Button
          type="button"
          size="sm"
          variant={dirty ? "primary" : "outline"}
          onClick={onSave}
          disabled={!canSave}
          title={dirty && saveBlocked ? saveBlocked : undefined}>
          {saving ? <Loader2 className="animate-spin" aria-hidden /> : <Save aria-hidden />}
          Save
        </Button>
        <Button type="button" size="sm" variant="outline" aria-pressed={codeOpen} onClick={onToggleCode} className={cn(codeOpen && "bg-accent")}>
          <Code2 aria-hidden />
          Preview
        </Button>
        <Button type="button" size="sm" variant={dirty ? "outline" : "primary"} onClick={onDeploy}>
          <Rocket aria-hidden />
          Deploy
        </Button>
      </div>
    </header>
  );
}
