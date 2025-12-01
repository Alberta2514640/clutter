import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useRouter } from "next/navigation";

/**
 * Hook to handle Google login
 */
export const useLogin = () => {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationFn: (googleIdToken: string) => apiClient.login(googleIdToken),
    onSuccess: (data) => {
      // Update the auth query cache with the user data
      queryClient.setQueryData(["auth", "currentUser"], data.user);

      // Navigate to dashboard
      router.push("/dashboard");
    },
    onError: (error: Error) => {
      console.error("Login error:", error);
      // Error handling can be done in the component
    },
  });
};