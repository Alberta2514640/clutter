"use client";

import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";

export default function HeaderSettings() {
  const meQ = useMe();
  const { organization } = useCurrentOrganization(meQ.data?.token ?? null);

  return (
    <header className="py-6 md:px-6 md:py-10">
      <p className="text-xs uppercase tracking-wide text-teal-400 mb-1">Settings</p>
      <h1 className="text-3xl font-semibold mb-1">Organization settings</h1>
      <p className="text-sm text-slate-400">
        {organization ? (
          <>
            For <span className="text-slate-200">{organization.name}</span>: its details, members and linked cloud accounts. Change the organization in the sidebar.
          </>
        ) : (
          "Your organization's details, members and linked cloud accounts."
        )}
      </p>
    </header>
  );
}
