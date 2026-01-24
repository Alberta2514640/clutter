// lib/features/organization/types.ts

export interface OrgMember {
  id: string;
  name: string;
  email: string;
  role: string;
  userId?: string;
}

export interface Organization {
  tenantId: string;
  name: string;
  slug: string;
  timeZone: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface AvailableUser {
  id: string;
  name: string;
  email: string;
}
