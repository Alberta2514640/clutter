// src/lib/features/runs/api.ts
import type {
  ApiEnvelope,
  CreatePlaybookUploadUrlInput,
  CreatePlaybookUploadUrlResponse,
  SubmitAnsibleJobInput,
  SubmitAnsibleJobResponse,
  UploadPlaybookFileToS3Input,
} from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

async function apiFetch<T>(path: string, token: string, init: RequestInit = {}): Promise<T> {
  if (!API_BASE) throw new Error("API_ENDPOINT is not set");

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `${token}`,
      ...(init.headers ?? {}),
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Request failed: ${res.status}`);
  }

  return (await res.json()) as T;
}

export const runsApi = {
  // POST /ansible/playbooks/upload-url
  createPlaybookUploadUrl: async (
    token: string,
    input: CreatePlaybookUploadUrlInput
  ): Promise<CreatePlaybookUploadUrlResponse> => {
    const json = await apiFetch<ApiEnvelope<CreatePlaybookUploadUrlResponse>>("/ansible/playbooks/upload-url", token, {
      method: "POST",
      body: JSON.stringify(input),
    });

    return json.data;
  },

  // PUT {{upload_url}} to S3
  uploadPlaybookFileToS3: async (input: UploadPlaybookFileToS3Input): Promise<void> => {
    const res = await fetch(input.upload_url, {
      method: "PUT",
      headers: {
        "x-amz-server-side-encryption": "AES256",
        "Content-Type": "application/octet-stream",
      },
      body: input.file,
    });

    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(text || `S3 upload failed: ${res.status}`);
    }
  },

  // POST /ansible/jobs
  submitAnsibleJob: async (token: string, input: SubmitAnsibleJobInput): Promise<SubmitAnsibleJobResponse> => {
    const json = await apiFetch<ApiEnvelope<SubmitAnsibleJobResponse>>("/ansible/jobs", token, {
      method: "POST",
      body: JSON.stringify(input),
    });

    return json.data;
  },
};
