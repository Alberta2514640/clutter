"use client";

import { useParams, useRouter } from "next/navigation";
import { useEffect } from "react";

/**
 * Redirects /projects/[projectId] to /projects/[projectId]/diagrams
 * This is the default landing page for a project
 */
export default function ProjectIndexPage() {
  const params = useParams<{ projectId: string }>();
  const router = useRouter();
  const projectId = params.projectId;
  
  useEffect(() => {
    if (projectId) {
      router.replace(`/projects/${projectId}/diagrams`);
    }
  }, [projectId, router]);
  
  // No UI needed - immediate redirect
  return null;
}