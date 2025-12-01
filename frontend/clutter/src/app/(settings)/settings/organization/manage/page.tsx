"use client";

import AddUsersDropdown, { type MemberOption } from "./_components/AddUsersDropdown";
import ManageForm from "./_components/ManageForm";
import Members, { type OrgMember } from "./_components/Members";

//TO DO: Api call
const mockUsers: MemberOption[] = [
  { id: "2", name: "Santiago Fuentes", email: "santiago@ucalgary.com" },
];

const currentMembers: OrgMember[] = [
  { id: "m1", name: "Hamza Amar", email: "hamza.amar@ucalgary.ca", role: "Project Admin" },
];

export default function OrganizationManagePage() {

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

            <div className="lg:border-l border-t lg:border-t-0 border-slate-800 lg:pl-6">
              <AddUsersDropdown
                options={mockUsers}
                onPick={(member) => {
                  console.log("picked:", member);
                  // call your add-member/invite logic here
                }}
                placeholder="Add users..."
              />

              <Members members={currentMembers} />
            </div>
            
          </div>
        </div>
      </section>
    </main>
  );
}
