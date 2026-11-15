import type {
  CreateLambdaCodeUploadUrlInput,
  CreateLambdaCodeUploadUrlResponse,
  UploadLambdaCodeToS3Input,
} from "./types";
import { apiFetch } from "../user/session";

export const codeUploadApi = {
  // POST /terraform-engine/code-upload/presigned-url
  createLambdaCodeUploadUrl: async (
    token: string,
    input: CreateLambdaCodeUploadUrlInput
  ): Promise<CreateLambdaCodeUploadUrlResponse> => {
    return apiFetch<CreateLambdaCodeUploadUrlResponse>(
      "/terraform-engine/code-upload/presigned-url",
      token,
      {
        method: "POST",
        body: JSON.stringify(input),
      }
    );
  },

  // PUT {{upload_url}} to S3 (lambda zip)
  uploadLambdaCodeToS3: async (input: UploadLambdaCodeToS3Input): Promise<void> => {
    const res = await fetch(input.upload_url, {
      method: "PUT",
      headers: {
        "x-amz-server-side-encryption": "AES256",
        "Content-Type": "application/zip",
      },
      body: input.file,
    });

    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(text || `S3 upload failed: ${res.status}`);
    }
  },
};
