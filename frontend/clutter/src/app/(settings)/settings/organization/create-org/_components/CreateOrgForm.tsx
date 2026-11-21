"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  organizationDescriptionHint,
  organizationDescriptionProblem,
  organizationNameHint,
  organizationNamePlaceholder,
  organizationNameProblem,
  organizationNameTaken,
} from "@/lib/features/organization/rules";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";
import { useRef, useState, type FormEvent } from "react";

// The fields A takes for a new organization (POST /organization).
export interface CreateOrgFormValues {
  name: string;
  description: string;
}

type Props = {
  onSubmit: (values: CreateOrgFormValues) => void | Promise<void>;
  onCancel: () => void;
  // Any change to a field clears the last refusal.
  onEdit: () => void;
  isCreating: boolean;
  // A refused the name as one the person already uses (409).
  conflict: boolean;
  failed: boolean;
};

export default function CreateOrgForm({ onSubmit, onCancel, onEdit, isCreating, conflict, failed }: Props) {
  const nameRef = useRef<HTMLInputElement>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [showProblems, setShowProblems] = useState(false);

  const nameProblem = organizationNameProblem(name);
  const descriptionProblem = organizationDescriptionProblem(description);
  const nameError = (showProblems ? nameProblem : null) ?? (conflict ? organizationNameTaken : null);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (nameProblem || descriptionProblem) {
      setShowProblems(true);
      nameRef.current?.focus();
      return;
    }
    await onSubmit({ name: name.trim(), description: description.trim() });
  };

  return (
    <form onSubmit={submit} noValidate className="surface-card">
      <div className="space-y-6 p-5 md:p-6">
        <div className="space-y-2">
          <Label htmlFor="new-team-name">Team name</Label>
          <Input
            ref={nameRef}
            id="new-team-name"
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              onEdit();
            }}
            onBlur={() => name && setShowProblems(true)}
            placeholder={organizationNamePlaceholder}
            autoFocus
            autoComplete="organization"
            aria-invalid={!!nameError}
            aria-describedby="new-team-name-hint"
            className="max-w-md"
          />
          <p id="new-team-name-hint" className={cn("text-sm", nameError ? "text-danger" : "text-muted-foreground")}>
            {nameError ?? `${organizationNameHint} You can rename it later.`}
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="new-team-description">Description</Label>
          <Textarea
            id="new-team-description"
            rows={3}
            placeholder="What is this team for?"
            value={description}
            onChange={(e) => {
              setDescription(e.target.value);
              onEdit();
            }}
            aria-invalid={!!descriptionProblem}
            aria-describedby="new-team-description-hint"
          />
          <p id="new-team-description-hint" className={cn("text-sm", descriptionProblem ? "text-danger" : "text-muted-foreground")}>
            {descriptionProblem ?? organizationDescriptionHint}
          </p>
        </div>

        {failed && (
          <p role="alert" className="border-danger/25 bg-danger/10 text-foreground rounded-lg border p-3 text-sm">
            Clutter could not create the team. Try again.
          </p>
        )}
      </div>

      <div className="bg-muted/40 flex flex-col-reverse gap-3 rounded-b-xl border-t px-5 py-3 sm:flex-row sm:justify-end md:px-6">
        <Button type="button" variant="ghost" onClick={onCancel} disabled={isCreating}>
          Cancel
        </Button>
        <Button type="submit" disabled={isCreating}>
          {isCreating && <Loader2 className="animate-spin" aria-hidden="true" />}
          {isCreating ? "Creating…" : "Create team"}
        </Button>
      </div>
    </form>
  );
}
