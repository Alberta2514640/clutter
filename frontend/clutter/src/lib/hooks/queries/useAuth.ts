import { useQuery } from "@tanstack/react-query";
import { apiClient, User } from "@/lib/api-client";

/**
 * Hook to get the current authenticated user
 * Returns null if not authenticated
 */
export const useAuth = () => {
  return useQuery<User | null>({
    queryKey: ["auth", "currentUser"],
    queryFn: () => apiClient.getCurrentUser(),
    staleTime: Infinity, // User data rarely changes
    retry: false, // Don't retry if auth fails
  });
};

/**
 * Hook to check if user is authenticated
 */
export const useIsAuthenticated = () => {
  const { data: user } = useAuth();
  return !!user;
};