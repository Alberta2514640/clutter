import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useInvalidateDeploymentChecks } from "../deployments/hooks";
import { functionCodeApi } from "./api";
import { functionCodeKeys } from "./keys";

// The code uploaded for a diagram's function blocks, one entry per block.
export const useFunctionCode = (token?: string | null, organizationId?: string | null, diagramId?: string | null) => {
  return useQuery({
    queryKey: functionCodeKeys.forDiagram(organizationId ?? "", diagramId ?? ""),
    queryFn: () => functionCodeApi.list(token as string, organizationId as string, diagramId as string),
    enabled: !!token && !!organizationId && !!diagramId,
    staleTime: 0,
  });
};

type UploadInput = {
  organizationId: string;
  diagramId: string;
  nodeId: string;
  file: File;
  onProgress: (loaded: number, total: number) => void;
};

// A new or removed zip changes what a preview or a run uses, so both refresh
// the diagram's list and drop its deployments' previews.
export const useUploadFunctionCode = (token?: string | null) => {
  const qc = useQueryClient();
  const invalidateChecks = useInvalidateDeploymentChecks();
  return useMutation({
    mutationFn: (v: UploadInput) =>
      functionCodeApi.upload(token as string, v.organizationId, v.diagramId, v.nodeId, v.file, v.onProgress),
    onSuccess: (_code, v) => {
      void qc.invalidateQueries({ queryKey: functionCodeKeys.forDiagram(v.organizationId, v.diagramId) });
      invalidateChecks();
    },
  });
};

export const useRemoveFunctionCode = (token?: string | null) => {
  const qc = useQueryClient();
  const invalidateChecks = useInvalidateDeploymentChecks();
  return useMutation({
    mutationFn: (v: { organizationId: string; diagramId: string; nodeId: string }) =>
      functionCodeApi.remove(token as string, v.organizationId, v.diagramId, v.nodeId),
    onSuccess: (_r, v) => {
      void qc.invalidateQueries({ queryKey: functionCodeKeys.forDiagram(v.organizationId, v.diagramId) });
      invalidateChecks();
    },
  });
};
