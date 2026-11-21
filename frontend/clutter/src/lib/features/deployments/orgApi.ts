import { adapterFetch, seg } from "@/lib/adapterApi";
import type { OrgDeploymentListItem } from "./types";

export const orgDeploymentsApi = {
  // Every deployment in the organization, oldest first. A project id keeps only
  // that project's diagrams' deployments; B answers 404 for a project of
  // another organization.
  list: async (token: string, organizationId: string, projectId?: string | null): Promise<OrgDeploymentListItem[]> => {
    const query = projectId ? `?project=${seg(projectId)}` : "";
    const json = await adapterFetch<{ items: OrgDeploymentListItem[] | null }>(
      `/v1/organizations/${seg(organizationId)}/deployments${query}`,
      token,
      { method: "GET" },
    );
    return json.items ?? [];
  },
};
