"use client";

import { useMemo } from "react";
import AddUsersDropdown, { type MemberOption } from "./_components/AddUsersDropdown";
import ManageForm from "./_components/ManageForm";
import Members from "./_components/Members";

import { useAddMember, useAvailableUsers, useDeleteOrganization, useMembers, useOrganization, useUpdateOrganization, } from "@/lib/features/organization/hooks";

export default function OrganizationManagePage() {
  const orgQ = useOrganization();
  const membersQ = useMembers();
  const usersQ = useAvailableUsers();

  const updateOrg = useUpdateOrganization();
  const deleteOrg = useDeleteOrganization();
  const addMember = useAddMember();

  const organization = orgQ.data;
  const members = membersQ.data ?? [];
  const availableUsers = usersQ.data ?? [];

  const isLoading = orgQ.isLoading || membersQ.isLoading || usersQ.isLoading;
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
                orgName: organization?.name || "",
                slug: organization?.slug || "",
                timeZone: organization?.timeZone || "America/Edmonton",
              }}
              onSubmit={async (values) => {
                await updateOrg.mutateAsync({
                  name: values.orgName,
                  slug: values.slug,
                  timeZone: values.timeZone,
                });
              }}
              onDelete={async () => {
                const confirmed = window.confirm(
                  "Are you sure you want to delete this organization? This action cannot be undone."
                );

                if (confirmed) {
                  await deleteOrg.mutateAsync();
                  window.location.href = "/";
                }
              }}
              isSaving={isSaving}
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

          {(orgQ.isError || membersQ.isError || usersQ.isError) && (
            <div className="mt-4 text-sm text-red-400">
              {orgQ.isError && <div>Organization error: {String(orgQ.error)}</div>}
              {membersQ.isError && <div>Members error: {String(membersQ.error)}</div>}
              {usersQ.isError && <div>Users error: {String(usersQ.error)}</div>}
            </div>
          )}
        </div>
      </section>
    </main>
  );
}