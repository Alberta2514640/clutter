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
  description?: string;
};

// PUT /organization/{id} (backend/api/organization/update). An empty or
// missing field leaves that field as it is, so a description cannot be
// cleared. A name another of the caller's teams uses is refused with 409.
export type UpdateOrganizationInput = {
  organizationName?: string;
  description?: string;
};

export type CreateCloudFormationStackUrlInput = {
  organizationId: string;
  accountName: string;
  region?: string;
};

export type UpdateOrganizationAwsAccountInput = {
  role_arn: string;
};

export interface OrganizationAwsAccount {
  id: string;
  organization_id?: string;
  unique_id?: string;
  account_name: string;
  role_arn: string | null;
  external_id?: string;
  status: "incomplete" | "complete" | "revoked" | string;
  created_by?: string;
  created_at?: string;
}

export interface OrganizationAwsAccountsResponse {
  complete: OrganizationAwsAccount[];
  incomplete: OrganizationAwsAccount[];
}

export interface CloudFormationStackUrlResponse {
  account_id: string;
  url: string;
}

// common envelope your API seems to use
export type ApiEnvelope<T> = {
  data: T;
  message?: string;
};
