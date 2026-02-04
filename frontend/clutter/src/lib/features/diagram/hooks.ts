// lib/features/diagram/hooks.ts

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { diagramApi } from "./api";
import { diagramKeys } from "./keys";
import type { CreateDiagramInput, Diagram, DiagramEdge, DiagramNode } from "./types";

export const useDiagrams = (token?: string | null, projectId?: string | null) => {
  return useQuery({
    queryKey: projectId ? diagramKeys.list(projectId) : ["diagram", "list", "no-project"],
    queryFn: () => diagramApi.list(token as string, projectId as string),
    enabled: !!token && !!projectId,
    staleTime: 5 * 60 * 1000,
  });
};

export const useDiagram = (token?: string | null, projectId?: string | null, diagramId?: string | null) => {
  return useQuery({
    queryKey: projectId && diagramId ? diagramKeys.detail(projectId, diagramId) : ["diagram", "detail", "missing"],
    queryFn: () => diagramApi.get(token as string, projectId as string, diagramId as string),
    enabled: !!token && !!projectId && !!diagramId,
    staleTime: 5 * 60 * 1000,
  });
};

export const useCreateDiagram = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateDiagramInput) => diagramApi.create(token as string, input),
    onSuccess: (created) => {
      // update the list cache for that project
      qc.setQueryData<Diagram[]>(diagramKeys.list(created.id), (prev) => [created, ...(prev ?? [])]);
    },
  });
};

export const useUpdateDiagramData = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (input: {
      projectId: string;
      diagramId: string;
      name: string;
      nodes: DiagramNode[];
      edges: DiagramEdge[];
    }) =>
      diagramApi.update(token as string, {
        projectId: input.projectId,
        diagramId: input.diagramId,
        name: input.name,
        uiLayout: { nodes: input.nodes, edges: input.edges },
      }),

    onSuccess: (updated, vars) => {
      // update detail cache
      qc.setQueryData<Diagram>(diagramKeys.detail(vars.projectId, vars.diagramId), updated);

      // update list cache for that project
      qc.setQueryData<Diagram[]>(diagramKeys.list(vars.projectId), (prev) => {
        if (!prev) return prev;
        return prev.map((d) => (d.id === updated.id ? updated : d));
      });
    },
  });
};

export const useDeleteDiagram = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (input: { projectId: string; diagramId: string }) =>
      diagramApi.delete(token as string, input.projectId, input.diagramId),

    onSuccess: (_void, vars) => {
      // remove detail cache for that diagram
      qc.removeQueries({ queryKey: diagramKeys.detail(vars.projectId, vars.diagramId) });

      // remove it from the project list cache
      qc.setQueryData<Diagram[]>(diagramKeys.list(vars.projectId), (prev) => {
        if (!prev) return prev;
        return prev.filter((d) => d.id !== vars.diagramId);
      });
    },
  });
};
