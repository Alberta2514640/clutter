"use client";

import ConfirmDialog from "@/components/common/ConfirmDialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useCurrentOrganization, useDeleteOrganization, useUpdateOrganization } from "@/lib/features/organization/hooks";
import { useLogout, useMe } from "@/lib/features/user/hooks";
import { isConflict } from "@/lib/features/user/session";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import ManageForm, { type ManageFormValues } from "./_components/ManageForm";
import Members from "./_components/Members";

export default function OrganizationManagePage() {
  const router = useRouter();
  const logout = useLogout();

  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const orgQ = useCurrentOrganization(token);

  const updateOrg = useUpdateOrganization(token);
  const deleteOrg = useDeleteOrganization(token);

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const organization = orgQ.organization;

  const initialValues = useMemo<ManageFormValues>(
    () => ({
      orgName: organization?.name ?? "",
      orgId: organization?.id ?? "",
      description: organization?.description ?? "",
    }),
    [organization?.name, organization?.id, organization?.description],
  );

  const isLoading = meQ.isLoading || orgQ.isLoading;
  const isSaving = updateOrg.isPending || deleteOrg.isPending;

  const handleDelete = async () => {
    if (!token || !organization?.id) return;
    setDeleteError(null);
    const deletedId = organization.id;
    try {
      await deleteOrg.mutateAsync(deletedId);
    } catch (err) {
      setDeleteError(
        isConflict(err)
          ? "This organization still has live deployments. Destroy them on each diagram's Deploy tab, then delete the organization."
          : "Clutter could not delete the organization. Try again.",
      );
      return;
    }
    setShowDeleteConfirm(false);

    // Carry on in another of the user's organizations; sign out only when none remain.
    const remaining = orgQ.organizations.filter((o) => o.id !== deletedId);
    if (remaining.length > 0) {
      orgQ.select(remaining[0].id);
      router.replace("/dashboard");
      return;
    }
    logout();
    router.replace("/login");
    router.refresh();
  };

  if (!isLoading && !token) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-400">Sign in to manage your organization.</div>
      </div>
    );
  }

  if (isLoading && !organization) {
    return (
      <div className="flex items-center justify-center py-12">
        <div role="status" className="text-gray-400">
          Loading organization settings...
        </div>
      </div>
    );
  }

  if (orgQ.isError) {
    return (
      <div className="py-6">
        <Alert className="border-red-800 bg-red-900/20">
          <AlertDescription>Clutter could not load your organization. Reload the page to try again.</AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!organization) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-400">You have no organization yet.</div>
      </div>
    );
  }

  return (
    <>
      <div>
        <section className="rounded-2xl border border-slate-800 bg-slate-900/70 shadow-xl">
          <div className="border-b border-slate-800 px-6 py-4">
            <h2 className="text-lg font-semibold">Organization details</h2>
            <p className="mt-1 max-w-xl text-sm text-slate-400">Update the description of {organization.name}.</p>
          </div>

          <div className="px-6 py-6">
            {updateOrg.isError && (
              <Alert className="mb-6 border-red-800 bg-red-900/20">
                <AlertDescription>Clutter could not save your changes. Try again.</AlertDescription>
              </Alert>
            )}
            {updateOrg.isSuccess && (
              <Alert role="status" className="mb-6 border-teal-500/30 bg-teal-500/10">
                <AlertDescription className="text-teal-50">Your changes are saved.</AlertDescription>
              </Alert>
            )}

            <div className="grid items-start gap-6 lg:grid-cols-2">
              <ManageForm
                initialValues={initialValues}
                onSubmit={async (values: ManageFormValues) => {
                  if (!token) return;
                  try {
                    await updateOrg.mutateAsync({
                      organizationId: organization.id,
                      data: { description: values.description },
                    });
                  } catch {
                    // Shown from updateOrg.isError above.
                  }
                }}
                onDelete={() => {
                  setDeleteError(null);
                  setShowDeleteConfirm(true);
                }}
                isSaving={isSaving}
                isDeleting={deleteOrg.isPending}
              />

              <div className="border-slate-800 lg:border-l lg:pl-6 border-t pt-6 lg:pt-0 lg:border-t-0">
                <Members me={meQ.data ?? null} memberCount={organization.total_members} />
              </div>
            </div>
          </div>
        </section>
      </div>

      {showDeleteConfirm && (
        <ConfirmDialog
          title="Delete organization"
          confirmLabel="Delete organization"
          pendingLabel="Deleting…"
          isPending={deleteOrg.isPending}
          error={deleteError}
          onCancel={() => setShowDeleteConfirm(false)}
          onConfirm={() => void handleDelete()}>
          <p>
            Delete <span className="font-medium text-white">&quot;{organization.name}&quot;</span> and its projects and diagrams? This cannot be undone.
          </p>
          <p>Clutter refuses while any of its deployments is still live.</p>
        </ConfirmDialog>
      )}
    </>
  );
}
