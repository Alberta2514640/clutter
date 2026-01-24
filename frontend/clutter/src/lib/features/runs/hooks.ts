import { useQuery } from "@tanstack/react-query";
import { runsApi } from "./api";
import { runKeys } from "./keys";

export const useRecentRuns = (tenantId?: string | null) =>
  useQuery({
    queryKey: tenantId ? runKeys.recent(tenantId) : ["runs", "recent", "no-tenant"],
    queryFn: () => runsApi.listRecentByTenant(tenantId as string),
    enabled: !!tenantId, // ✅ waits for user.tenantId
    staleTime: 30 * 1000,
  });
