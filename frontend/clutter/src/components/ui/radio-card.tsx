"use client"

import * as React from "react"
import * as RadioGroupPrimitive from "@radix-ui/react-radio-group"

import { cn } from "~/lib/utils"

// Full-width option cards for one question (onboarding). The whole card is
// the radio: arrow keys move and choose, Space chooses, and Enter calls
// onEnter (the screen's Continue). The chosen card gets the primary border
// and a tinted fill. Give the group an accessible name (aria-labelledby).

function RadioCardGroup({
  className,
  onEnter,
  onKeyDown,
  ...props
}: React.ComponentProps<typeof RadioGroupPrimitive.Root> & {
  // Radix keeps Enter from choosing a radio (WAI-ARIA), so the group hands it on.
  onEnter?: () => void
}) {
  return (
    <RadioGroupPrimitive.Root
      data-slot="radio-card-group"
      className={cn("grid gap-3", className)}
      onKeyDown={(event) => {
        onKeyDown?.(event)
        if (event.key === "Enter" && onEnter) onEnter()
      }}
      {...props}
    />
  )
}

function RadioCard({ className, children, ...props }: React.ComponentProps<typeof RadioGroupPrimitive.Item>) {
  return (
    <RadioGroupPrimitive.Item
      data-slot="radio-card"
      className={cn(
        "group/radio-card surface-card surface-card-interactive text-card-foreground relative flex w-full items-center gap-4 p-4 text-left",
        "outline-none focus-visible:ring-ring/50 focus-visible:ring-[3px]",
        "data-[state=checked]:border-primary data-[state=checked]:ring-1 data-[state=checked]:ring-primary",
        "disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:translate-y-0",
        className,
      )}
      {...props}
    >
      <span className="flex min-w-0 flex-1 items-center gap-4">{children}</span>
      <span
        aria-hidden="true"
        className="border-input group-data-[state=checked]/radio-card:border-primary flex size-5 shrink-0 items-center justify-center rounded-full border transition-colors"
      >
        <RadioGroupPrimitive.Indicator className="bg-primary size-2.5 rounded-full animate-check" />
      </span>
    </RadioGroupPrimitive.Item>
  )
}

// The card's text. These are spans, because a radio is a button and holds
// phrasing content only.
function RadioCardTitle({ className, ...props }: React.ComponentProps<"span">) {
  return <span data-slot="radio-card-title" className={cn("block text-sm font-semibold", className)} {...props} />
}

function RadioCardDescription({ className, ...props }: React.ComponentProps<"span">) {
  return <span data-slot="radio-card-description" className={cn("text-muted-foreground mt-0.5 block text-sm", className)} {...props} />
}

export { RadioCard, RadioCardDescription, RadioCardGroup, RadioCardTitle }
