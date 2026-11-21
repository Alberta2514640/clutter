"use client"

import * as React from "react"
import * as TabsPrimitive from "@radix-ui/react-tabs"

import { cn } from "~/lib/utils"

// Tabs that switch panels on one page. "line" (the default) underlines the
// chosen tab, for a page's sections; "pill" is a segmented control, for a
// small choice inside a panel. Tabs that change the address are NavTabs
// (components/common/NavTabs.tsx), which share these styles.

type TabsVariant = "line" | "pill"

const VariantContext = React.createContext<TabsVariant>("line")

const tabsListClass: Record<TabsVariant, string> = {
  line: "border-border inline-flex w-full items-center gap-1 border-b",
  pill: "bg-muted text-muted-foreground inline-flex h-8 w-fit items-center justify-center rounded-lg p-1",
}

const tabsTriggerClass: Record<TabsVariant, string> = {
  line: "text-muted-foreground hover:text-foreground data-[state=active]:text-foreground aria-[current=page]:text-foreground relative -mb-px inline-flex h-9 items-center justify-center gap-2 border-b-2 border-transparent px-3 text-[13px] font-medium whitespace-nowrap transition-[color,border-color] motion-base outline-none focus-visible:ring-ring/50 focus-visible:ring-[3px] rounded-t-md data-[state=active]:border-primary aria-[current=page]:border-primary disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  pill: "text-muted-foreground hover:text-foreground data-[state=active]:bg-background data-[state=active]:text-foreground dark:data-[state=active]:bg-input/30 inline-flex h-full flex-1 items-center justify-center gap-1.5 rounded-md px-3 text-[13px] font-medium whitespace-nowrap transition-[color,background-color,box-shadow] motion-base outline-none focus-visible:ring-ring/50 focus-visible:ring-[3px] data-[state=active]:ring-1 data-[state=active]:ring-border disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
}

function Tabs({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return <TabsPrimitive.Root data-slot="tabs" className={cn("flex flex-col gap-4", className)} {...props} />
}

function TabsList({
  className,
  variant = "line",
  ...props
}: React.ComponentProps<typeof TabsPrimitive.List> & { variant?: TabsVariant }) {
  return (
    <VariantContext.Provider value={variant}>
      <TabsPrimitive.List data-slot="tabs-list" data-variant={variant} className={cn(tabsListClass[variant], className)} {...props} />
    </VariantContext.Provider>
  )
}

function TabsTrigger({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Trigger>) {
  const variant = React.useContext(VariantContext)
  return <TabsPrimitive.Trigger data-slot="tabs-trigger" className={cn(tabsTriggerClass[variant], className)} {...props} />
}

function TabsContent({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="tabs-content"
      className={cn("flex-1 outline-none data-[state=active]:animate-fade-in", className)}
      {...props}
    />
  )
}

export { Tabs, TabsContent, TabsList, TabsTrigger, tabsListClass, tabsTriggerClass }
