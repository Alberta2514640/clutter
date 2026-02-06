"use client";

import { EmptyState } from "../_components/EmptyState";
import { useProjectContext } from "../_contexts/ProjectContext";

export default function VariablesPage() {
  const { user } = useProjectContext();
  const firstName = user?.displayName?.split(" ")[0] || "there";
  
  const handleAddVariable = () => {
    // TODO: Implement variable creation
    console.log("Add variable clicked");
  };
  
  return (
    <EmptyState
      icon="📦"
      title={`${firstName}, let's add your first variable`}
      description="Variables store reusable values that your workflows depend on"
      actionLabel="Add variable"
      onAction={handleAddVariable}
    />
  );
}