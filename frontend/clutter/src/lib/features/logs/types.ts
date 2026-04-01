// lib/features/logs/types.ts

// ─── Log file list ────────────────────────────────────────────────────────────

export type LogFileItem = {
  file: string;
  deploymentId: string;
};

type LogDeployment = {
  deploymentId: string;
  files: string[];
};

export type LogFilesEnvelope = {
  data: { deployments: LogDeployment[] };
  message?: string;
};

// ─── Log file URL ─────────────────────────────────────────────────────────────

export type LogFileUrlEnvelope = {
  data: { url: string };
  message?: string;
};

// ─── Live logs ────────────────────────────────────────────────────────────────

// Actual shape from GET /terraform-engine/logs/live
export type LiveLogEvent = {
  timestamp: number; // unix ms
  message: string;
};

export type LiveLogsEnvelope = {
  data: {
    events: LiveLogEvent[];
    isComplete: boolean;
    taskStatus: string; // "RUNNING" | "STOPPED" | etc.
    nextToken: string | null; // cursor — pass on next poll to get only new lines
    logGroup: string;
    logStream: string;
    taskArn: string;
  };
  message?: string;
};

// Normalised shape we expose to the UI
export type LiveLogLine = {
  timestamp: number; // unix ms — format in the component
  message: string;
};

export type LiveLogsPage = {
  lines: LiveLogLine[];
  isComplete: boolean;
  taskStatus: string;
};

// ─── Recent activity ──────────────────────────────────────────────────────────

export type RecentActivityItem = {
  diagram_name: string;
  command_id: string;
  command: "apply" | "destroy" | "plan" | string;
  status: string;
  created_at: string;
  duration_seconds: number | null;
};

export type RecentActivityEnvelope = {
  data: RecentActivityItem[];
  message?: string;
};

export type ProjectRecentActivityItem = {
  projectId: string;
  projectName: string;
  diagramId: string;
  diagramName: string;
  commandId: string;
  command: string;
  status: string;
  createdAt: string;
  durationSeconds: number | null;
};

export type ProjectRecentActivityHookResult = {
  data: ProjectRecentActivityItem[];
  isLoading: boolean;
  isError: boolean;
  errors: unknown[];
};
