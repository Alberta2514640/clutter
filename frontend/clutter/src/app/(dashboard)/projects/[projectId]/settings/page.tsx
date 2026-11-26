"use client";

import ConfirmDialog from "@/components/common/ConfirmDialog";
import ErrorState from "@/components/common/ErrorState";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/sonner";
import { Textarea } from "@/components/ui/textarea";
import { useOrgDeployments } from "@/lib/features/deployments/orgHooks";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import { deleteCheck } from "@/lib/features/projects/deploymentSummary";
import { useDeleteProject, useProjects, useUpdateProject } from "@/lib/features/projects/hooks";
import { PROJECT_DESCRIPTION_MAX, projectDeleteGoes, projectDeleteLine, projectDescriptionProblem, projectNameProblem } from "@/lib/features/projects/rules";
import type { Project } from "@/lib/features/projects/types";
import { isConflict } from "@/lib/features/user/session";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { useProjectContext } from "../_contexts/ProjectContext";

// The project's name and description, and deleting it. The form is keyed by
// the project's saved values, so a save or another tab's change starts it again.
function GeneralForm({ project, token, takenNames }: { project: Project; token: string | null; takenNames: string[] }) {
  const updateM = useUpdateProject(token);
  const savedName = project.name;
  const savedDescription = project.description ?? "";
  const [name, setName] = useState(savedName);
  const [description, setDescription] = useState(savedDescription);
  const [conflict, setConflict] = useState(false);

  const changed = name.trim() !== savedName || description.trim() !== savedDescription.trim();
  const nameProblem = projectNameProblem(name, takenNames) ?? (conflict ? "Another project in this team has that name. Choose another name." : null);
  const descriptionProblem = projectDescriptionProblem(description);
  const canSave = changed && !nameProblem && !descriptionProblem && !updateM.isPending;

  const save = async (event: FormEvent) => {
    event.preventDefault();
    if (!canSave) return;
    setConflict(false);
    const trimmedDescription = description.trim();
    try {
      await updateM.mutateAsync({
        projectId: project.id,
        data: {
          organizationId: project.organizationId,
          name: name.trim(),
          // A's update writes both fields; an empty description clears it.
          description: trimmedDescription,
        },
      });
      toast.success("Changes saved");
    } catch (err) {
      if (isConflict(err)) setConflict(true);
      else toast.error("Clutter could not save your changes. Try again.");
    }
  };

  return (
    <form onSubmit={save} className="surface-card" noValidate>
      <div className="space-y-6 p-5 md:p-6">
        <div>
          <h2 className="text-foreground text-base font-semibold">General</h2>
          <p className="text-muted-foreground mt-1 text-sm">The name and description everyone on the team sees.</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="project-settings-name">Name</Label>
          <Input
            id="project-settings-name"
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              setConflict(false);
            }}
            disabled={updateM.isPending}
            aria-invalid={!!nameProblem}
            aria-describedby="project-settings-name-hint"
            className="max-w-md"
          />
          <p id="project-settings-name-hint" className={cn("text-sm", nameProblem ? "text-danger" : "text-muted-foreground")}>
            {nameProblem ?? "Up to 32 characters, different from the team's other projects."}
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="project-settings-description">Description</Label>
          <Textarea
            id="project-settings-description"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={updateM.isPending}
            aria-invalid={!!descriptionProblem}
            aria-describedby="project-settings-description-hint"
            placeholder="What is this project for?"
          />
          <p id="project-settings-description-hint" className={cn("text-sm", descriptionProblem ? "text-danger" : "text-muted-foreground")}>
            {descriptionProblem ?? `Optional. Up to ${PROJECT_DESCRIPTION_MAX} characters.`}
          </p>
        </div>
      </div>

      <div className="bg-muted/40 flex flex-wrap items-center justify-end gap-3 rounded-b-xl border-t px-5 py-3 md:px-6">
        {changed && (
          <Button
            type="button"
            variant="ghost"
            disabled={updateM.isPending}
            onClick={() => {
              setName(savedName);
              setDescription(savedDescription);
              setConflict(false);
            }}>
            Discard changes
          </Button>
        )}
        <Button type="submit" disabled={!canSave}>
          {updateM.isPending && <Loader2 className="animate-spin" aria-hidden="true" />}
          {updateM.isPending ? "Saving…" : "Save changes"}
        </Button>
      </div>
    </form>
  );
}

