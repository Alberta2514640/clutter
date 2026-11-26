"use client";

import CopyButton from "@/components/common/CopyButton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/sonner";
import { Textarea } from "@/components/ui/textarea";
import { useUpdateOrganization } from "@/lib/features/organization/hooks";
import {
  organizationDescriptionHint,
  organizationDescriptionProblem,
  organizationNameHint,
  organizationNameProblem,
  organizationNameTaken,
} from "@/lib/features/organization/rules";
import type { Organization } from "@/lib/features/organization/types";
import { isConflict } from "@/lib/features/user/session";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";
import { useState, type FormEvent } from "react";

type Props = { team: Organization; token: string | null; otherNames: string[] };

// The team's name and description. A's update leaves an empty field as it
// is, so a description can be replaced but not emptied.
export default function TeamGeneralForm({ team, token, otherNames }: Props) {
  const updateM = useUpdateOrganization(token);
  const savedDescription = team.description ?? "";
  const [name, setName] = useState(team.name);
  const [description, setDescription] = useState(savedDescription);
  const [conflict, setConflict] = useState(false);

  const nameChanged = name.trim() !== team.name;
  const descriptionChanged = description.trim() !== savedDescription.trim();
  const nameProblem =
    organizationNameProblem(name) ?? (otherNames.includes(name.trim()) || conflict ? organizationNameTaken : null);
  const descriptionProblem =
    organizationDescriptionProblem(description) ??
    (descriptionChanged && !description.trim() ? "A description cannot be emptied yet. Write a new one instead." : null);
  const changed = nameChanged || descriptionChanged;
  const canSave = changed && !nameProblem && !descriptionProblem && !updateM.isPending;

  const save = async (event: FormEvent) => {
    event.preventDefault();
    if (!canSave) return;
    setConflict(false);
    try {
      await updateM.mutateAsync({
        organizationId: team.id,
        data: {
          ...(nameChanged ? { organizationName: name.trim() } : {}),
          ...(descriptionChanged ? { description: description.trim() } : {}),
        },
      });
      toast.success("Changes saved");
    } catch (err) {
      if (isConflict(err)) setConflict(true);
      else toast.error("Clutter could not save your changes. Try again.");
    }
  };

  return (
    <form onSubmit={save} noValidate className="surface-card">
      <div className="space-y-6 p-5 md:p-6">
        <div className="space-y-2">
          <Label htmlFor="team-name">Team name</Label>
          <Input
            id="team-name"
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              setConflict(false);
            }}
            disabled={updateM.isPending}
            aria-invalid={!!nameProblem}
            aria-describedby="team-name-hint"
            className="max-w-md"
          />
          <p id="team-name-hint" className={cn("text-sm", nameProblem ? "text-danger" : "text-muted-foreground")}>
            {nameProblem ?? organizationNameHint}
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="team-description">Description</Label>
          <Textarea
            id="team-description"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={updateM.isPending}
            aria-invalid={!!descriptionProblem}
            aria-describedby="team-description-hint"
            placeholder="What is this team for?"
          />
          <p id="team-description-hint" className={cn("text-sm", descriptionProblem ? "text-danger" : "text-muted-foreground")}>
            {descriptionProblem ?? organizationDescriptionHint}
          </p>
        </div>
      </div>

      <div className="bg-muted/40 flex flex-wrap items-center justify-end gap-3 rounded-b-xl border-t px-5 py-3 md:px-6">
        {/* The ID never changes; people give it when they ask for help. */}
        <p className="text-muted-foreground mr-auto flex min-w-0 items-center gap-2 text-xs" title="This ID never changes. Give it when you ask for help.">
          <span className="shrink-0">Team ID</span>
          <code className="truncate font-mono">{team.id}</code>
          <CopyButton value={team.id} label="team ID" />
        </p>
        {changed && (
          <Button
            type="button"
            variant="ghost"
            disabled={updateM.isPending}
            onClick={() => {
              setName(team.name);
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
