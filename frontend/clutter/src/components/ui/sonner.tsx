"use client"

import { Toaster as Sonner, type ToasterProps } from "sonner"

import { useTheme } from "~/components/common/ThemeProvider"

// The app's one Toaster, mounted in src/app/providers.tsx. Toasts confirm an
// action that changed data ("Project created"), slide in and fade out, and
// follow the content area's theme. Show one with toast() from this module:
// toast.success("Connection added"), toast.error("Could not save. Try again.").
function Toaster(props: ToasterProps) {
  const { resolvedTheme } = useTheme()
  return (
    <Sonner
      theme={resolvedTheme}
      position="bottom-right"
      richColors
      closeButton
      className="toaster group"
      // The close button draws sonner's 20 px circle; an invisible 24 px area
      // around it is its target (WCAG 2.5.8). The area is placed from the
      // button's padding box, inside its 1 px border, hence 3 px.
      toastOptions={{ classNames: { toast: "font-sans", description: "!text-muted-foreground", closeButton: "before:absolute before:-inset-0.75" } }}
      style={
        {
          "--normal-bg": "var(--popover)",
          "--normal-text": "var(--popover-foreground)",
          "--normal-border": "var(--border)",
          "--success-bg": "color-mix(in oklab, var(--success) 10%, var(--popover))",
          "--success-text": "var(--success)",
          "--success-border": "color-mix(in oklab, var(--success) 30%, var(--popover))",
          "--error-bg": "color-mix(in oklab, var(--danger) 10%, var(--popover))",
          "--error-text": "var(--danger)",
          "--error-border": "color-mix(in oklab, var(--danger) 30%, var(--popover))",
          "--warning-bg": "color-mix(in oklab, var(--warning) 10%, var(--popover))",
          "--warning-text": "var(--warning)",
          "--warning-border": "color-mix(in oklab, var(--warning) 30%, var(--popover))",
          "--info-bg": "color-mix(in oklab, var(--info) 10%, var(--popover))",
          "--info-text": "var(--info)",
          "--info-border": "color-mix(in oklab, var(--info) 30%, var(--popover))",
          "--border-radius": "calc(var(--radius) + 4px)",
        } as React.CSSProperties
      }
      {...props}
    />
  )
}

export { toast } from "sonner"
export { Toaster }
