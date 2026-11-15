// src/lib/features/user/hooks.ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { clearStoredSession, userApi } from "./api";
import { userKeys } from "./keys";

export const useMe = () => {
  return useQuery({
    queryKey: userKeys.me(),
    queryFn: userApi.getMe,
    staleTime: Infinity,
    retry: false,
  });
};

export function useLoginWithGoogle() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: userApi.loginWithGoogle,
    onSuccess: (user) => {
      qc.setQueryData(userKeys.me(), user);
    },
  });
}

// LOGOUT helper (clear storage + cache)
export function useLogout() {
  const qc = useQueryClient();

  return () => {
    clearStoredSession();
    qc.setQueryData(userKeys.me(), null);
    qc.clear();
  };
}
