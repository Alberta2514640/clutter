import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

interface UseRecentRunsOptions {
  enabled?: boolean;
}

/**
 * Hook to fetch recent runs across all projects
 */
export const useRecentRuns = (options?: UseRecentRunsOptions) => {
  return useQuery({
    queryKey: ["runs", "recent"],
    queryFn: () => apiClient.getRecentRuns(),
    enabled: options?.enabled !== false,
    staleTime: 1000 * 60 * 2, // 2 minutes (runs change more frequently)
    refetchInterval: 1000 * 30, // Auto-refetch every 30 seconds for live updates
  });
};