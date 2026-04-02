import { useMutation, useQuery } from "@tanstack/react-query";
import { runsApi } from "./api";
import { runKeys } from "./keys";
import type {
  AnsibleJob,
  CreatePlaybookUploadUrlInput,
  SubmitAnsibleJobInput,
  UploadPlaybookFileToS3Input,
} from "./types";
import { ANSIBLE_TERMINAL_STATUSES } from "./types";

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

// ─── Ansible job status polling ───────────────────────────────────────────────

export const useAnsibleJob = (token?: string | null, jobId?: string | null, pollIntervalMs = 3_000) => {
  const query = useQuery<AnsibleJob>({
    queryKey: jobId ? runKeys.ansibleJobDetail(jobId) : ["runs", "ansible-job", "disabled"],
    queryFn: () => runsApi.getAnsibleJob(token as string, jobId as string),
    enabled: !!token && !!jobId,
    staleTime: 0,

    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (!status) return pollIntervalMs; // not loaded yet — keep polling
      if (ANSIBLE_TERMINAL_STATUSES.has(status)) return false; // done — stop
      return pollIntervalMs;
    },

    refetchIntervalInBackground: true,
  });

  const isDone = ANSIBLE_TERMINAL_STATUSES.has(query.data?.status ?? "");

  return { ...query, isDone };
};
