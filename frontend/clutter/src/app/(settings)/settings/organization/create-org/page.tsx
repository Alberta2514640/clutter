"use client";

import PageHeader from "@/components/common/PageHeader";
import { toast } from "@/components/ui/sonner";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import { useCreateOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { isConflict } from "@/lib/features/user/session";
import { useRouter } from "next/navigation";
import CreateOrgForm from "./_components/CreateOrgForm";

// A further team, reached from the team switcher's "New team". The new team
// becomes the current one (useCreateOrganization) and opens on its Projects.
export default function CreateTeamPage() {
  const router = useRouter();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const createOrg = useCreateOrganization(token);

  return (
    <div className="max-w-2xl">
      <PageHeader
        eyebrow="Teams"
        title="New team"
        description="Use another team to keep work apart, for example for a client. Each team has its own projects and connections."
      />
      <CreateOrgForm
        isCreating={createOrg.isPending}
        conflict={createOrg.isError && isConflict(createOrg.error)}
        failed={createOrg.isError && !isConflict(createOrg.error)}
        onEdit={() => createOrg.isError && createOrg.reset()}
        onCancel={() => router.back()}
        onSubmit={async (values) => {
          if (!token) return;
          try {
            await createOrg.mutateAsync({
              organizationName: values.name,
              ...(values.description ? { description: values.description } : {}),
            });
            toast.success("Team created");
            router.push(PROJECTS_PATH);
          } catch {
            // Shown by the form from conflict and failed.
          }
        }}
      />
    </div>
  );
}
