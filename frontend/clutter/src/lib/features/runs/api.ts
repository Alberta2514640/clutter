// src/lib/features/runs/api.ts
import type {
  ApiEnvelope,
  CreatePlaybookUploadUrlInput,
  CreatePlaybookUploadUrlResponse,
  Run,
  SubmitAnsibleJobInput,
  SubmitAnsibleJobResponse,
  UploadPlaybookFileToS3Input,
} from "./types";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;
const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

const MOCK_RUNS: Run[] = [
  {
    runId: "run_001",
    projectId: "p_web_app",
    projectName: "Web Application",
    workspaceId: "ws_prod",
    action: "apply",
    status: "SUCCESS",
    startedAt: "2025-01-20T14:00:00Z",
    endedAt: "2025-01-20T14:05:30Z",
  },
  {
    runId: "run_002",
    projectId: "p_web_app",
    projectName: "Web Application",
    workspaceId: "ws_staging",
    action: "plan",
    status: "SUCCESS",
    startedAt: "2025-01-19T10:30:00Z",
    endedAt: "2025-01-19T10:32:15Z",
  },
  {
    runId: "run_003",
    projectId: "p_data_pipeline",
    projectName: "Data Pipeline",
    workspaceId: "ws_prod",
    action: "apply",
    status: "RUNNING",
    startedAt: "2025-01-21T08:15:00Z",
  },
  {
    runId: "run_004",
    projectId: "p_monitoring",
    projectName: "Monitoring Stack",
    workspaceId: "ws_prod",
    action: "apply",
    status: "FAILED",
    startedAt: "2025-01-18T16:00:00Z",
    endedAt: "2025-01-18T16:03:45Z",
  },
];

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
  // keep tenantId param for future backend wiring
  listRecentByTenant: async (_tenantId: string): Promise<Run[]> => {
    await sleep(300);
    return clone(MOCK_RUNS);
  },

  // POST /ansible/playbooks/upload-url
  createPlaybookUploadUrl: async (
    token: string,
    input: CreatePlaybookUploadUrlInput
  ): Promise<CreatePlaybookUploadUrlResponse> => {
    const json = await apiFetch<ApiEnvelope<CreatePlaybookUploadUrlResponse>>(
      "/ansible/playbooks/upload-url",
      token,
      {
        method: "POST",
        body: JSON.stringify(input),
      }
    );

    return json.data;
  },

  // PUT {{upload_url}} to S3
  uploadPlaybookFileToS3: async (
    input: UploadPlaybookFileToS3Input
  ): Promise<void> => {
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
  submitAnsibleJob: async (
    token: string,
    input: SubmitAnsibleJobInput
  ): Promise<SubmitAnsibleJobResponse> => {
    const json = await apiFetch<ApiEnvelope<SubmitAnsibleJobResponse>>(
      "/ansible/jobs",
      token,
      {
        method: "POST",
        body: JSON.stringify(input),
      }
    );

    return json.data;
  },
};
