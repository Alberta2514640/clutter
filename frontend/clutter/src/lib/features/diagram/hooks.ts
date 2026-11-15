// lib/features/diagram/hooks.ts

import { projectKeys } from "@/lib/features/projects/keys";
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { DiagramSummary, Project } from "../projects/types";
import { diagramApi } from "./api";
import { emptyBlocksDocument, isBlocksDocument, type BlocksDocument } from "./blocksDocument";
import { diagramKeys } from "./keys";
import { filesPredateStatus } from "./types";
import type {
  CreateDiagramInput,
  Diagram,
  DiagramEdge,
  DiagramNode,
  GenerateTerraformInput,
  GenerateTerraformResult,
  RunTerraformResult,
  TerraformCommandInput,
} from "./types";

export const useDiagrams = (token?: string | null, projectId?: string | null) => {
  return useQuery({
    queryKey: projectId ? diagramKeys.list(projectId) : ["diagram", "list", "no-project"],
    queryFn: () => diagramApi.list(token as string, projectId as string),
    enabled: !!token && !!projectId,
    staleTime: 5 * 60 * 1000,
  });
};

// Whether any of these projects holds an older AWS-only diagram (one without
// the building-block marker). Shares the diagram lists' cache with useDiagrams.
export const useHasOlderDiagrams = (token: string | null | undefined, projectIds: string[]) => {
  const queries = useQueries({
    queries: projectIds.map((projectId) => ({
      queryKey: diagramKeys.list(projectId),
      queryFn: () => diagramApi.list(token as string, projectId),
      enabled: !!token,
      staleTime: 5 * 60 * 1000,
    })),
  });
  return {
    isLoading: queries.some((q) => q.isLoading),
    hasOlder: queries.some((q) => (q.data ?? []).some((d) => !isBlocksDocument(d.data))),
  };
};

export const useDiagram = (token?: string | null, projectId?: string | null, diagramId?: string | null) => {
  return useQuery({
    queryKey: projectId && diagramId ? diagramKeys.detail(projectId, diagramId) : ["diagram", "detail", "missing"],
    queryFn: () => diagramApi.get(token as string, projectId as string, diagramId as string),
    enabled: !!token && !!projectId && !!diagramId,
    staleTime: 5 * 60 * 1000,
  });
};

// Generation runs just after a save returns, so the first answer can be behind
// the save. pollWhileBehind keeps asking until the answer covers the latest save.
// Nothing is cached: an old answer must never stand in for a new save.
export const useTerraformFiles = (token: string | null, projectId: string, diagramId: string, pollWhileBehind: boolean) => {
  return useQuery({
    queryKey: diagramKeys.terraformFiles(projectId, diagramId),
    queryFn: () => diagramApi.terraformFiles(token as string, projectId, diagramId),
    enabled: !!token,
    staleTime: 0,
    gcTime: 0,
    retry: false,
    refetchInterval: (query) =>
      pollWhileBehind && query.state.status !== "error" && !query.state.data?.current && !filesPredateStatus(query.state.data)
        ? 1000
        : false,
  });
};

// A new diagram is a building-block diagram, which B's editor opens. A's create
// route always starts a diagram empty and unmarked, so the marker is saved
// next, with the document to start from (an example's blocks) or an empty
// one; if that fails, the unmarked diagram is removed again.
export const useCreateDiagram = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: CreateDiagramInput & { organizationId: string; document?: BlocksDocument }) => {
      const { organizationId: _orgId, document, ...payload } = input;
      const created = await diagramApi.create(token as string, payload);
      try {
        return await diagramApi.update(token as string, {
          projectId: payload.projectId,
          diagramId: created.id,
          name: created.name,
          data: document ?? emptyBlocksDocument(),
        });
      } catch (err) {
        await diagramApi.delete(token as string, payload.projectId, created.id).catch(() => undefined);
        throw err;
      }
    },
    onSuccess: (created, vars) => {
      qc.setQueryData<Diagram[]>(diagramKeys.list(created.projectId), (prev) => [created, ...(prev ?? [])]);
      qc.setQueryData<Project>(projectKeys.detail(vars.organizationId, created.projectId), (prev) => {
        if (!prev) return prev;
        const nextSummary: DiagramSummary = { id: created.id, name: created.name };
        const filtered = (prev.diagrams ?? []).filter((d) => d.id !== created.id);
        return {
          ...prev,
          diagrams: [nextSummary, ...filtered],
        };
      });
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
        data: { nodes: input.nodes, edges: input.edges },
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

// B's editor saves a building-block document through the same route. Nothing
// generates Terraform after it: B renders per deployment.
export const useSaveBlocksDiagram = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (input: { projectId: string; diagramId: string; name: string; document: BlocksDocument }) =>
      diagramApi.update(token as string, {
        projectId: input.projectId,
        diagramId: input.diagramId,
        name: input.name,
        data: input.document,
      }),
    onSuccess: (updated, vars) => {
      qc.setQueryData<Diagram>(diagramKeys.detail(vars.projectId, vars.diagramId), updated);
      qc.setQueryData<Diagram[]>(diagramKeys.list(vars.projectId), (prev) =>
        prev ? prev.map((d) => (d.id === updated.id ? updated : d)) : prev,
      );
    },
  });
};

// Generation is requested after every successful save. The preview reads the
// files route, so refreshing that query shows the new run, or its failure.
export const useGenerateTerraform = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation<GenerateTerraformResult, Error, GenerateTerraformInput>({
    mutationFn: (input: GenerateTerraformInput) => diagramApi.generateTerraform(token as string, input),
    onSettled: (_result, _error, vars) => {
      void qc.invalidateQueries({ queryKey: diagramKeys.terraformFiles(vars.projectId, vars.diagramId) });
    },
  });
};

export const useDeleteDiagram = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (input: { organizationId: string; projectId: string; diagramId: string }) =>
      diagramApi.delete(token as string, input.projectId, input.diagramId),
    onSuccess: (_void, vars) => {
      qc.removeQueries({ queryKey: diagramKeys.detail(vars.projectId, vars.diagramId) });
      qc.setQueryData<Diagram[]>(diagramKeys.list(vars.projectId), (prev) => {
        if (!prev) return prev;
        return prev.filter((d) => d.id !== vars.diagramId);
      });
      qc.setQueryData<Project>(projectKeys.detail(vars.organizationId, vars.projectId), (prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          diagrams: (prev.diagrams ?? []).filter((d) => d.id !== vars.diagramId),
        };
      });
    },
  });
};

export const useRunTerraform = (token?: string | null) => {
  return useMutation<RunTerraformResult, Error, TerraformCommandInput>({
    mutationFn: (input: TerraformCommandInput) => diagramApi.runTerraform(token as string, input),
  });
};
