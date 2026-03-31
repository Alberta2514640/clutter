import { useLogFileUrl } from "@/lib/features/logs/hooks";

export default function FileRow({
  token,
  orgId,
  projectId,
  diagramId,
  deploymentId,
  file,
  loadingFileKey,
  onOpenViewer,
}: {
  token: string | null;
  orgId: string | null;
  projectId: string;
  diagramId: string;
  deploymentId: string;
  file: string;
  loadingFileKey: string | null;
  onOpenViewer: (args: {
    key: string;
    deploymentId: string;
    file: string;
    refetchUrl: () => Promise<{ data?: string }>;
  }) => Promise<void>;
}) {
  const key = `${deploymentId}-${file}`;
  const { refetch } = useLogFileUrl(token, orgId, projectId, diagramId, deploymentId, file);

  const isLoadingThisFile = loadingFileKey === key;

  return (
    <button
      type="button"
      onClick={() =>
        onOpenViewer({
          key,
          deploymentId,
          file,
          refetchUrl: async () => {
            const result = await refetch();
            return { data: result.data };
          },
        })
      }
      disabled={isLoadingThisFile}
      className="flex w-full items-center justify-between rounded-md border border-white/10 bg-white/5 px-3 py-2 text-left text-xs transition-colors hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
    >
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="truncate font-mono text-slate-200">{file}</span>
        <span className="tabular-nums text-slate-500">{deploymentId}</span>
      </div>

      <div className="ml-4 shrink-0 flex items-center gap-1.5 text-slate-300">
        {isLoadingThisFile ? (
          <>
            <span className="h-3 w-3 rounded-full border border-slate-400 border-t-transparent animate-spin" />
            Opening…
          </>
        ) : (
          <>
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
              <path
                d="M1.5 6h9M6 1.5v9"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
              />
            </svg>
            Open
          </>
        )}
      </div>
    </button>
  );
}