import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "~/lib/utils"

// A short status or label. Colour is never the only signal: the badge's text
// says the status, and StatusDot can sit inside it.
const badgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium whitespace-nowrap transition-colors [&>svg]:pointer-events-none [&>svg]:size-3",
  {
    variants: {
      variant: {
        neutral: "border-border bg-muted text-muted-foreground",
        // Status badges: the scale's step 3 fill and step 11 text.
        primary: "border-primary/30 bg-primary/10 text-link",
        success: "border-success/20 bg-success-subtle text-success",
        warning: "border-warning/25 bg-warning-subtle text-warning",
        danger: "border-danger/20 bg-danger-subtle text-danger",
        info: "border-info/20 bg-info-subtle text-info",
        // A tint, not a status: the examples' Advanced level (LevelBadge).
        violet: "border-tint-violet-edge bg-tint-violet text-tint-violet-fg",
        outline: "border-border text-foreground",
      },
    },
    defaultVariants: {
      variant: "neutral",
    },
  }
)

function Badge({
  className,
  variant,
  asChild = false,
  ...props
}: React.ComponentProps<"span"> & VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot : "span"
  return <Comp data-slot="badge" className={cn(badgeVariants({ variant }), className)} {...props} />
}

export { Badge, badgeVariants }
