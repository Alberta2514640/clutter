"use client";

import ErrorState from "@/components/common/ErrorState";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/sonner";
import { useOrgDeployments } from "@/lib/features/deployments/orgHooks";
import { homePath } from "@/lib/features/organization/home";
import { useCurrentOrganization, useDeleteOrganization } from "@/lib/features/organization/hooks";
import { teamDeleteGoes } from "@/lib/features/organization/rules";
import type { Organization } from "@/lib/features/organization/types";
import { deleteCheck } from "@/lib/features/projects/deploymentSummary";
import { useMe } from "@/lib/features/user/hooks";
import { isConflict } from "@/lib/features/user/session";
import { AlertTriangle, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

// The confirmation asks for the team's name, so a team is never deleted by a
// stray click. It stays open with the refusal when the delete fails.
function DeleteTeamDialog({ team, onCancel, onDeleted }: { team: Organization; onCancel: () => void; onDeleted: () => void }) {
  const meQ = useMe();
  const deleteOrg = useDeleteOrganization(meQ.data?.token ?? null);
  const [typed, setTyped] = useState("");
  const [error, setError] = useState<string | null>(null);
  const matches = typed.trim() === team.name;

  const confirm = async () => {
    if (!matches) return;
    setError(null);
    try {
      await deleteOrg.mutateAsync(team.id);
    } catch (err) {
      setError(
        isConflict(err)
          ? "A deployment in this team may still hold resources in your cloud. Destroy it in its diagram's Deploy view first."
          : "Clutter could not delete the team. Try again.",
      );
      return;
    }
    onDeleted();
  };

  return (
    <AlertDialog
      open
      onOpenChange={(open) => {
        if (!open && !deleteOrg.isPending) onCancel();
      }}>
      <AlertDialogContent>
        <div className="flex items-start gap-4">
          <span className="bg-danger/10 text-danger flex size-10 shrink-0 items-center justify-center rounded-lg">
            <AlertTriangle className="size-5" aria-hidden="true" />
          </span>
          <div className="min-w-0 space-y-1.5">
            <AlertDialogTitle>Delete this team?</AlertDialogTitle>
            <AlertDialogDescription>
              <span className="text-foreground font-medium">{team.name}</span>
              {teamDeleteGoes(team.total_projects)} are deleted for good. This cannot be undone.
            </AlertDialogDescription>
          </div>
        </div>

        <form
          onSubmit={(event) => {
            event.preventDefault();
            void confirm();
          }}
          className="space-y-2">
          <Label htmlFor="confirm-team-name">
            Type <span className="font-mono">{team.name}</span> to confirm
          </Label>
          <Input
            id="confirm-team-name"
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            autoComplete="off"
            spellCheck={false}
            disabled={deleteOrg.isPending}
          />
        </form>

        {error && (
          <p role="alert" className="border-danger/30 bg-danger/10 text-danger rounded-lg border p-3 text-sm">
            {error}
          </p>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleteOrg.isPending}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={!matches || deleteOrg.isPending}
            onClick={(event) => {
              event.preventDefault();
              void confirm();
            }}>
            {deleteOrg.isPending && <Loader2 className="animate-spin" aria-hidden="true" />}
            {deleteOrg.isPending ? "Deleting…" : "Delete team"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

// Team settings, Danger zone: delete the team. It is off while any of the
// team's deployments may still hold resources, since A refuses then (409).
export default function TeamDangerZonePage() {
  const router = useRouter();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const current = useCurrentOrganization(token);
  const team = current.organization;
  const deploymentsQ = useOrgDeployments(token, team?.id ?? null);
  const [open, setOpen] = useState(false);
  if (!team) return null;

  const check = deleteCheck(deploymentsQ);
  const holding = check.holding;
  const blocked = check.state !== "clear";

  const onDeleted = () => {
    setOpen(false);
    toast.success(`Deleted "${team.name}"`);
    // Carry on in another of the person's teams, or start again at Welcome.
    const remaining = current.organizations.filter((o) => o.id !== team.id);
    if (remaining[0]) current.select(remaining[0].id);
    router.replace(homePath(remaining));
  };

  return (
    <section aria-labelledby="danger-heading" className="surface-card [--card-border:var(--card-border-danger)]">
      <div className="flex flex-col gap-4 p-5 md:flex-row md:items-center md:justify-between md:p-6">
        <div className="min-w-0">
          <h2 id="danger-heading" className="text-danger text-base font-semibold">
            Delete this team
          </h2>
          <p className="text-muted-foreground mt-1 text-sm">
            Deletes the team&apos;s projects, diagrams and connections for good. Nothing in your cloud changes.
          </p>
          {check.state === "checking" && (
            <p role="status" className="text-muted-foreground mt-2 flex items-center gap-2 text-sm">
              <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
              Checking the team&apos;s deployments…
            </p>
          )}
          {check.state === "failed" && (
            <ErrorState
              variant="inline"
              className="mt-2"
              description="Clutter could not check the team's deployments, so Delete is off."
              onRetry={() => void deploymentsQ.refetch()}
              retrying={deploymentsQ.isFetching}
            />
          )}
          {check.state === "holding" && (
            <p className="text-foreground mt-2 text-sm">
              {holding === 1 ? "One deployment may" : `${holding} deployments may`} still hold resources in your cloud. Destroy{" "}
              {holding === 1 ? "it" : "them"} in each diagram&apos;s Deploy view first.
            </p>
          )}
        </div>
        <Button type="button" variant="destructive" className="md:shrink-0" disabled={blocked} onClick={() => setOpen(true)}>
          Delete team
        </Button>
      </div>
      {open && <DeleteTeamDialog team={team} onCancel={() => setOpen(false)} onDeleted={onDeleted} />}
    </section>
  );
}
