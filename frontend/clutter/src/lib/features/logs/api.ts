// lib/features/logs/api.ts

import type {
  LiveLogsEnvelope,
  LiveLogsPage,
  LogFileItem,
  LogFileUrlEnvelope,
  LogFilesEnvelope,
  RecentActivityEnvelope,
  RecentActivityItem,
} from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

async function apiFetch<T>(path: string, token: string, init: RequestInit = {}): Promise<T> {
  if (!API_BASE) throw new Error("NEXT_PUBLIC_API_ENDPOINT is not set");
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

// Only STOPPED is truly terminal. DEPROVISIONING is transitional — the task
// can still emit logs and the server may not have set isComplete yet.
// Always trust the server's isComplete as the primary stop signal.
const TERMINAL_TASK_STATUSES = new Set(["STOPPED"]);

export const logsApi = {
  // GET /terraform-engine/logs
  listFiles: async (token: string, orgId: string, projId: string, diagramId: string): Promise<LogFileItem[]> => {
    const qs = new URLSearchParams({ orgId, projId, diagramId });
    const json = await apiFetch<LogFilesEnvelope>(`/terraform-engine/logs?${qs.toString()}`, token);
    const deployments = json.data?.deployments ?? [];
    return deployments.flatMap((d) => d.files.map((file) => ({ deploymentId: d.deploymentId, file })));
  },

  // GET /terraform-engine/logs/url
  getFileUrl: async (
    token: string,
    orgId: string,
    projId: string,
    diagramId: string,
    deploymentId: string,
    file: string
  ): Promise<string> => {
    const qs = new URLSearchParams({ orgId, projId, diagramId, deploymentId, file });
    const json = await apiFetch<LogFileUrlEnvelope>(`/terraform-engine/logs/url?${qs.toString()}`, token);
    return json.data.url;
  },

  // GET /terraform-engine/logs/live
  // Pass nextToken to receive only lines newer than the last poll.
  getLiveLogs: async (
    token: string,
    orgId: string,
    projId: string,
    diagramId: string,
    taskArn: string
  ): Promise<LiveLogsPage> => {
    const params: Record<string, string> = { orgId, projId, diagramId, taskArn };
    const qs = new URLSearchParams(params);

    const json = await apiFetch<LiveLogsEnvelope>(`/terraform-engine/logs/live?${qs.toString()}`, token);

    const d = json.data;
    const isTerminal = TERMINAL_TASK_STATUSES.has(d.taskStatus ?? "");

    return {
      lines: (d.events ?? []).map((e) => ({ timestamp: e.timestamp, message: e.message })),
      // Server isComplete is the truth. STOPPED is a safety fallback for cases
      // where the server sends isComplete: false on the very last poll.
      isComplete: d.isComplete || (isTerminal && !d.nextToken),
      taskStatus: d.taskStatus ?? "",
    };
  },

  // GET /terraform-engine/logs/recent-activity
  getRecentActivity: async (token: string, orgId: string, diagramId: string): Promise<RecentActivityItem[]> => {
    const qs = new URLSearchParams({ orgId, diagramId });
    const json = await apiFetch<RecentActivityEnvelope>(
      `/terraform-engine/logs/recent-activity?${qs.toString()}`,
      token
    );
    return json.data ?? [];
  },
};
