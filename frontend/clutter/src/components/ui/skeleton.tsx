import { cn } from "~/lib/utils"

// A placeholder the size of the content it stands for, so nothing moves when
// the content arrives. A light band sweeps across it; under reduced motion it
// is a still block. Put role="status" and a hidden label on the region it fills.
function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      aria-hidden="true"
      className={cn(
        "bg-muted relative overflow-hidden rounded-md",
        "after:absolute after:inset-0 after:-translate-x-full after:animate-shimmer after:bg-[linear-gradient(90deg,transparent,var(--shimmer),transparent)] after:content-['']",
        className
      )}
      {...props}
    />
  )
}

export { Skeleton }
