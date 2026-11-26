"use client";

import RouteError from "@/components/common/RouteError";

// An unexpected error on Projects or inside a project, shown in the app shell.
export default function DashboardError({ retry }: { error: unknown; retry: () => void }) {
  return <RouteError retry={retry} />;
}
