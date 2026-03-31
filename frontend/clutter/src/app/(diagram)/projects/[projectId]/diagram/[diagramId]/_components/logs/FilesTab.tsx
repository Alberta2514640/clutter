"use client";

import { useLogFiles } from "@/lib/features/logs/hooks";
import { useState } from "react";
import FileRow from "./FileRow";
import LogFileViewerModal from "./LogFileViewerModal";
import { FilesTabProps } from "./types";

export default function FilesTab({
  token,
  orgId,
  projectId,
  diagramId,
}: FilesTabProps) {
  const { data: files, isLoading, isError } = useLogFiles(token, orgId, projectId, diagramId);

  const [viewerOpen, setViewerOpen] = useState(false);
  const [viewerTitle, setViewerTitle] = useState("");
  const [viewerContent, setViewerContent] = useState("");
  const [viewerError, setViewerError] = useState<string | null>(null);
  const [loadingFileKey, setLoadingFileKey] = useState<string | null>(null);

  if (isLoading) {
    return <p className="text-slate-500 text-xs font-mono p-4">Loading log files…</p>;
  }

  if (isError) {
    return <p className="text-red-400 text-xs font-mono p-4">Failed to load log files.</p>;
  }

  if (!files?.length) {
    return <p className="text-slate-600 text-xs font-mono p-4">No log files found.</p>;
  }

  // Group files by deploymentId, preserving insertion order
  const grouped = files.reduce<Record<string, typeof files>>((acc, f) => {
    if (!acc[f.deploymentId]) acc[f.deploymentId] = [];
    acc[f.deploymentId].push(f);
    return acc;
  }, {});

  const handleOpenViewer = async (args: {
    key: string;
    deploymentId: string;
    file: string;
    refetchUrl: () => Promise<{ data?: string }>;
  }) => {
    const { key, deploymentId, file, refetchUrl } = args;

    setViewerOpen(true);
    setViewerTitle(`${file} — ${deploymentId}`);
    setViewerContent("");
    setViewerError(null);
    setLoadingFileKey(key);

    try {
      const { data: url } = await refetchUrl();

      if (!url) {
        throw new Error("Could not get file URL.");
      }

      const res = await fetch(url);
      if (!res.ok) {
        throw new Error(`Failed to fetch log file: ${res.status}`);
      }

      const text = await res.text();
      setViewerContent(text);
    } catch (err) {
      setViewerError(err instanceof Error ? err.message : "Failed to load log file.");
    } finally {
      setLoadingFileKey(null);
    }
  };

  return (
    <>
      <div className="h-full overflow-y-auto p-4 space-y-4">
        {Object.entries(grouped).map(([deploymentId, groupFiles]) => (
          <div key={deploymentId}>
            {/* Deployment divider */}
            <div className="flex items-center gap-2 mb-2">
              <span className="shrink-0 font-mono text-[10px] text-slate-500">
                {deploymentId}
              </span>
              <div className="h-px flex-1 bg-white/10" />
            </div>

            {/* Files in this deployment */}
            <div className="space-y-1.5">
              {groupFiles.map((f) => (
                <FileRow
                  key={`${f.deploymentId}-${f.file}`}
                  token={token}
                  orgId={orgId}
                  projectId={projectId}
                  diagramId={diagramId}
                  deploymentId={f.deploymentId}
                  file={f.file}
                  loadingFileKey={loadingFileKey}
                  onOpenViewer={handleOpenViewer}
                />
              ))}
            </div>
          </div>
        ))}
      </div>

      <LogFileViewerModal
        open={viewerOpen}
        title={viewerTitle}
        content={viewerContent}
        loading={!!loadingFileKey}
        error={viewerError}
        onClose={() => {
          setViewerOpen(false);
          setViewerTitle("");
          setViewerContent("");
          setViewerError(null);
          setLoadingFileKey(null);
        }}
      />
    </>
  );
}