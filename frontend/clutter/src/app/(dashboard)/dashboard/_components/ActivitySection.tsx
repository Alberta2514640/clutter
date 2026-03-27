"use client";

import { useMemo } from "react";
import { useQueries } from "@tanstack/react-query";
import { Activity, FileText, FolderTree, Layers3 } from "lucide-react";
import Link from "next/link";

import { Card, CardContent } from "@/components/ui/card";
import { projectsApi } from "@/lib/features/projects/api";
import { projectKeys } from "@/lib/features/projects/keys";
import type {
  Project,
  TerraformLogNode,
  TerraformLogsResponse,
} from "@/lib/features/projects/types";

type ActivitySectionProps = {
  token: string | null;
  organizationId: string | null;
  projects: Project[];
};

type DiagramTarget = {
  projectId: string;
  diagramId: string;
};

type LogEntry = {
  id: string;
  label: string;
  path?: string;
  type: "folder" | "file" | "unknown";
};

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function inferEntryType(value: unknown, explicitType?: unknown): LogEntry["type"] {
  if (explicitType === "folder" || explicitType === "directory") return "folder";
  if (explicitType === "file") return "file";
  if (Array.isArray(value)) return "folder";
  if (isPlainObject(value)) {
    if (Array.isArray(value.children) && value.children.length > 0) return "folder";
    if (typeof value.path === "string" && value.path.includes(".")) return "file";
  }
  return "unknown";
}

