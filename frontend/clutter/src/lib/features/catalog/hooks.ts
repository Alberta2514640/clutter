import { useQuery } from "@tanstack/react-query";
import { catalogApi } from "./api";
import { catalogKeys } from "./keys";

// The catalog changes only when B is redeployed.
export const useCatalog = (token?: string | null) => {
  return useQuery({
    queryKey: catalogKeys.list(),
    queryFn: () => catalogApi.list(token as string),
    enabled: !!token,
    staleTime: 10 * 60 * 1000,
  });
};

export const useLinkedAccounts = (token?: string | null, organizationId?: string | null, connector?: string | null) => {
  return useQuery({
    queryKey: catalogKeys.linked(organizationId ?? "", connector ?? ""),
    queryFn: () => catalogApi.linked(token as string, organizationId as string, connector as string),
    enabled: !!token && !!organizationId && !!connector,
    staleTime: 60 * 1000,
  });
};
