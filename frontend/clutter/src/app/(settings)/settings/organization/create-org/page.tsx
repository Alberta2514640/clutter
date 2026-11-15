"use client";

import { useCreateOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { isConflict } from "@/lib/features/user/session";
import { useRouter } from "next/navigation";
import CreateOrgForm from "./_components/CreateOrgForm";

// A further organization, reached from the sidebar's "New organization". The
// new one becomes the current organization (useCreateOrganization).
export default function OrganizationCreatePage() {
  const router = useRouter();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const createOrg = useCreateOrganization(token);

  const error = !createOrg.isError
    ? null
    : isConflict(createOrg.error)
      ? "You already have an organization with that name. Choose another name."
      : "Clutter could not create the organization. Try again.";

  return (
    <CreateOrgForm
      isCreating={createOrg.isPending}
      error={error}
      onCancel={() => router.back()}
      onSubmit={async (values) => {
        if (!token) return;
        try {
          await createOrg.mutateAsync({
            organizationName: values.orgName,
            ...(values.description ? { description: values.description } : {}),
          });
          router.push("/dashboard");
        } catch {
          // Shown from createOrg.error above.
        }
      }}
    />
  );
}
