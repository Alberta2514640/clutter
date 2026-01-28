// src/lib/features/user/hooks.ts
//to do this need to properly reflect the changes from google auth
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { userApi } from "./api";
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
    localStorage.removeItem("google_data");
    qc.setQueryData(userKeys.me(), null);
    qc.removeQueries({ queryKey: userKeys.me() });
  };
}
