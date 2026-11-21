"use client";

import { tabsListClass, tabsTriggerClass } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { usePathname } from "next/navigation";

export type NavTab = {
  href: string;
  label: string;
  // Whether this tab is the current page; the default is "the path starts with href".
  isActive?: (pathname: string) => boolean;
};

// NavTabs are tabs that change the address (a project's Diagrams,
// Deployments and Settings). They look like the "line" Tabs and mark the
// current one with aria-current="page".
export default function NavTabs({ tabs, label, className }: { tabs: NavTab[]; label: string; className?: string }) {
  const pathname = usePathname() ?? "";
  // overflow-x-auto alone makes the y axis scroll too, and the active tab's
  // underline sits a pixel below the list, so a stray scrollbar shows.
  return (
    <nav aria-label={label} className={cn(tabsListClass.line, "overflow-x-auto overflow-y-hidden", className)}>
      {tabs.map((tab) => {
        const active = tab.isActive ? tab.isActive(pathname) : pathname === tab.href || pathname.startsWith(`${tab.href}/`);
        return (
          <Link key={tab.href} href={tab.href} aria-current={active ? "page" : undefined} className={tabsTriggerClass.line}>
            {tab.label}
          </Link>
        );
      })}
    </nav>
  );
}
