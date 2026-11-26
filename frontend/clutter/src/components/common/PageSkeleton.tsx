import IsoLoader from "@/components/brand/IsoLoader";
import { Skeleton } from "@/components/ui/skeleton";

// PageSkeleton holds a page's place while its route loads: the logo
// assembling where PageHeader's kicker sits, a header, then a grid of cards,
// at the sizes PageHeader and a card list take.
export default function PageSkeleton({ label = "Loading…", cards = 6 }: { label?: string; cards?: number }) {
  return (
    <div role="status" aria-live="polite" className="space-y-8">
      <span className="sr-only">{label}</span>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-2">
          <IsoLoader size={20} />
          <Skeleton className="h-8 w-56" />
          <Skeleton className="h-4 w-80 max-w-full" />
        </div>
        <Skeleton className="h-9 w-32" />
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: cards }, (_, i) => (
          <Skeleton key={i} className="h-32 rounded-xl" />
        ))}
      </div>
    </div>
  );
}
