import type { LiveLogLine } from "@/lib/features/logs/types";

export type LogsTab = "live" | "files";

export type LiveLogsState = {
  lines: LiveLogLine[];
  isComplete: boolean;
  taskStatus: string;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
};

export type LogsPanelProps = {
  token: string | null;
  orgId: string | null;
  projectId: string;
  diagramId: string;
  taskArn: string | null;
  liveLogs: LiveLogsState;
};

export type LiveLogsTabProps = {
  taskArn: string | null;
  lines: LiveLogLine[];
  isComplete: boolean;
  taskStatus: string;
  isLoading: boolean;
};

export type FilesTabProps = {
  token: string | null;
  orgId: string | null;
  projectId: string;
  diagramId: string;
};

export type LogFileViewerModalProps = {
  open: boolean;
  title: string;
  content: string;
  loading: boolean;
  error: string | null;
  onClose: () => void;
};
