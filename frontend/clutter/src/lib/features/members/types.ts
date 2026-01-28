// lib/features/members/types.ts
export interface OrgMember {
  id: string;
  name: string;
  email: string;
  role: string;
  userId?: string;
}

export interface AvailableUser {
  id: string;
  name: string;
  email: string;
}
