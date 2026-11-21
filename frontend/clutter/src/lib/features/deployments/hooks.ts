import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { deploymentsApi } from "./api";
import { deploymentKeys } from "./keys";
import { isActive } from "./types";
import type { CreateDeploymentInput, DeploymentListItem, Run, RunAction, UpdateDeploymentInput } from "./types";

// How often a deployment list asks about a run in progress. B pushes nothing.
export const RUN_POLL_MS = 2000;

// Every list of the organization's deployments (orgHooks.ts) shows the rows a
// mutation changes, so every mutation refreshes them when it settles.
const refreshOrganizationLists = (qc: QueryClient) => void qc.invalidateQueries({ queryKey: deploymentKeys.organizations() });

// One diagram's deployments, each with its latest run. It polls while any run
// is queued or running, so status and logs stay live.
export const useDiagramDeployments = (token?: string | null, organizationId?: string | null, diagramId?: string | null) => {
  return useQuery({
    queryKey: deploymentKeys.forDiagram(organizationId ?? "", diagramId ?? ""),
    queryFn: () => deploymentsApi.listForDiagram(token as string, organizationId as string, diagramId as string),
    enabled: !!token && !!organizationId && !!diagramId,
    staleTime: 0,
    refetchInterval: (query) => ((query.state.data ?? []).some((d) => isActive(d.latest_run)) ? RUN_POLL_MS : false),
  });
};

// Compatibility is computed from the diagram as saved, so it is refetched
// after each save (see useInvalidateDeploymentChecks).
export const useCompatibility = (token?: string | null, organizationId?: string | null, deploymentId?: string | null) => {
  return useQuery({
    queryKey: deploymentKeys.compatibility(organizationId ?? "", deploymentId ?? ""),
    queryFn: () => deploymentsApi.compatibility(token as string, organizationId as string, deploymentId as string),
    enabled: !!token && !!organizationId && !!deploymentId,
    staleTime: 0,
  });
};

// The preview is rendered on demand and never kept: an old one must not stand
// in for a later save. The code panel mounts it only while it is open.
export const usePreview = (token: string | null, organizationId: string | null, deploymentId: string) => {
  return useQuery({
    queryKey: deploymentKeys.preview(organizationId ?? "", deploymentId),
    queryFn: () => deploymentsApi.preview(token as string, organizationId as string, deploymentId),
    enabled: !!token && !!organizationId,
    staleTime: 0,
    gcTime: 0,
    retry: false,
  });
};

// A run's log. The key carries the run's status, so the log is read once more
// when the run ends, and it polls only while the run is active.
export const useRunLogs = (token: string | null, organizationId: string | null, run: Run | null) => {
  const runId = run?.id ?? null;
  return useQuery({
    queryKey: [...deploymentKeys.runLogs(organizationId ?? "", runId ?? ""), run?.status ?? ""],
    queryFn: () => deploymentsApi.runLogs(token as string, organizationId as string, runId as string),
    enabled: !!token && !!organizationId && !!runId,
    staleTime: 0,
    retry: false,
    refetchInterval: isActive(run) ? RUN_POLL_MS : false,
  });
};

// After a save, every deployment's compatibility and preview may change, and
// so does the saved diagram's version each deployment list row carries
// (diagram_version), which "Update" compares with the last deploy's
// (changesNotDeployed). Without the list refetched, a save left "Up to date"
// on until something else refetched it.
export const useInvalidateDeploymentChecks = () => {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: [...deploymentKeys.base, "compatibility"] });
    qc.removeQueries({ queryKey: [...deploymentKeys.base, "preview"] });
    void qc.invalidateQueries({ queryKey: [...deploymentKeys.base, "diagram"] });
    refreshOrganizationLists(qc);
  };
};

export const useCreateDeployment = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { organizationId: string; data: CreateDeploymentInput }) =>
      deploymentsApi.create(token as string, input.organizationId, input.data),
    onSuccess: (_created, vars) => {
      void qc.invalidateQueries({ queryKey: deploymentKeys.forDiagram(vars.organizationId, vars.data.diagram_id) });
    },
    onSettled: () => refreshOrganizationLists(qc),
  });
};

export const useUpdateDeployment = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { organizationId: string; diagramId: string; deploymentId: string; data: UpdateDeploymentInput }) =>
      deploymentsApi.update(token as string, input.organizationId, input.deploymentId, input.data),
    onSuccess: (updated, vars) => {
      qc.setQueryData<DeploymentListItem[]>(deploymentKeys.forDiagram(vars.organizationId, vars.diagramId), (prev) =>
        prev?.map((d) => (d.id === updated.id ? { ...d, ...updated } : d)),
      );
      void qc.invalidateQueries({ queryKey: deploymentKeys.forDiagram(vars.organizationId, vars.diagramId) });
      void qc.invalidateQueries({ queryKey: deploymentKeys.compatibility(vars.organizationId, vars.deploymentId) });
      qc.removeQueries({ queryKey: deploymentKeys.preview(vars.organizationId, vars.deploymentId) });
    },
    onSettled: () => refreshOrganizationLists(qc),
  });
};

// Removing a deployment frees its connection for a new deployment of the diagram.
export const useRemoveDeployment = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { organizationId: string; diagramId: string; deploymentId: string }) =>
      deploymentsApi.remove(token as string, input.organizationId, input.deploymentId),
    onSuccess: (_none, vars) => {
      // Drop the row first: a query removed while its row is still on screen
      // is fetched again at once, for an id B no longer has (a 404).
      qc.setQueryData<DeploymentListItem[]>(deploymentKeys.forDiagram(vars.organizationId, vars.diagramId), (prev) =>
        prev?.filter((d) => d.id !== vars.deploymentId),
      );
      void qc.cancelQueries({ queryKey: deploymentKeys.compatibility(vars.organizationId, vars.deploymentId) });
      qc.removeQueries({ queryKey: deploymentKeys.compatibility(vars.organizationId, vars.deploymentId) });
      qc.removeQueries({ queryKey: deploymentKeys.preview(vars.organizationId, vars.deploymentId) });
    },
    onSettled: (_none, _err, vars) => {
      void qc.invalidateQueries({ queryKey: deploymentKeys.forDiagram(vars.organizationId, vars.diagramId) });
      refreshOrganizationLists(qc);
    },
  });
};

export const useStartRun = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { organizationId: string; diagramId: string; deploymentId: string; action: RunAction }) =>
      deploymentsApi.startRun(token as string, input.organizationId, input.deploymentId, input.action),
    onSettled: (_run, _err, vars) => {
      void qc.invalidateQueries({ queryKey: deploymentKeys.forDiagram(vars.organizationId, vars.diagramId) });
      refreshOrganizationLists(qc);
    },
  });
};
