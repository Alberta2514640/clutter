"use client";
import { useOrganizationActions, useOrganizationState } from "@/lib/stores/organizationStore";
import { useEffect } from "react";
import AddUsersDropdown, { type MemberOption } from "./_components/AddUsersDropdown";
import ManageForm from "./_components/ManageForm";
import Members from "./_components/Members";

export default function OrganizationManagePage() {
  // Get state from store
  const { organization, members, availableUsers, isLoading, isSaving } = useOrganizationState();
  // Get actions from store
  const { loadOrganization, loadMembers, loadAvailableUsers, updateOrganization, deleteOrganization, addMember, } = useOrganizationActions();

  // Load data on mount
  useEffect(() => {
    const loadData = async () => {
      await Promise.all([
        loadOrganization(),
        loadMembers(),
        loadAvailableUsers(),
      ]);
    };
    
    loadData();
  }, []);

  // Convert availableUsers to MemberOption format
  const userOptions: MemberOption[] = availableUsers.map((user) => ({
    id: user.id,
    name: user.name,
    email: user.email,
  }));

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
            Update the core details for this organization. Changes may take a few
            seconds to propagate.
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
                await updateOrganization({
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
                  await deleteOrganization();
                  // Optionally redirect to a different page
                  window.location.href = "/";
                }
              }}
              isSaving={isSaving}
            />
            <div className="lg:border-l border-t lg:border-t-0 border-slate-800 lg:pl-6">
              <AddUsersDropdown
                options={userOptions}
                onPick={async (member) => {
                  console.log("Adding member:", member);
                  // Default role - you can add a role selector in the dropdown if needed
                  await addMember(member.id, "Member");
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