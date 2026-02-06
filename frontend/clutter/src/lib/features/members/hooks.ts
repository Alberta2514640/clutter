// lib/features/members/hooks.ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { availableUsersApi, membersApi } from "./api";
import { membersKeys } from "./keys";
import type { OrgMember } from "./types";

export const useMembers = () => {
  return useQuery({
    queryKey: membersKeys.list(),
    queryFn: membersApi.list,
  });
};

export const useAvailableUsers = () => {
  return useQuery({
    queryKey: membersKeys.availableUsers(),
    queryFn: availableUsersApi.list,
  });
};

export const useAddMember = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { userId: string; role: string }) => membersApi.add(input.userId, input.role),
    onSuccess: (created: OrgMember) => {
      qc.setQueryData<OrgMember[]>(membersKeys.list(), (prev) => [...(prev ?? []), created]);
    },
  });
};

export const useRemoveMember = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (memberId: string) => membersApi.remove(memberId),
    onSuccess: (_void, memberId) => {
      qc.setQueryData<OrgMember[]>(membersKeys.list(), (prev) => (prev ?? []).filter((m) => m.id !== memberId));
    },
  });
};

export const useUpdateMemberRole = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { memberId: string; role: string }) => membersApi.updateRole(input.memberId, input.role),
    onSuccess: (updatedMember, input) => {
      qc.setQueryData<OrgMember[]>(membersKeys.list(), (prev) =>
        (prev ?? []).map((m) => (m.id === input.memberId ? updatedMember : m))
      );
    },
  });
};
