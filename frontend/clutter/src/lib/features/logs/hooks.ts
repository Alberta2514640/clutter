// lib/features/logs/hooks.ts

import { useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { logsApi } from "./api";
import { logKeys } from "./keys";
import type { LiveLogLine, LogFileItem } from "./types";

// ─── Log file list ────────────────────────────────────────────────────────────

export const useLogFiles = (
  token?: string | null,
  orgId?: string | null,
  projId?: string | null,
  diagramId?: string | null
) => {
  return useQuery<LogFileItem[]>({
    queryKey: orgId && projId && diagramId ? logKeys.files(orgId, projId, diagramId) : ["logs", "files", "disabled"],
    queryFn: () => logsApi.listFiles(token as string, orgId as string, projId as string, diagramId as string),
    enabled: !!token && !!orgId && !!projId && !!diagramId,
    staleTime: 30 * 1000,
  });
};

// ─── Log file URL ─────────────────────────────────────────────────────────────

export const useLogFileUrl = (
  token?: string | null,
  orgId?: string | null,
  projId?: string | null,
  diagramId?: string | null,
  deploymentId?: string | null,
  file?: string | null
) => {
  return useQuery<string>({
    queryKey:
      orgId && projId && diagramId && deploymentId && file
        ? logKeys.fileUrl(orgId, projId, diagramId, deploymentId, file)
        : ["logs", "fileUrl", "disabled"],
    queryFn: () =>
      logsApi.getFileUrl(
        token as string,
        orgId as string,
        projId as string,
        diagramId as string,
        deploymentId as string,
        file as string
      ),
    enabled: false, // only on explicit refetch()
    staleTime: 0,
    gcTime: 0,
  });
};

// ─── Live logs (accumulated polling) ──────────────────────────────────────────

export type UseLiveLogsOptions = {
  token?: string | null;
  orgId?: string | null;
  projId?: string | null;
  diagramId?: string | null;
  taskArn?: string | null;
  pollIntervalMs?: number;
};

/**
 * Polls the live-log endpoint and accumulates lines locally.
 *
 * Important:
 * - This hook should be used from a component keyed by taskArn so that
 *   local state resets naturally when a new deployment/task starts.
 * - Example:
 *   <LogsPanel key={taskArn ?? "no-task"} taskArn={taskArn} />
 */
export const useLiveLogsAccumulated = ({
  token,
  orgId,
  projId,
  diagramId,
  taskArn,
  pollIntervalMs = 2_000,
}: UseLiveLogsOptions) => {
  const enabled = !!token && !!orgId && !!projId && !!diagramId && !!taskArn;

  // Store all accumulated lines for this mounted task instance
  const accRef = useRef<LiveLogLine[]>([]);
  const [lines, setLines] = useState<LiveLogLine[]>([]);

  const query = useQuery({
    queryKey:
      orgId && projId && diagramId && taskArn
        ? logKeys.live(orgId, projId, diagramId, taskArn)
        : ["logs", "live", "disabled"],

    queryFn: async () => {
      const page = await logsApi.getLiveLogs(
        token as string,
        orgId as string,
        projId as string,
        diagramId as string,
        taskArn as string
      );

      if (page.lines.length > 0) {
        const seen = new Set(accRef.current.map((line) => `${line.timestamp}-${line.message}`));

        const freshLines = page.lines.filter((line) => !seen.has(`${line.timestamp}-${line.message}`));

        if (freshLines.length > 0) {
          accRef.current = [...accRef.current, ...freshLines];
          setLines([...accRef.current]);
        }
      }

      return page;
    },

    enabled,
    staleTime: 0,
    structuralSharing: false,

    refetchInterval: (query) => {
      if (!enabled) return false;
      if (query.state.data?.isComplete) return false;
      return pollIntervalMs;
    },

    refetchIntervalInBackground: true,
  });

  return {
    lines,
    isComplete: query.data?.isComplete ?? false,
    taskStatus: query.data?.taskStatus ?? "",
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
  };
};
