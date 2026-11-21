import IsoLoader from "@/components/brand/IsoLoader";
import { Skeleton } from "@/components/ui/skeleton";

// Sign-in and onboarding are one centred column, with the logo assembling
// above it.
export default function AuthLoading() {
  return (
    <div role="status" className="mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center gap-4 px-4">
      <span className="sr-only">Loading…</span>
      <IsoLoader size={36} className="mb-2" />
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-5 w-72 max-w-full" />
      <Skeleton className="mt-4 h-11 w-full" />
    </div>
  );
}
