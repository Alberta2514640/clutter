"use client";

import { EmptyState } from "../_components/EmptyState";
import { useProjectContext } from "../_contexts/ProjectContext";

export default function CredentialsPage() {
  const { user } = useProjectContext();
  const firstName = user?.displayName?.split(" ")[0] || "there";
  
  const handleAddCredential = () => {
    // TODO: Implement credential creation
    console.log("Add credential clicked");
  };
  
  return (
    <EmptyState
      icon="🔑"
      title={`${firstName}, let's set up a credential`}
      description="Credentials let workflows interact with your apps and services"
      actionLabel="Add first credential"
      onAction={handleAddCredential}
    />
  );
}