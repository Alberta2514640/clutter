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
