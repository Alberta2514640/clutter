"use client";

import Sidebar from "@/components/common/Sidebar";

export default function ProjectsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen overflow-hidden bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white">
      <Sidebar />
      <div className="min-h-0 flex-1 overflow-y-auto">{children}</div>
    </div>
  );
}
