"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useSyncExternalStore, type ReactNode } from "react";
import {
  applyTheme,
  DARK_QUERY,
  DEFAULT_THEME,
  readThemeChoice,
  resolveTheme,
  systemPrefersDark,
  THEME_STORAGE_KEY,
  type ResolvedTheme,
  type ThemeChoice,
} from "./theme";

type ThemeContextValue = {
  // What the viewer chose: light, dark, or follow the system.
  theme: ThemeChoice;
  // What the content area shows now.
  resolvedTheme: ResolvedTheme;
  setTheme: (theme: ThemeChoice) => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

// The choice made in this tab when storage refuses it (a private window).
let memoryChoice: ThemeChoice | null = null;
const listeners = new Set<() => void>();
const currentChoice = () => memoryChoice ?? readThemeChoice();

// One subscription covers the three things that change the theme: a choice in
// this tab, a choice in another tab (storage event), and the system setting.
function subscribe(onChange: () => void) {
  listeners.add(onChange);
  const media = window.matchMedia(DARK_QUERY);
  const onStorage = (event: StorageEvent) => {
    if (event.key === THEME_STORAGE_KEY || event.key === null) onChange();
  };
  media.addEventListener("change", onChange);
  window.addEventListener("storage", onStorage);
  return () => {
    listeners.delete(onChange);
    media.removeEventListener("change", onChange);
    window.removeEventListener("storage", onStorage);
  };
}

// ThemeProvider owns the theme after the first paint. The class on <html> is
// set by the inline script before paint and by applyTheme after it, never by
// a render, so hydration cannot undo it.
export function ThemeProvider({ children }: { children: ReactNode }) {
  const theme = useSyncExternalStore(subscribe, currentChoice, () => DEFAULT_THEME);
  const systemDark = useSyncExternalStore(subscribe, systemPrefersDark, () => false);
  const resolvedTheme = resolveTheme(theme, systemDark);

  // Another tab's choice, or a system change while on "system", repaints this tab.
  useEffect(() => subscribe(() => applyTheme(currentChoice())), []);

  const setTheme = useCallback((next: ThemeChoice) => {
    memoryChoice = next;
    try {
      window.localStorage.setItem(THEME_STORAGE_KEY, next);
      memoryChoice = null;
    } catch {
      // Storage is blocked: the choice lasts for this tab only.
    }
    applyTheme(next);
    listeners.forEach((listener) => listener());
  }, []);

  const value = useMemo(() => ({ theme, resolvedTheme, setTheme }), [theme, resolvedTheme, setTheme]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) throw new Error("useTheme must be used inside ThemeProvider");
  return context;
}
