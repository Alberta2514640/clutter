// lib/features/organization/hooks.ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { catalogKeys } from "../catalog/keys";
import { connectionKeys } from "../connections/keys";
import { organizationApi } from "./api";
import { orgKeys } from "./keys";
import { sortOrganizations } from "./sort";
import { useCurrentOrganizationStore } from "./uiStore";
import type {
  CreateCloudFormationStackUrlInput,
  CreateOrganizationInput,
  OrganizationAwsAccount,
  Organization,
  UpdateOrganizationAwsAccountInput,
  UpdateOrganizationInput,
} from "./types";

export const useOrganizations = (token?: string | null) => {
  return useQuery({
    queryKey: orgKeys.list(),
    queryFn: () => organizationApi.list(token as string),
    enabled: !!token,
    staleTime: 5 * 60 * 1000,
  });
};

// The organization the viewer chose (stored per browser), or the first by
// name. organizations is in that same stable order (sortOrganizations), so the
// switcher and the fallback never change between loads. Pages about one
// project use the project's own organization instead (useProjectOrganization).
export const useCurrentOrganization = (token?: string | null) => {
  const orgsQ = useOrganizations(token);
  const chosenId = useCurrentOrganizationStore((s) => s.organizationId);
  const select = useCurrentOrganizationStore((s) => s.select);
  const organizations = useMemo(() => sortOrganizations(orgsQ.data ?? []), [orgsQ.data]);
  const organization = organizations.find((o) => o.id === chosenId) ?? organizations[0] ?? null;
  return { ...orgsQ, organizations, organization, organizationId: organization?.id ?? null, select };
};

// A new organization becomes the current one, so the pages that follow work in it.
export const useCreateOrganization = (token?: string | null) => {
  const qc = useQueryClient();
  const select = useCurrentOrganizationStore((s) => s.select);
  return useMutation({
    mutationFn: (input: CreateOrganizationInput) => organizationApi.create(token as string, input),
    onSuccess: (created) => {
      qc.setQueryData<Organization[]>(orgKeys.list(), (prev) => [created, ...(prev ?? [])]);
      select(created.id);
    },
  });
};

export const useUpdateOrganization = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { organizationId: string; data: UpdateOrganizationInput }) =>
      organizationApi.update(token as string, input.organizationId, input.data),
    // A's answer to a change that changes nothing carries no organization.
    onSuccess: (updated) => {
      if (!updated?.id) {
        void qc.invalidateQueries({ queryKey: orgKeys.list() });
        return;
      }
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
    },
  });
};

// B's linked-account lists (Connections) read the same account links, so every
// change to them refreshes those lists too (catalogKeys.linkedForOrg).

export const useCreateCloudFormationStackUrl = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationKey: orgKeys.cloudFormationStackUrl(),
    mutationFn: (input: CreateCloudFormationStackUrlInput) =>
      organizationApi.createCloudFormationStackUrl(token as string, input),
    // The call starts an unfinished link, which the link flow then resumes.
    onSuccess: (_data, input) => qc.invalidateQueries({ queryKey: orgKeys.accounts(input.organizationId) }),
  });
};

export const useOrganizationAccounts = (token?: string | null, organizationId?: string | null) => {
  return useQuery({
    queryKey: orgKeys.accounts(organizationId ?? ""),
    queryFn: () => organizationApi.listAccounts(token as string, organizationId as string),
    enabled: !!token && !!organizationId,
    staleTime: 60 * 1000,
  });
};

export const useLinkFakeAccount = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (organizationId: string) => organizationApi.linkFakeAccount(token as string, organizationId),
    onSuccess: (_accountName, organizationId) => {
      qc.invalidateQueries({ queryKey: orgKeys.accounts(organizationId) });
      qc.invalidateQueries({ queryKey: catalogKeys.linkedForOrg(organizationId) });
    },
  });
};

export const useLinkDevstack = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (input: { organizationId: string; connector: string; name: string }) =>
      organizationApi.linkDevstack(token as string, input.organizationId, input.connector, input.name),
    onSuccess: (_name, input) => {
      qc.invalidateQueries({ queryKey: connectionKeys.list(input.organizationId) });
    },
  });
};

export const useDeleteOrganizationAccount =(token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (input: { organizationId: string; accountId: string }) =>
      organizationApi.deleteAccount(token as string, input.organizationId, input.accountId),
    onSuccess: (_void, input) => {
      qc.setQueryData<OrganizationAwsAccount[]>(
        orgKeys.accounts(input.organizationId),
        (prev) => (prev ?? []).filter((account) => account.id !== input.accountId)
      );
      qc.invalidateQueries({ queryKey: catalogKeys.linkedForOrg(input.organizationId) });
    },
  });
};

export const useUpdateOrganizationAccount = (token?: string | null) => {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (input: {
      organizationId: string;
      accountId: string;
      data: UpdateOrganizationAwsAccountInput;
    }) => organizationApi.updateAccount(token as string, input.organizationId, input.accountId, input.data),
    onSuccess: (_void, input) => {
      qc.invalidateQueries({ queryKey: orgKeys.accounts(input.organizationId) });
      qc.invalidateQueries({ queryKey: catalogKeys.linkedForOrg(input.organizationId) });
    },
  });
};
