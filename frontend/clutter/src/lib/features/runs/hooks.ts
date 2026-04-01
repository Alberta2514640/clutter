import { useMutation } from "@tanstack/react-query";
import { runsApi } from "./api";
import { runKeys } from "./keys";
import type { CreatePlaybookUploadUrlInput, SubmitAnsibleJobInput, UploadPlaybookFileToS3Input } from "./types";

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
