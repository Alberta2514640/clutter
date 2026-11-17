// lib/features/projects/hooks.ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { projectsApi } from "./api";
import { projectKeys } from "./keys";
import type { CreateProjectInput, Project, UpdateProjectInput } from "./types";

export const useProjects = (token?: string | null, organizationId?: string | null) => {
  return useQuery({
    queryKey: organizationId ? projectKeys.list(organizationId) : projectKeys.lists(),
    queryFn: () => projectsApi.listByOrganization(token as string, organizationId as string),
    enabled: !!token && !!organizationId,
    staleTime: 60 * 1000,
  });
};

export const useProject = (token?: string | null, organizationId?: string | null, projectId?: string | null) => {
  return useQuery({
    queryKey: organizationId && projectId ? projectKeys.detail(organizationId, projectId) : projectKeys.details(),
    queryFn: () => projectsApi.getById(token as string, organizationId as string, projectId as string),
    enabled: !!token && !!organizationId && !!projectId,
    staleTime: 60 * 1000,
  });
};

// useProjectOrganization finds the organization that owns a project among the
// caller's organizations. Pages reached by project id (the diagram editor) must
// act in the project's own organization, not the user's first one: the backend
// refuses ids from another organization.
export const useProjectOrganization = (
  token?: string | null,
  projectId?: string | null,
  organizationIds?: string[] | null,
) => {
  return useQuery({
    queryKey: projectId ? projectKeys.owner(projectId) : projectKeys.all,
    queryFn: async (): Promise<string | null> => {
      for (const organizationId of organizationIds ?? []) {
        const projects = await projectsApi.listByOrganization(token as string, organizationId);
        if (projects.some((project) => project.id === projectId)) return organizationId;
      }
      return null;
    },
    enabled: !!token && !!projectId && !!organizationIds && organizationIds.length > 0,
    staleTime: 5 * 60 * 1000,
  });
};

export const useCreateProject = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateProjectInput) => projectsApi.create(token as string, input),
    onSuccess: (created) => {
      qc.setQueryData<Project[]>(projectKeys.list(created.organizationId), (prev) => [created, ...(prev ?? [])]);
      qc.setQueryData(projectKeys.detail(created.organizationId, created.id), created);
    },
  });
};

export const useUpdateProject = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { projectId: string; data: UpdateProjectInput }) =>
      projectsApi.update(token as string, input.projectId, input.data),
    onSuccess: (updated) => {
      qc.setQueryData(projectKeys.detail(updated.organizationId, updated.id), updated);
      qc.setQueryData<Project[]>(projectKeys.list(updated.organizationId), (prev) =>
        (prev ?? []).map((p) => (p.id === updated.id ? updated : p))
      );
    },
  });
};

export const useDeleteProject = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (input: { projectId: string; organizationId: string }) =>
      projectsApi.delete(token as string, input.organizationId, input.projectId),
    onSuccess: (_void, vars) => {
      qc.removeQueries({ queryKey: projectKeys.detail(vars.organizationId, vars.projectId) });
      qc.setQueryData<Project[]>(projectKeys.list(vars.organizationId), (prev) =>
        (prev ?? []).filter((p) => p.id !== vars.projectId)
      );
    },
  });
};
