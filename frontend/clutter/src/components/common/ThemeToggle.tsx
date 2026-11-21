"use client";

import { cn } from "@/lib/utils";
import * as RadioGroupPrimitive from "@radix-ui/react-radio-group";
import { Monitor, Moon, Sun, type LucideIcon } from "lucide-react";
import { isThemeChoice, type ThemeChoice } from "./theme";
import { useTheme } from "./ThemeProvider";

export const THEME_OPTIONS: { value: ThemeChoice; label: string; icon: LucideIcon }[] = [
  { value: "light", label: "Light", icon: Sun },
  { value: "dark", label: "Dark", icon: Moon },
  { value: "system", label: "System", icon: Monitor },
];

// ThemeToggle is a segmented Light, Dark, System choice, for the Preferences
// page. The user menu offers the same three as menu items.
export default function ThemeToggle({ className }: { className?: string }) {
  const { theme, setTheme } = useTheme();
  return (
    <RadioGroupPrimitive.Root
      aria-label="Theme"
      value={theme}
      onValueChange={(value) => {
        if (isThemeChoice(value)) setTheme(value);
      }}
      orientation="horizontal"
      className={cn("bg-muted inline-flex h-9 items-center rounded-lg p-1", className)}>
      {THEME_OPTIONS.map(({ value, label, icon: Icon }) => (
        <RadioGroupPrimitive.Item
          key={value}
          value={value}
          className="text-muted-foreground hover:text-foreground data-[state=checked]:bg-background data-[state=checked]:text-foreground dark:data-[state=checked]:bg-input/30 inline-flex h-full items-center gap-1.5 rounded-md px-3 text-sm font-medium transition-[color,background-color,box-shadow] outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 data-[state=checked]:shadow-sm">
          <Icon className="size-4" aria-hidden="true" />
          {label}
        </RadioGroupPrimitive.Item>
      ))}
    </RadioGroupPrimitive.Root>
  );
}
