import { cn } from "@/lib/utils";

// The home page's section kicker: a mono micro-label above a heading.
export function Kicker({ children, className }: { children: React.ReactNode; className?: string }) {
  return <p className={cn("micro-label mb-4", className)}>{children}</p>;
}

// The home page's section heading.
export function SectionHeading({ id, children, className }: { id: string; children: React.ReactNode; className?: string }) {
  return (
    <h2 id={id} className={cn("text-3xl leading-[1.1] font-semibold tracking-tight text-balance md:text-5xl", className)}>
      {children}
    </h2>
  );
}
