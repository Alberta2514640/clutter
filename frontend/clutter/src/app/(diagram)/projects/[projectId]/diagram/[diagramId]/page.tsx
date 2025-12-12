"use client";

import { ReactFlowProvider } from "@xyflow/react";
import { use } from "react";
import DiagramEditor from "./_components/DiagramEditor";

type Params = { projectId: string; diagramId: string };

export default function DiagramPage({ params }: { params: Promise<Params> }) {
  const resolved = use(params);

  return (
    <ReactFlowProvider>
      <DiagramEditor projectId={resolved.projectId} diagramId={resolved.diagramId} />
    </ReactFlowProvider>
  );
}