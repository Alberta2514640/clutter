// src/lib/features/user/hooks.ts
//to do this need to properly reflect the changes from google auth
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { membersApi } from "../members/api";
import { clearTokenCookie, userApi } from "./api";
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
    onSuccess: async (user) => {
      qc.setQueryData(userKeys.me(), user);
      await membersApi.bootstrapWithMe(user); // placeholder for mock future deletion probably
    },
  });
}

// LOGOUT helper (clear storage + cache)
export function useLogout() {
  const qc = useQueryClient();

  return () => {
    clearTokenCookie();
    qc.setQueryData(userKeys.me(), null);
    qc.clear();
  };
}
