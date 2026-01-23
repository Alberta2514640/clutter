"use client";
import Sidebar from "@/components/common/Sidebar";
import { Button } from "@/components/ui/button";
import { useProjectId } from "@/lib/hooks/useProjectId";
import { useProjectActions, useProjectState } from "@/lib/stores/projectStore";
import { ChevronDown, Layers } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect } from "react";

interface ProjectLayoutProps {
  children: React.ReactNode;
}

const navigationItems = [
  { label: "Diagrams", href: "diagrams" },
  { label: "Credentials", href: "credentials" },
  { label: "Variables", href: "variables" },
  { label: "Settings", href: "settings" },
];

export default function ProjectLayout({ children }: ProjectLayoutProps) {
  const pathname = usePathname();
  const projectId = useProjectId();
  
  // Get project data from store
  const { currentProject, isLoading } = useProjectState();
  const { loadProject } = useProjectActions();

  // Load project data when component mounts
  useEffect(() => {
    if (projectId && (!currentProject || currentProject.projectId !== projectId)) {
      loadProject(projectId);
    }
  }, [projectId, currentProject, loadProject]);

  const isActive = (href: string) => pathname.endsWith(href);

  // Get project name or show loading/fallback
  const projectName = currentProject?.name || "Loading...";

  return (
    <div className="flex min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white">
      <Sidebar />
      <div className="flex flex-col flex-1">
        {/* HEADER */}
        <header className="border-b border-slate-800 px-12 py-8 flex items-center justify-between">
          {/* Left Section */}
          <div className="flex items-center gap-4">
            <div className="w-14 h-14 flex items-center justify-center rounded-2xl bg-gradient-to-br from-teal-500 to-blue-600 p-[2px]">
              <div className="w-full h-full flex items-center justify-center rounded-2xl bg-slate-900">
                <Layers className="w-6 h-6 text-teal-300" />
              </div>
            </div>
            <h1 className="text-2xl font-semibold">
              {isLoading ? "Loading..." : projectName}
            </h1>
          </div>
          {/* Right Section (Buttons) */}
          <div className="flex items-center gap-2">
            <Button className="bg-gradient-to-br from-teal-600 to-blue-600 hover:opacity-90 text-white px-4">
              Create workflow
            </Button>
            <Button className="bg-slate-800 border border-slate-700 hover:border-teal-500/50 text-gray-300 px-2">
              <ChevronDown className="w-4 h-4" />
            </Button>
          </div>
        </header>
        {/* TABS */}
        <nav className="px-12 border-b border-slate-800">
          <div className="flex gap-8">
            {navigationItems.map((item) => {
              const active = isActive(item.href);
              return (
                <Link
                  key={item.href}
                  href={`/projects/${projectId}/${item.href}`}
                  className={`py-4 text-sm font-medium relative transition ${
                    active
                      ? "text-teal-400"
                      : "text-gray-400 hover:text-gray-200"
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
        {/* CONTENT AREA */}
        <main className="flex-1 px-12 py-8">{children}</main>
      </div>
    </div>
  );
}