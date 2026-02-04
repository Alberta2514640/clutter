"use client";

import { useParams, useRouter } from "next/navigation";
import { useEffect } from "react";


///this whole page component is a whole mess we need to change it / organized it 
export default function ProjectIndexRedirect() {
  const { projectId } = useParams<{ projectId: string }>();
  const router = useRouter();

  useEffect(() => {
    if (!projectId) return;
    router.replace(`/projects/${projectId}/diagrams`);
  }, [projectId, router]);

  return null;
}

// "use client";

// import { useParams, useRouter } from "next/navigation";
// import { useEffect } from "react";

// export default function ProjectRedirect() {
//   const params = useParams();
//   const router = useRouter();
//   const projectId = params.projectId as string;

//   useEffect(() => {
//     router.replace(`/projects/${projectId}/diagrams`);
//   }, [projectId, router]);

//   return null; // Or a loading spinner
// }
