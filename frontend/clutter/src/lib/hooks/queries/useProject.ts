import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

interface UseProjectsOptions {
  enabled?: boolean;
}

/**
 * Hook to fetch all projects for the organization
 */
export const useProjects = (organizationId: string, options?: UseProjectsOptions) => {
  organizationId = "org-123"; // Temporary hardcode for testing
  return useQuery({
    queryKey: ["projects", organizationId],
    queryFn: () => apiClient.getProjects(organizationId),
    enabled: options?.enabled !== false && !!organizationId, // Only fetch if enabled and organizationId exists
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
};