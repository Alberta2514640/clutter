"use client";

import StatsCards from "../_components/StatsCards";
import AddMembersCard from "./_components/AddMembers";
import ManageForm from "./_components/ManageForm";

export default function OrganizationManagePage() {

  return (
    <main>
      <StatsCards 
        planType={'Trial'}
        projectCount={3}
        memberCount={6}
      />

      <section className="bg-slate-900/70 border border-slate-800 rounded-2xl shadow-xl">
        <div className="border-b border-slate-800 px-6 py-4">
          <h2 className="text-lg font-semibold">Workspace settings</h2>
          <p className="text-sm text-slate-400 mt-1 max-w-xl">
            Update the core details for this organization. Changes may take a few
            seconds to propagate.
          </p>
        </div>

        <div className="px-6 py-6">
          <div className="grid gap-6 lg:grid-cols-2">
            
            <ManageForm
              initialValues={{
                orgName: "Demo Organization",
                slug: "demo-org",
                timeZone: "America/Edmonton",
              }}
              onSubmit={(values) => {
                // TODO: call API to update tenant
                console.log("update org settings", values);
              }}
              onDelete={async () => {
                // TODO: handle delete
                console.log("delete org clicked");
              }}
            />

            <div className="lg:border-l border-t lg:border-t-0 border-slate-800 lg:pl-6 pt-6 lg:pt-0">
              <AddMembersCard />
            </div>
            
          </div>
        </div>
      </section>
    </main>
  );
}
