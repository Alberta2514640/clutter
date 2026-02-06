"use client";

import Sidebar from "@/components/common/Sidebar";

export default function ProjectsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white">
      <Sidebar />
      <div className="flex-1">{children}</div>
    </div>
  );
}
