import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "next/navigation";

import { projectsApi } from "./api";
import { projectKeys } from "./keys";
import type { CreateProjectInput, Project, UpdateProjectInput } from "./types";

export const useProjects = (token?: string | null, organizationId?: string | null) => {
  return useQuery({
    // key should represent the resource: projects for org
    queryKey: organizationId ? projectKeys.list(organizationId) : ["projects", "list", "no-org"],
    queryFn: () => projectsApi.listByOrganization(token as string, organizationId as string),
    enabled: !!token && !!organizationId,
    staleTime: 60 * 1000,
  });
};

export function useProjectId() {
  const params = useParams();
  const raw = params?.projectId;
  return (Array.isArray(raw) ? raw[0] : raw) ?? null;
}

export const useProject = (token?: string | null, projectId?: string | null) => {
  return useQuery({
    queryKey: projectId ? projectKeys.byId(projectId) : ["projects", "byId", "none"],
    queryFn: () => projectsApi.getById(token as string, projectId as string),
    enabled: !!token && !!projectId,
    staleTime: 60 * 1000,
  });
};

export const useCreateProject = () => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: { token?: string | null } & CreateProjectInput) => {
      return projectsApi.create(input.token as string, {
        organizationId: input.organizationId,
        name: input.name,
        description: input.description,
      });
    },

    onSuccess: (created) => {
      qc.setQueryData(projectKeys.byId(created.id), created);
      qc.invalidateQueries({ queryKey: projectKeys.list(created.organizationId) });
    },
  });
};

export const useUpdateProject = () => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: { token?: string | null; projectId: string; data: UpdateProjectInput }) => {
      return projectsApi.update(input.token as string, input.projectId, input.data);
    },

    onSuccess: (updated) => {
      qc.setQueryData(projectKeys.byId(updated.id), updated);
      qc.invalidateQueries({ queryKey: projectKeys.list(updated.organizationId) });
    },
  });
};

export const useDeleteProject = () => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: { token?: string | null; projectId: string }) => {
      return projectsApi.delete(input.token as string, input.projectId);
    },

    onSuccess: (_data, vars) => {
      qc.removeQueries({ queryKey: projectKeys.byId(vars.projectId) });
      // we don’t know orgId here unless you pass it in; so invalidate all project lists
      qc.invalidateQueries({ queryKey: ["projects", "list"] });
    },
  });
};
