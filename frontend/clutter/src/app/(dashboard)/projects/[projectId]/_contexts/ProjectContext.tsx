"use client";

import { useProject } from "@/lib/features/projects/hooks";
import { Project } from "@/lib/features/projects/types";
import { useMe } from "@/lib/features/user/hooks";
import { UserData } from "@/lib/features/user/types";
import { createContext, ReactNode, useContext } from "react";

export interface ProjectContextValue {
  // User data
  user: UserData | null;
  userLoading: boolean;
  userError: Error | null;
  token: string | null;

  // Project scope
  organizationId: string;
  projectId: string;

  // Project data
  project: Project | null;
  projectLoading: boolean;
  projectError: Error | null;

  // Computed values
  isLoading: boolean;
  hasError: boolean;
  error: Error | null;
}

const ProjectContext = createContext<ProjectContextValue | null>(null);

interface ProjectProviderProps {
  children: ReactNode;
  organizationId: string;
  projectId: string;
}

/**
 * ProjectProvider - Centralized data fetching for project pages
 * 
 * Fetches user and project data once and shares it via Context API.
 * This eliminates duplicate API calls across components.
 * 
 * @example
 * ```tsx
 * <ProjectProvider projectId={projectId}>
 *   <YourComponents />
 * </ProjectProvider>
 * ```
 */
export function ProjectProvider({ children, organizationId,  projectId }: ProjectProviderProps) {
  // Fetch user data once
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  
  // Fetch project data once
  const projectQ = useProject(token, organizationId, projectId);
  
  const value: ProjectContextValue = {
    // User data
    user: meQ.data ?? null,
    userLoading: meQ.isLoading,
    userError: meQ.error as Error | null,
    token,
    
    // Project data
    organizationId,
    projectId,
    project: projectQ.data ?? null,
    projectLoading: projectQ.isLoading,
    projectError: projectQ.error as Error | null,
    
    // Computed values
    isLoading: meQ.isLoading || projectQ.isLoading,
    hasError: meQ.isError || projectQ.isError,
    error: (meQ.error || projectQ.error) as Error | null,
  };

  return (
    <ProjectContext.Provider value={value}>
      {children}
    </ProjectContext.Provider>
  );
}

/**
 * useProjectContext - Hook to access project context
 * 
 * Must be used within a ProjectProvider.
 * Provides access to user, project, and loading states.
 * 
 * @throws {Error} If used outside ProjectProvider
 * 
 * @example
 * ```tsx
 * function MyComponent() {
 *   const { user, project, token } = useProjectContext();
 *   // Use the data...
 * }
 * ```
 */
export function useProjectContext() {
  const context = useContext(ProjectContext);
  
  if (!context) {
    throw new Error("useProjectContext must be used within ProjectProvider");
  }
  
  return context;
}