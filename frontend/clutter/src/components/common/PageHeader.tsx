import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

type Props = {
  title: ReactNode;
  // One plain line: what this page is for.
  description?: ReactNode;
  // The page's primary action, and any secondary ones, on the right.
  actions?: ReactNode;
  // A small muted uppercase kicker above the title, such as a breadcrumb or "Settings".
  eyebrow?: ReactNode;
  // Under the title row, such as NavTabs.
  children?: ReactNode;
  className?: string;
};

// The one page header: the page's h1, a one-line description and the
// primary action. It sits at the top of the AppShell's content column.
export default function PageHeader({ title, description, actions, eyebrow, children, className }: Props) {
  return (
    <header className={cn("mb-8 space-y-6", className)}>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 space-y-1">
          {eyebrow && <div className="micro-label">{eyebrow}</div>}
          <h1 className="text-foreground truncate text-2xl leading-8 font-semibold tracking-[-0.02em]">{title}</h1>
          {description && <p className="text-muted-foreground max-w-prose text-sm">{description}</p>}
        </div>
        {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
      </div>
      {children}
    </header>
  );
}
