import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useRouter } from "next/navigation";

interface CreateProjectData {
  name: string;
  description: string;
}

/**
 * Hook to create a new project
 */
export const useCreateProject = (organizationId: string) => {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationFn: (data: CreateProjectData) => apiClient.createProject(organizationId, data),
    onSuccess: (newProject: any) => {
      // Invalidate projects query to refetch the list
      queryClient.invalidateQueries({ queryKey: ["projects", organizationId] });

      // Navigate to the new project page
      // Assuming your API returns the project with an 'id' field
      if (newProject?.id) {
        router.push(`/projects/${newProject.id}`);
      }
    },
    onError: (error: Error) => {
      console.error("Failed to create project:", error);
    },
  });
};

