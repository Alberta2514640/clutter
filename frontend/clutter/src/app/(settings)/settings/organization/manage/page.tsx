"use client";

import { useAddMember, useAvailableUsers, useMembers } from "@/lib/features/members/hooks";
import { useDeleteOrganization, useOrganizations, useUpdateOrganization, } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { useMemo } from "react";
import AddUsersDropdown, { type MemberOption } from "./_components/AddUsersDropdown";
import ManageForm, { type ManageFormValues } from "./_components/ManageForm";
import Members from "./_components/Members";

export default function OrganizationManagePage() {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const orgQ = useOrganizations(token);

  // members still fake
  const membersQ = useMembers();
  const usersQ = useAvailableUsers();

  const updateOrg = useUpdateOrganization(token);
  const deleteOrg = useDeleteOrganization(token);
  const addMember = useAddMember();

  const organizations = orgQ.data ?? [];
  const organization = organizations[0] ?? null;

  const members = membersQ.data ?? [];
  const availableUsers = usersQ.data ?? [];

  const isLoading =
    meQ.isLoading || orgQ.isLoading || membersQ.isLoading || usersQ.isLoading;

  const isSaving = updateOrg.isPending || deleteOrg.isPending || addMember.isPending;

  const userOptions: MemberOption[] = useMemo(
    () =>
      availableUsers.map((user) => ({
        id: user.id,
        name: user.name,
        email: user.email,
      })),
    [availableUsers]
  );

  if (!isLoading && !token) {
    return (
      <main className="flex items-center justify-center py-12">
        <div className="text-gray-400">Please sign in to manage your organization.</div>
      </main>
    );
  }

  if (isLoading && !organization) {
    return (
      <main className="flex items-center justify-center py-12">
        <div className="text-gray-400">Loading organization settings...</div>
      </main>
    );
  }

  return (
    <main>
      <section className="bg-slate-900/70 border border-slate-800 rounded-2xl shadow-xl">
        <div className="border-b border-slate-800 px-6 py-4">
          <h2 className="text-lg font-semibold">Workspace settings</h2>
          <p className="text-sm text-slate-400 mt-1 max-w-xl">
            Update the core details for this organization. Changes may take a few seconds to propagate.
          </p>
        </div>

        <div className="px-6 py-6">
          <div className="grid items-start gap-6 lg:grid-cols-2">
            <ManageForm
              initialValues={{
                orgName: organization?.name ?? "",
                orgId: organization?.id ?? "", // ✅ required now
                description: organization?.description ?? "",
              }}
              onSubmit={async (values: ManageFormValues) => {
                if (!token || !organization?.id) return;

                await updateOrg.mutateAsync({
                  organizationId: organization.id,
                  data: {
                    description: values.description,
                  },
                });
              }}
              onDelete={async () => {
                if (!token || !organization?.id) return;

                const confirmed = window.confirm(
                  "Are you sure you want to delete this organization? This action cannot be undone."
                );

                if (confirmed) {
                  await deleteOrg.mutateAsync(organization.id);
                  window.location.href = "/";
                }
              }}
              isSaving={isSaving}
              isDeleting={deleteOrg.isPending}
            />

            <div className="lg:border-l border-t lg:border-t-0 border-slate-800 lg:pl-6">
              <AddUsersDropdown
                options={userOptions}
                onPick={async (member) => {
                  await addMember.mutateAsync({ userId: member.id, role: "Member" });
                }}
                placeholder="Add users..."
                disabled={isSaving}
              />
              <Members members={members} />
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}