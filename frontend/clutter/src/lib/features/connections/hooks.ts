import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { connectionsApi } from "./api";
import { connectionKeys } from "./keys";
import type { Connection, CreateConnectionInput } from "./types";

export const useConnections = (token?: string | null, organizationId?: string | null) => {
  return useQuery({
    queryKey: connectionKeys.list(organizationId ?? ""),
    queryFn: () => connectionsApi.list(token as string, organizationId as string),
    enabled: !!token && !!organizationId,
    staleTime: 30 * 1000,
  });
};

// B caches a cloud's lists for ten minutes; so does the page. A refusal is
// not retried: the settings fall back to typed values and say why.
export const useConnectionOptions = (token?: string | null, organizationId?: string | null, connectionId?: string | null) => {
  return useQuery({
    queryKey: connectionKeys.options(organizationId ?? "", connectionId ?? ""),
    queryFn: () => connectionsApi.options(token as string, organizationId as string, connectionId as string),
    enabled: !!token && !!organizationId && !!connectionId,
    staleTime: 10 * 60 * 1000,
    retry: false,
  });
};

export const useCreateConnection = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { organizationId: string; data: CreateConnectionInput }) =>
      connectionsApi.create(token as string, input.organizationId, input.data),
    onSuccess: (created, vars) => {
      qc.setQueryData<Connection[]>(connectionKeys.list(vars.organizationId), (prev) => [...(prev ?? []), created]);
    },
  });
};

const replace = (prev: Connection[] | undefined, next: Connection) => (prev ?? []).map((c) => (c.id === next.id ? next : c));

export const useVerifyConnection = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { organizationId: string; connectionId: string }) =>
      connectionsApi.verify(token as string, input.organizationId, input.connectionId),
    onSuccess: (updated, vars) => {
      qc.setQueryData<Connection[]>(connectionKeys.list(vars.organizationId), (prev) => replace(prev, updated));
    },
  });
};

export const useDeleteConnection = (token?: string | null) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { organizationId: string; connectionId: string }) =>
      connectionsApi.delete(token as string, input.organizationId, input.connectionId),
    onSuccess: (_void, vars) => {
      qc.setQueryData<Connection[]>(connectionKeys.list(vars.organizationId), (prev) =>
        (prev ?? []).filter((c) => c.id !== vars.connectionId),
      );
    },
  });
};
