import NotFoundPanel from "@/components/common/NotFoundPanel";
import type { Metadata } from "next";

export const metadata: Metadata = { title: "Page not found" };

// notFound() on Projects or inside a project, shown in the app shell.
export default function DashboardNotFound() {
  return <NotFoundPanel />;
}
