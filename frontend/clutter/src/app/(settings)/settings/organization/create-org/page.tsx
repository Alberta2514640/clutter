"use client";

import CreateOrgForm from "./_components/CreateOrgForm";

export default function OrganizationCreatePage() {

  return (
    <CreateOrgForm
        initialValues={{
            orgName: "",
            slug: "",
            region: "us-west-2",
            timeZone: "America/Edmonton",
            description: "",
            visibility: "private",
        }}
        onSubmit={(values) => {
            //to do
            console.log(values)
        }}
    />
  );
}
