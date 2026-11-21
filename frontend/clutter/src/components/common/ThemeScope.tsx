"use client";

import { cn } from "@/lib/utils";
import { createContext, useContext, type ComponentProps } from "react";

// A part of the page with a fixed theme, whatever the viewer chose: the
// sidebar ("dark"), the sign-in and onboarding pages ("light"), and A's older
// editor ("a-editor": dark, with the token values it was built against).
export type Scope = "dark" | "light" | "a-editor";

const scopeClass: Record<Scope, string> = {
  dark: "dark",
  light: "light",
  "a-editor": "dark a-editor",
};

const ScopeContext = createContext<string | undefined>(undefined);

// Menus, tooltips, selects and dialogs render in a portal at the end of
// <body>, outside the scope's element. The ui primitives add this class to
// their content so they keep the theme of the place they were opened from.
export const useThemeScopeClass = () => useContext(ScopeContext);

export default function ThemeScope({ theme, className, ...props }: ComponentProps<"div"> & { theme: Scope }) {
  const scope = scopeClass[theme];
  return (
    <ScopeContext.Provider value={scope}>
      <div data-theme-scope={theme} className={cn(scope, className)} {...props} />
    </ScopeContext.Provider>
  );
}
