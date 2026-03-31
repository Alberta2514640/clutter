import { useMutation, useQuery } from "@tanstack/react-query";
import { runsApi } from "./api";
import { runKeys } from "./keys";
import type { CreatePlaybookUploadUrlInput, SubmitAnsibleJobInput, UploadPlaybookFileToS3Input } from "./types";

export const useRecentRuns = (tenantId?: string | null) => {
  return useQuery({
    queryKey: tenantId ? runKeys.recent(tenantId) : ["runs", "recent", "no-tenant"],
    queryFn: () => runsApi.listRecentByTenant(tenantId as string),
    enabled: !!tenantId, //  waits for user.tenantId
    staleTime: 30 * 1000,
  });
};

export const useCreatePlaybookUploadUrl = (token?: string | null) => {
  return useMutation({
    mutationKey: runKeys.playbookUploadUrl(),
    mutationFn: (input: CreatePlaybookUploadUrlInput) => runsApi.createPlaybookUploadUrl(token as string, input),
  });
};

export const useUploadPlaybookFileToS3 = () => {
  return useMutation({
    mutationKey: runKeys.playbookS3Upload(),
    mutationFn: (input: UploadPlaybookFileToS3Input) => runsApi.uploadPlaybookFileToS3(input),
  });
};

export const useSubmitAnsibleJob = (token?: string | null) => {
  return useMutation({
    mutationKey: runKeys.ansibleJob(),
    mutationFn: (input: SubmitAnsibleJobInput) => runsApi.submitAnsibleJob(token as string, input),
  });
};
