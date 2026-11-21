// The viewer's theme choice, kept in this browser only. The inline script in
// the root layout reads it before the first paint, so a reload never flashes
// the wrong theme; ThemeProvider reads and changes it after that.

export type ThemeChoice = "light" | "dark" | "system";
export type ResolvedTheme = "light" | "dark";

export const THEME_STORAGE_KEY = "clutter-theme";
// The content area is dark until the viewer chooses otherwise. The sign-in
// and onboarding pages are always light, and the home page mixes both.
export const DEFAULT_THEME: ThemeChoice = "dark";
export const DARK_QUERY = "(prefers-color-scheme: dark)";

export const isThemeChoice = (value: unknown): value is ThemeChoice =>
  value === "light" || value === "dark" || value === "system";

export function readThemeChoice(): ThemeChoice {
  try {
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
    return isThemeChoice(stored) ? stored : DEFAULT_THEME;
  } catch {
    return DEFAULT_THEME;
  }
}

export const systemPrefersDark = () => window.matchMedia(DARK_QUERY).matches;

export const resolveTheme = (choice: ThemeChoice, systemDark: boolean): ResolvedTheme =>
  choice === "dark" || (choice === "system" && systemDark) ? "dark" : "light";

// applyTheme puts the resolved theme on <html>. Transitions are held off for
// one frame so every colour changes at once instead of animating.
export function applyTheme(choice: ThemeChoice) {
  const root = document.documentElement;
  const resolved = resolveTheme(choice, systemPrefersDark());
  root.classList.add("theme-switching");
  root.classList.toggle("dark", resolved === "dark");
  root.style.colorScheme = resolved;
  window.getComputedStyle(root).getPropertyValue("color-scheme");
  window.requestAnimationFrame(() => root.classList.remove("theme-switching"));
}

// The same rule as resolveTheme and applyTheme, as a string the root layout
// inlines in <head>. It runs before React, so it cannot import them.
export const themeScript = `(function(){try{var t=localStorage.getItem(${JSON.stringify(THEME_STORAGE_KEY)});if(t!=="light"&&t!=="dark"&&t!=="system")t=${JSON.stringify(DEFAULT_THEME)};var d=t==="dark"||(t==="system"&&window.matchMedia(${JSON.stringify(DARK_QUERY)}).matches);var e=document.documentElement;if(d)e.classList.add("dark");else e.classList.remove("dark");e.style.colorScheme=d?"dark":"light"}catch(_){}})()`;
