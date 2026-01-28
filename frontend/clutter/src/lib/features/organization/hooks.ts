// lib/features/organization/hooks.ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { organizationApi } from "./api";
import { orgKeys } from "./keys";
import type { Organization } from "./types";

export const useOrganization = () => {
  return useQuery({
    queryKey: orgKeys.organization(),
    queryFn: organizationApi.get,
    staleTime: 5 * 60 * 1000,
  });
};

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
      qc.removeQueries({ queryKey: orgKeys.base });
      // also clear members cache since org is gone
      qc.removeQueries({ queryKey: ["members"] });
    },
  });
};
