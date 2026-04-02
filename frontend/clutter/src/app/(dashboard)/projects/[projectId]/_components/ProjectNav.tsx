"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useProjectContext } from "../_contexts/ProjectContext";

const navigationItems = [
  { label: "Diagrams", href: "diagrams" },
  { label: "Settings", href: "settings" },
];

export function ProjectNav() {
  const pathname = usePathname();
  const { projectId } = useProjectContext();
  
  const isActive = (href: string) => {
    // More precise active check
    return pathname.endsWith(`/${href}`) || pathname.includes(`/${href}/`);
  };
  
  return (
    <nav className="px-12 border-b border-slate-800">
      <div className="flex gap-8">
        {navigationItems.map((item) => {
          const active = isActive(item.href);
          
          return (
            <Link
              key={item.href}
              href={`/projects/${projectId}/${item.href}`}
              className={`py-4 text-sm font-medium relative transition ${
                active ? "text-teal-400" : "text-gray-400 hover:text-gray-200"
              }`}
            >
              {item.label}
              {active && (
                <span className="absolute left-0 right-0 -bottom-px h-[2px] bg-gradient-to-r from-teal-400 to-blue-500 rounded-full" />
              )}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}