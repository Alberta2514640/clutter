import { useMutation } from "@tanstack/react-query";
import { codeUploadApi } from "./api";
import { codeUploadKeys } from "./keys";
import type { CreateLambdaCodeUploadUrlInput, UploadLambdaCodeToS3Input } from "./types";

export const useCreateLambdaCodeUploadUrl = (token?: string | null) => {
  return useMutation({
    mutationKey: codeUploadKeys.lambdaUploadUrl(),
    mutationFn: (input: CreateLambdaCodeUploadUrlInput) =>
      codeUploadApi.createLambdaCodeUploadUrl(token as string, input),
  });
};

export const useUploadLambdaCodeToS3 = () => {
  return useMutation({
    mutationKey: codeUploadKeys.lambdaS3Upload(),
    mutationFn: (input: UploadLambdaCodeToS3Input) =>
      codeUploadApi.uploadLambdaCodeToS3(input),
  });
};
