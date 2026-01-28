// lib/features/organization/hooks.ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { organizationApi } from "./api";
import { orgKeys } from "./keys";
import type { CreateOrganizationInput, Organization, UpdateOrganizationInput } from "./types";

export const useOrganizations = (token?: string | null) => {
  return useQuery({
    queryKey: orgKeys.list(),
    queryFn: () => organizationApi.list(token as string),
    enabled: !!token,
    staleTime: 5 * 60 * 1000,
  });
};

export const useCreateOrganization = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateOrganizationInput) => organizationApi.create(token as string, input),
    onSuccess: (created) => {
      qc.setQueryData<Organization[]>(orgKeys.list(), (prev) => [created, ...(prev ?? [])]);
    },
  });
};

export const useUpdateOrganization = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { organizationId: string; data: UpdateOrganizationInput }) =>
      organizationApi.update(token as string, input.organizationId, input.data),
    onSuccess: (updated) => {
      qc.setQueryData<Organization[]>(orgKeys.list(), (prev) =>
        (prev ?? []).map((o) => (o.id === updated.id ? updated : o))
      );
    },
  });
};

export const useDeleteOrganization = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (organizationId: string) => organizationApi.delete(token as string, organizationId),
    onSuccess: (_void, organizationId) => {
      qc.setQueryData<Organization[]>(orgKeys.list(), (prev) => (prev ?? []).filter((o) => o.id !== organizationId));
      qc.removeQueries({ queryKey: ["members"] });
    },
  });
};
