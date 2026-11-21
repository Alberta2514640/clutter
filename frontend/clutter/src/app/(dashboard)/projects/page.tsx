import { PROJECTS_PATH } from "@/lib/features/organization/home";
import { redirect } from "next/navigation";

// The dashboard lists the current organization's projects; /projects has no
// page of its own, so an old link lands there.
export default function ProjectsPage() {
  redirect(PROJECTS_PATH);
}
