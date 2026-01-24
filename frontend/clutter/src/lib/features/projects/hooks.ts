import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { projectsApi } from "./api";
import { projectKeys } from "./keys";
import type { Project } from "./types";

export const useProjects = (tenantId?: string | null) => {
  return useQuery({
    queryKey: tenantId ? projectKeys.list(tenantId) : ["projects", "list", "no-tenant"],
    queryFn: () => projectsApi.listByTenant(tenantId as string),
    enabled: !!tenantId,
    staleTime: 60 * 1000,
  });
};

export const useProject = (projectId?: string | null) => {
  return useQuery({
    queryKey: projectId ? projectKeys.byId(projectId) : ["projects", "byId", "none"],
    queryFn: () => projectsApi.getById(projectId as string),
    enabled: !!projectId,
    staleTime: 60 * 1000,
  });
};

export const useUpdateProject = () => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (args: { projectId: string; data: Partial<Project> }) =>
      projectsApi.update(args.projectId, args.data),

    onSuccess: (updated) => {
      // update cache for single project
      qc.setQueryData(projectKeys.byId(updated.projectId), updated);

      // also update any list caches (we don’t know tenantId here, so just invalidate lists)
      qc.invalidateQueries({ queryKey: projectKeys.all });
    },
  });
};

export const useDeleteProject = () => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (args: { projectId: string }) => projectsApi.delete(args.projectId),

    onSuccess: (_data, vars) => {
      qc.removeQueries({ queryKey: projectKeys.byId(vars.projectId) });
      qc.invalidateQueries({ queryKey: projectKeys.all });
    },
  });
};
