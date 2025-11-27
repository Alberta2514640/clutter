"use client";

import { usePathname } from "next/navigation";

export function useProjectId() {
  const pathname = usePathname();

  // split URL → ["", "projects", "1", "diagrams"]
  const segments = pathname.split("/");

  // projectId always lives at index 2
  const projectId = segments[2];

  return projectId || null;
}