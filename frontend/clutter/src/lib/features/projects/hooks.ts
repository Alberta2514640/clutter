import { useQuery } from "@tanstack/react-query";
import { projectsApi } from "./api";
import { projectKeys } from "./keys";

export const useProjects = (tenantId?: string | null) =>
  useQuery({
    queryKey: tenantId ? projectKeys.list(tenantId) : ["projects", "list", "no-tenant"],
    queryFn: () => projectsApi.listByTenant(tenantId as string),
    enabled: !!tenantId, // ✅ only runs when tenantId exists
    staleTime: 60 * 1000,
  });
