// lib/features/organization/types.ts

export interface Organization {
  id: string;
  created_by: string;
  name: string;
  description: string | null;
  created_at: string;
  total_members: number;
  members: string[];
  total_projects: number;
  projects: unknown[];
  total_diagrams: number;
}

export type CreateOrganizationInput = {
  organizationName: string;
  description: string;
};

export type UpdateOrganizationInput = {
  description: string;
};

// common envelope your API seems to use
export type ApiEnvelope<T> = {
  data: T;
  message?: string;
};
