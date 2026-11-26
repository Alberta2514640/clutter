import IsoLoader from "@/components/brand/IsoLoader";
import { Skeleton } from "@/components/ui/skeleton";

// The editor's outline while it loads: a top bar, the palette, the canvas
// (with the logo assembling in its middle) and the side panel, at their sizes.
export default function DiagramLoading() {
  return (
    <div role="status" className="flex h-full flex-col">
      <span className="sr-only">Loading the editor…</span>
      <div className="flex h-14 items-center gap-4 border-b px-4">
        <Skeleton className="h-5 w-64" />
        <div className="ml-auto flex gap-2">
          <Skeleton className="h-9 w-24" />
          <Skeleton className="h-9 w-24" />
        </div>
      </div>
      <div className="flex min-h-0 flex-1">
        <div className="hidden w-64 space-y-3 border-r p-4 lg:block">
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} className="h-12" />
          ))}
        </div>
        <div className="bg-dot-grid flex flex-1 items-center justify-center">
          <IsoLoader size={48} />
        </div>
        <div className="hidden w-[440px] space-y-3 border-l p-4 lg:block">
          <Skeleton className="h-9 w-40" />
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
        </div>
      </div>
    </div>
  );
}
