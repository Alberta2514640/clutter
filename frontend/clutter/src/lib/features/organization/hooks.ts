// lib/features/organization/hooks.ts

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { availableUsersApi, membersApi, organizationApi } from "./api";
import { orgKeys } from "./keys";
import type { Organization, OrgMember } from "./types";

// -------- Queries --------

export const useOrganization = () => {
  return useQuery({
    queryKey: orgKeys.organization(),
    queryFn: organizationApi.get,
    staleTime: 5 * 60 * 1000,
  });
};

export const useMembers = () => {
  return useQuery({
    queryKey: orgKeys.members(),
    queryFn: membersApi.list,
  });
};

export const useAvailableUsers = () => {
  return useQuery({
    queryKey: orgKeys.availableUsers(),
    queryFn: availableUsersApi.list,
  });
};

// -------- Mutations --------

export const useUpdateOrganization = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Organization>) => organizationApi.update(data),
    onSuccess: (updated) => {
      qc.setQueryData(orgKeys.organization(), updated);
    },
  });
};

export const useDeleteOrganization = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => organizationApi.delete(),
    onSuccess: () => {
      // Clear all organization-related cached data
      qc.removeQueries({ queryKey: orgKeys.base });
    },
  });
};

export const useAddMember = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { userId: string; role: string }) => membersApi.add(input.userId, input.role),
    onSuccess: (created: OrgMember) => {
      qc.setQueryData<OrgMember[]>(orgKeys.members(), (prev) => [...(prev ?? []), created]);
    },
  });
};

export const useRemoveMember = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (memberId: string) => membersApi.remove(memberId),
    onSuccess: (_void, memberId) => {
      qc.setQueryData<OrgMember[]>(orgKeys.members(), (prev) => (prev ?? []).filter((m) => m.id !== memberId));
    },
  });
};

export const useUpdateMemberRole = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { memberId: string; role: string }) => membersApi.updateRole(input.memberId, input.role),
    onSuccess: (updatedMember, input) => {
      qc.setQueryData<OrgMember[]>(orgKeys.members(), (prev) =>
        (prev ?? []).map((m) => (m.id === input.memberId ? updatedMember : m))
      );
    },
  });
};
