import { useQuery } from "@tanstack/react-query";
import { resourcesApi } from "./api";
import { resourceKeys } from "./keys";

export function useSupportedResources() {
  return useQuery({
    queryKey: resourceKeys.all,
    queryFn: () => resourcesApi.list(),
    staleTime: 1000 * 60 * 10, // cache for 10 minutes — this data rarely changes
  });
}