function DangerZone({ project, token, organizationId }: { project: Project; token: string | null; organizationId: string }) {
  const router = useRouter();
  const deleteM = useDeleteProject(token);
  const deploymentsQ = useOrgDeployments(token, organizationId, project.id);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const check = deleteCheck(deploymentsQ);
  const holding = check.holding;
  const blocked = check.state !== "clear";
  const diagrams = project.diagrams?.length ?? 0;

  const confirm = async () => {
    setError(null);
    try {
      await deleteM.mutateAsync({ projectId: project.id, organizationId: project.organizationId });
    } catch (err) {
      setError(
        isConflict(err)
          ? "A diagram in this project still has deployments that may hold resources. Destroy them first."
          : "Clutter could not delete the project. Try again.",
      );
      return;
    }
    setOpen(false);
    toast.success(`Deleted "${project.name}"`);
    router.push(PROJECTS_PATH);
  };

  return (
    <section aria-labelledby="danger-heading" className="surface-card [--card-border:var(--card-border-danger)]">
      <div className="flex flex-col gap-4 p-5 md:flex-row md:items-center md:justify-between md:p-6">
        <div className="min-w-0">
          <h2 id="danger-heading" className="text-danger text-base font-semibold">
            Danger zone
          </h2>
          <p className="text-muted-foreground mt-1 text-sm">{projectDeleteLine(diagrams)}</p>
          {check.state === "checking" && (
            <p role="status" className="text-muted-foreground mt-2 flex items-center gap-2 text-sm">
              <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
              Checking this project&apos;s deployments…
            </p>
          )}
          {check.state === "failed" && (
            <ErrorState
              variant="inline"
              className="mt-2"
              description="Clutter could not check this project's deployments, so Delete is off."
              onRetry={() => void deploymentsQ.refetch()}
              retrying={deploymentsQ.isFetching}
            />
          )}
          {check.state === "holding" && (
            <p className="text-foreground mt-2 text-sm">
              {holding === 1 ? "One deployment may" : `${holding} deployments may`} still hold resources in your cloud. Destroy{" "}
              {holding === 1 ? "it" : "them"} first.{" "}
              <Link href={`/projects/${project.id}/deployments`} className="text-link font-medium underline-offset-4 hover:underline">
                Go to Deployments
              </Link>
            </p>
          )}
        </div>
        <Button
          type="button"
          variant="destructive"
          className="md:shrink-0"
          disabled={blocked || deleteM.isPending}
          onClick={() => {
            setError(null);
            setOpen(true);
          }}>
          Delete project
        </Button>
      </div>

      {open && (
        <ConfirmDialog
          title="Delete this project?"
          confirmLabel="Delete project"
          pendingLabel="Deleting…"
          isPending={deleteM.isPending}
          error={error}
          onCancel={() => setOpen(false)}
          onConfirm={() => void confirm()}>
          <p>
            <span className="text-foreground font-medium">{project.name}</span>
            {projectDeleteGoes(diagrams)} {diagrams === 0 ? "is" : "are"} deleted for good. This cannot be undone.
          </p>
        </ConfirmDialog>
      )}
    </section>
  );
}

export default function ProjectSettingsPage() {
  const { token, project, organizationId } = useProjectContext();
  const projectsQ = useProjects(token, organizationId);
  if (!project) return null;

  const takenNames = (projectsQ.data ?? []).filter((p) => p.id !== project.id).map((p) => p.name);

  return (
    <div className="max-w-3xl space-y-8">
      <GeneralForm key={`${project.id}:${project.name}:${project.description ?? ""}`} project={project} token={token} takenNames={takenNames} />
      <DangerZone project={project} token={token} organizationId={organizationId} />
    </div>
  );
}
