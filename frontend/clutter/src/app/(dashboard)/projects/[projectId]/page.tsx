import { redirect } from "next/navigation";

// A project opens on its Diagrams tab. The redirect runs on the server, so
// the address never renders a blank page.
export default async function ProjectIndexPage({ params }: { params: Promise<{ projectId: string }> }) {
  const { projectId } = await params;
  redirect(`/projects/${encodeURIComponent(projectId)}/diagrams`);
}
