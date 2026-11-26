import { useQuery } from "@tanstack/react-query";
import { RUN_POLL_MS } from "./hooks";
import { deploymentKeys } from "./keys";
import { orgDeploymentsApi } from "./orgApi";
import { isActive } from "./types";

// Every deployment in the organization, or one project's when projectId is
// set, each with its latest run. It polls while any run is queued or running.
// Every deployment mutation invalidates deploymentKeys.organizations().
export const useOrgDeployments = (token?: string | null, organizationId?: string | null, projectId?: string | null) => {
  return useQuery({
    queryKey: deploymentKeys.forOrganization(organizationId ?? "", projectId ?? ""),
    queryFn: () => orgDeploymentsApi.list(token as string, organizationId as string, projectId),
    enabled: !!token && !!organizationId,
    staleTime: 0,
    refetchInterval: (query) => ((query.state.data ?? []).some((d) => isActive(d.latest_run)) ? RUN_POLL_MS : false),
  });
};
