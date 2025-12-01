import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useRouter } from "next/navigation";

/**
 * Hook to handle logout
 */
export const useLogout = () => {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationFn: () => apiClient.logout(),
    onSuccess: () => {
      // Clear all auth-related queries
      queryClient.setQueryData(["auth", "currentUser"], null);
      
      // Optionally clear all queries
      queryClient.clear();

      // Navigate to login
      router.push("/login");
    },
  });
};