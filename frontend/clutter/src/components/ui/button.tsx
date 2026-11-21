import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "~/lib/utils"

// One primary action per view; every other button is secondary, outline,
// ghost or link. The primary is the screen's main action, in teal.
// Buttons with a fill or border lift 1px on hover. Controls are 32px high,
// small ones 28px; lg stays 40px for the landing page.
const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-[13px] font-medium transition-[color,background-color,border-color,box-shadow,translate,opacity] select-none disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        primary: "bg-primary text-primary-foreground hover:bg-primary/90 hover:-translate-y-px active:translate-y-0",
        // The shadcn name, kept for the screens that pass it (A's Terraform preview among them).
        default: "bg-primary text-primary-foreground hover:bg-primary/90 hover:-translate-y-px active:translate-y-0",
        destructive:
          "bg-destructive text-danger-foreground hover:bg-destructive/90 hover:-translate-y-px active:translate-y-0 focus-visible:ring-destructive/30",
        outline:
          "border border-input bg-transparent hover:bg-accent hover:text-accent-foreground hover:-translate-y-px active:translate-y-0",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-accent hover:-translate-y-px active:translate-y-0",
        ghost:
          "hover:bg-accent hover:text-accent-foreground",
        link: "text-link underline-offset-4 hover:underline",
      },
      size: {
        default: "h-8 px-3.5 py-1.5 has-[>svg]:px-3",
        xs: "h-7 rounded-md gap-1.5 px-2.5 text-xs has-[>svg]:px-2",
        sm: "h-7 rounded-md gap-1.5 px-2.5 has-[>svg]:px-2",
        lg: "h-10 rounded-md px-6 text-sm has-[>svg]:px-4",
        icon: "size-8",
        "icon-xs": "size-7",
        "icon-sm": "size-7",
        "icon-lg": "size-10",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "default",
    },
  }
)

type ButtonProps = React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }

function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: ButtonProps) {
  const Comp = asChild ? Slot : "button"

  return (
    <Comp
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants, type ButtonProps }
