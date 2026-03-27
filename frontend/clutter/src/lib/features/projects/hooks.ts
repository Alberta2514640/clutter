// lib/features/projects/hooks.ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { projectsApi } from "./api";
import { projectKeys } from "./keys";
import type {
  CreateProjectInput,
  GetTerraformLogsInput,
  Project,
  TerraformLogsResponse,
  UpdateProjectInput,
} from "./types";

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

export const useTerraformLogs = (token?: string | null, input?: GetTerraformLogsInput | null) => {
  return useQuery<TerraformLogsResponse>({
    queryKey:
      input?.orgId && input?.projId && input?.diagramId
        ? projectKeys.terraformLogs(input.orgId, input.projId, input.diagramId)
        : [...projectKeys.all, "terraform-logs", "missing"],
    queryFn: () => projectsApi.getTerraformLogs(token as string, input as GetTerraformLogsInput),
    enabled: !!token && !!input?.orgId && !!input?.projId && !!input?.diagramId,
    staleTime: 15 * 1000,
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
