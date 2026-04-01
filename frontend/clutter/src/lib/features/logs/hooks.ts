// lib/features/logs/hooks.ts

import { useQueries, useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { Project } from "../projects/types";
import { logsApi } from "./api";
import { logKeys } from "./keys";
import type { LiveLogLine, LogFileItem, ProjectRecentActivityItem, RecentActivityItem } from "./types";

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

// ─── Recent activity ──────────────────────────────────────────────────────────

export const useRecentActivity = (token?: string | null, orgId?: string | null, diagramId?: string | null) => {
  return useQuery<RecentActivityItem[]>({
    queryKey:
      orgId && diagramId ? ["logs", "recent-activity", orgId, diagramId] : ["logs", "recent-activity", "disabled"],
    queryFn: () => logsApi.getRecentActivity(token as string, orgId as string, diagramId as string),
    enabled: !!token && !!orgId && !!diagramId,
    staleTime: 10 * 1000,
    refetchInterval: 10 * 1000,
    refetchIntervalInBackground: true,
  });
};

export const useProjectRecentActivity = (token?: string | null, orgId?: string | null, projects: Project[] = []) => {
  const diagramTargets = projects.flatMap((project) =>
    (project.diagrams ?? []).map((diagram) => ({
      projectId: project.id,
      projectName: project.name,
      diagramId: diagram.id,
      diagramName: diagram.name,
    }))
  );

  const queries = useQueries({
    queries: diagramTargets.map((target) => ({
      queryKey: logKeys.recentActivity(orgId ?? "", target.diagramId),
      queryFn: () => logsApi.getRecentActivity(token as string, orgId as string, target.diagramId),
      enabled: !!token && !!orgId && !!target.diagramId,
      staleTime: 10 * 1000,
      refetchInterval: 10 * 1000,
      refetchIntervalInBackground: true,
    })),
  });

  const allActivity: ProjectRecentActivityItem[] = queries.flatMap((query, index) => {
    const target = diagramTargets[index];
    const items = (query.data ?? []) as RecentActivityItem[];

    return items.map((item) => ({
      projectId: target.projectId,
      projectName: target.projectName,
      diagramId: target.diagramId,
      diagramName: target.diagramName,
      commandId: item.command_id,
      command: item.command,
      status: item.status,
      createdAt: item.created_at,
      durationSeconds: item.duration_seconds,
    }));
  });

  const latestByProject = new Map<string, ProjectRecentActivityItem>();

  for (const item of allActivity) {
    const existing = latestByProject.get(item.projectId);

    if (!existing || new Date(item.createdAt).getTime() > new Date(existing.createdAt).getTime()) {
      latestByProject.set(item.projectId, item);
    }
  }

  const data = Array.from(latestByProject.values()).sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  return {
    data,
    isLoading: queries.some((query) => query.isLoading),
    isError: queries.some((query) => query.isError),
    errors: queries.map((query) => query.error).filter(Boolean),
  };
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
