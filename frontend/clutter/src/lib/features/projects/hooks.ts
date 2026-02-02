import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "next/navigation";

import { projectsApi } from "./api";
import { projectKeys } from "./keys";
import type { Project } from "./types";

export const useProjects = (token?: string | null, organizationId?: string | null) => {
  return useQuery({
    queryKey: token ? projectKeys.list(token) : ["projects", "list", "no-tenant"],
    queryFn: () => projectsApi.listByToken(token as string, organizationId as string),
    enabled: !!token,
    staleTime: 60 * 1000,
  });
};

export function useProjectId() {
  const params = useParams();
  return (params.projectId as string) ?? null;
}

export const useProject = (token?: string | null, projectId?: string | null) => {
  return useQuery({
    queryKey: projectId ? projectKeys.byId(projectId) : ["projects", "byId", "none"],
    queryFn: () => projectsApi.getById(token as string, projectId as string),
    enabled: !!projectId,
    staleTime: 60 * 1000,
  });
};

export const useCreateProject = () => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (args: { token?: string | null; organizationId: string; name: string; description?: string }) => {
      return projectsApi.create(args.token as string, {
        organizationId: args.organizationId,
        name: args.name,
        description: args.description,
      });
    },

    onSuccess: (created) => {
      qc.invalidateQueries({ queryKey: projectKeys.all });

      if (created?.projectId) {
        qc.setQueryData(projectKeys.byId(created.projectId), created);
      }
    },
  });
};

export const useUpdateProject = () => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (args: { projectId: string; data: Partial<Project>; token?: string | null }) => projectsApi.update(args.projectId, args.data, args.token as string),

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
    mutationFn: async (args: { projectId: string; token?: string | null }) => projectsApi.delete(args.projectId, args.token as string),

    onSuccess: (_data, vars) => {
      qc.removeQueries({ queryKey: projectKeys.byId(vars.projectId) });
      qc.invalidateQueries({ queryKey: projectKeys.all });
    },
  });
};
