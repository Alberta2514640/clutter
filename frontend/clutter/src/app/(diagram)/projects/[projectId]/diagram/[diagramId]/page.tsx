"use client";

import { ReactFlowProvider } from "@xyflow/react";
import { use } from "react";
import EditorSwitch from "./_components/EditorSwitch";

type Params = { projectId: string; diagramId: string };

export default function DiagramPage({ params }: { params: Promise<Params> }) {
  const resolved = use(params);

  return (
    <ReactFlowProvider>
      <EditorSwitch projectId={resolved.projectId} diagramId={resolved.diagramId} />
    </ReactFlowProvider>
  );
}
