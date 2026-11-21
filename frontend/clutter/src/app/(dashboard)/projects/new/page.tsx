"use client";

import EmptyState from "@/components/common/EmptyState";
import ErrorState from "@/components/common/ErrorState";
import PageHeader from "@/components/common/PageHeader";
import PageSkeleton from "@/components/common/PageSkeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/sonner";
import { Textarea } from "@/components/ui/textarea";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useCreateProject, useProjects } from "@/lib/features/projects/hooks";
import { PROJECT_DESCRIPTION_MAX, projectDescriptionProblem, projectNameProblem } from "@/lib/features/projects/rules";
import { useMe } from "@/lib/features/user/hooks";
import { cn } from "@/lib/utils";
import { Loader2, Users } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useRef, useState, type FormEvent } from "react";

export default function NewProjectPage() {
  const router = useRouter();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const current = useCurrentOrganization(token);
  const organizationId = current.organizationId;
  const projectsQ = useProjects(token, organizationId);
  const createProjectM = useCreateProject(token);
  const nameRef = useRef<HTMLInputElement>(null);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [showProblems, setShowProblems] = useState(false);
  const [failed, setFailed] = useState(false);

  if (current.isError) {
    return (
      <ErrorState
        title="Your teams could not load"
        description="Check your connection, then try again."
        onRetry={() => void current.refetch()}
        retrying={current.isFetching}
      />
    );
  }
  if (current.isLoading || meQ.isLoading) return <PageSkeleton label="Loading…" cards={0} />;
  if (!current.organization || !organizationId) {
    return (
      <>
        <PageHeader title="New project" />
        <EmptyState
          icon={Users}
          title="Create a team first"
          description="Projects belong to a team. Make your team, then come back to add a project."
          action={
            <Button asChild>
              <Link href="/settings/organization/create-org">Create a team</Link>
            </Button>
          }
        />
      </>
    );
  }

  const taken = (projectsQ.data ?? []).map((p) => p.name);
  const nameProblem = projectNameProblem(name, taken);
  const descriptionProblem = projectDescriptionProblem(description);
  const shownNameProblem = showProblems ? nameProblem : null;

  const create = async (event: FormEvent) => {
    event.preventDefault();
    setFailed(false);
    if (nameProblem || descriptionProblem) {
      setShowProblems(true);
      nameRef.current?.focus();
      return;
    }
    const desc = description.trim();
    try {
      const created = await createProjectM.mutateAsync({
        organizationId,
        name: name.trim(),
        // A's create refuses an empty description, so it is left out.
        ...(desc ? { description: desc } : {}),
      });
      toast.success("Project created");
      router.replace(`/projects/${created.id}/diagrams`);
    } catch {
      setFailed(true);
    }
  };

  return (
    <div className="max-w-2xl">
      <PageHeader
        eyebrow={<Link href={PROJECTS_PATH} className="hover:text-foreground">Projects</Link>}
        title="New project"
        description={`A project holds diagrams. It belongs to ${current.organization.name}.`}
      />

      <form onSubmit={create} noValidate className="surface-card">
        <div className="space-y-6 p-5 md:p-6">
          <div className="space-y-2">
            <Label htmlFor="project-name">Name</Label>
            <Input
              ref={nameRef}
              id="project-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onBlur={() => name && setShowProblems(true)}
              placeholder="For example, Web shop"
              autoFocus
              aria-invalid={!!shownNameProblem}
              aria-describedby="project-name-hint"
            />
            <p id="project-name-hint" className={cn("text-sm", shownNameProblem ? "text-danger" : "text-muted-foreground")}>
              {shownNameProblem ?? "Up to 32 characters."}
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="project-description">Description</Label>
            <Textarea
              id="project-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What is this project for?"
              rows={3}
              aria-invalid={!!descriptionProblem}
              aria-describedby="project-description-hint"
            />
            <p id="project-description-hint" className={cn("text-sm", descriptionProblem ? "text-danger" : "text-muted-foreground")}>
              {descriptionProblem ?? `Optional. Up to ${PROJECT_DESCRIPTION_MAX} characters.`}
            </p>
          </div>

          {failed && (
            <p role="alert" className="border-danger/25 bg-danger/10 text-foreground rounded-lg border p-3 text-sm">
              Clutter could not create the project. Try again.
            </p>
          )}
        </div>

        <div className="bg-muted/40 flex flex-col-reverse gap-3 rounded-b-xl border-t px-5 py-3 sm:flex-row sm:justify-end md:px-6">
          <Button asChild variant="ghost">
            <Link href={PROJECTS_PATH}>Cancel</Link>
          </Button>
          <Button type="submit" disabled={createProjectM.isPending}>
            {createProjectM.isPending && <Loader2 className="animate-spin" aria-hidden="true" />}
            {createProjectM.isPending ? "Creating…" : "Create project"}
          </Button>
        </div>
      </form>
    </div>
  );
}