function normalizeTerraformLogs(payload: TerraformLogsResponse): LogEntry[] {
  const entries: LogEntry[] = [];

  const visit = (value: unknown, fallbackLabel?: string) => {
    if (!value) return;

    if (Array.isArray(value)) {
      value.forEach((item) => visit(item));
      return;
    }

    if (typeof value === "string") {
      entries.push({
        id: value,
        label: value.split("/").pop() || value,
        path: value,
        type: value.includes(".") ? "file" : "unknown",
      });
      return;
    }

    if (!isPlainObject(value)) return;

    const node = value as TerraformLogNode;
    const label =
      (typeof node.name === "string" && node.name) ||
      (typeof node.path === "string" && node.path.split("/").filter(Boolean).pop()) ||
      fallbackLabel ||
      "Untitled";

    if (label !== "data" && label !== "message") {
      entries.push({
        id: `${label}-${entries.length}`,
        label,
        path: typeof node.path === "string" ? node.path : undefined,
        type: inferEntryType(node, node.type),
      });
    }

    if (Array.isArray(node.children)) {
      node.children.forEach((child) => visit(child));
      return;
    }

    Object.entries(node).forEach(([key, child]) => {
      if (key === "name" || key === "type" || key === "path" || key === "children") return;
      if (child == null) return;
      if (typeof child === "string" && (key === "message" || key === "data")) return;
      visit(child, key);
    });
  };

  visit(payload);

  const seen = new Set<string>();
  return entries.filter((entry) => {
    const key = `${entry.type}:${entry.path ?? entry.label}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

export default function ActivitySection({ token, organizationId, projects }: ActivitySectionProps) {
  const diagramTargets = useMemo<DiagramTarget[]>(
    () =>
      projects.flatMap((project) =>
        (project.diagrams ?? []).map((diagram) => ({
          projectId: project.id,
          diagramId: diagram.id,
        })),
      ),
    [projects],
  );

  const logQueries = useQueries({
    queries: diagramTargets.map((target) => ({
      queryKey: organizationId
        ? projectKeys.terraformLogs(organizationId, target.projectId, target.diagramId)
        : [...projectKeys.all, "terraform-logs", "missing", target.projectId, target.diagramId],
      queryFn: () =>
        projectsApi.getTerraformLogs(token as string, {
          orgId: organizationId as string,
          projId: target.projectId,
          diagramId: target.diagramId,
        }),
      enabled: !!token && !!organizationId,
      staleTime: 15 * 1000,
    })),
  });

  const projectActivity = useMemo(() => {
    let queryIndex = 0;

    return projects.map((project) => {
      const diagrams = (project.diagrams ?? []).map((diagram) => {
        const query = logQueries[queryIndex++];
        const logEntries = normalizeTerraformLogs((query?.data as TerraformLogsResponse) ?? null);

        return {
          id: diagram.id,
          name: diagram.name,
          href: `/projects/${project.id}/diagram/${diagram.id}`,
          isLoading: query?.isLoading ?? false,
          isError: query?.isError ?? false,
          error: query?.error instanceof Error ? query.error.message : null,
          logEntries,
        };
      });

      return {
        id: project.id,
        name: project.name,
        href: `/projects/${project.id}/diagrams`,
        diagrams,
        totalEntries: diagrams.reduce((sum, diagram) => sum + diagram.logEntries.length, 0),
      };
    });
  }, [logQueries, projects]);

  const hasProjects = projects.length > 0;
  const isLoading = logQueries.some((query) => query.isLoading);

  return (
    <div>
      <h2 className="mb-4 flex items-center gap-2 text-2xl font-bold">
        <Activity className="h-6 w-6 text-teal-400" />
        Terraform Logs
      </h2>

      {!hasProjects ? (
        <Card className="border-slate-800/50 bg-slate-900/50 backdrop-blur-xl">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <FolderTree className="mb-4 h-16 w-16 text-gray-600" />
            <h3 className="mb-2 text-xl font-semibold text-white">No projects yet</h3>
            <p className="text-center text-gray-400">Create a project and diagram to start collecting deployment logs.</p>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-slate-800/50 bg-slate-900/50 backdrop-blur-xl">
          <CardContent className="p-0">
            <div className="divide-y divide-slate-800">
              {projectActivity.map((project) => (
                <div key={project.id} className="p-5">
                  <div className="mb-4 flex items-start justify-between gap-4">
                    <div>
                      <Link href={project.href} className="text-lg font-semibold text-white transition hover:text-teal-300">
                        {project.name}
                      </Link>
                      <p className="mt-1 text-sm text-slate-400">
                        {project.diagrams.length} diagram{project.diagrams.length === 1 ? "" : "s"} tracked
                      </p>
                    </div>
                    <div className="rounded-full border border-teal-500/20 bg-teal-500/10 px-3 py-1 text-xs font-medium text-teal-200">
                      {project.totalEntries} log item{project.totalEntries === 1 ? "" : "s"}
                    </div>
                  </div>

                  <div className="space-y-3">
                    {project.diagrams.length === 0 ? (
                      <div className="rounded-xl border border-slate-800 bg-slate-950/50 px-4 py-3 text-sm text-slate-400">
                        No diagrams in this project yet.
                      </div>
                    ) : (
                      project.diagrams.map((diagram) => (
                        <div key={diagram.id} className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
                          <div className="mb-3 flex items-center justify-between gap-3">
                            <div className="min-w-0">
                              <Link href={diagram.href} className="truncate text-sm font-semibold text-white transition hover:text-teal-300">
                                {diagram.name}
                              </Link>
                              <p className="mt-1 text-xs text-slate-500">Deployment logs grouped by diagram</p>
                            </div>
                            <div className="rounded-full border border-slate-700 bg-slate-900 px-3 py-1 text-[11px] text-slate-300">
                              {diagram.logEntries.length} item{diagram.logEntries.length === 1 ? "" : "s"}
                            </div>
                          </div>

                          {diagram.isLoading ? (
                            <div className="text-sm text-slate-400">Loading logs…</div>
                          ) : diagram.isError ? (
                            <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 text-sm text-red-200">
                              {diagram.error ?? "Failed to load terraform logs."}
                            </div>
                          ) : diagram.logEntries.length === 0 ? (
                            <div className="text-sm text-slate-500">No terraform log folders or files found for this diagram.</div>
                          ) : (
                            <div className="grid gap-2 md:grid-cols-2">
                              {diagram.logEntries.slice(0, 6).map((entry) => (
                                <div key={entry.id} className="flex items-start gap-3 rounded-lg border border-slate-800 bg-slate-900/70 px-3 py-2">
                                  <div className="mt-0.5 text-slate-400">
                                    {entry.type === "folder" ? (
                                      <FolderTree className="h-4 w-4" />
                                    ) : entry.type === "file" ? (
                                      <FileText className="h-4 w-4" />
                                    ) : (
                                      <Layers3 className="h-4 w-4" />
                                    )}
                                  </div>
                                  <div className="min-w-0">
                                    <div className="truncate text-sm font-medium text-white">{entry.label}</div>
                                    <div className="truncate text-xs uppercase tracking-wide text-slate-500">
                                      {entry.type === "folder" ? "Folder" : entry.type === "file" ? "File" : "Log item"}
                                    </div>
                                    {entry.path ? <div className="truncate text-xs text-slate-500">{entry.path}</div> : null}
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      ))
                    )}
                  </div>
                </div>
              ))}
            </div>

            {isLoading && (
              <div className="border-t border-slate-800 px-5 py-3 text-sm text-slate-400">
                Refreshing terraform logs across projects…
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
