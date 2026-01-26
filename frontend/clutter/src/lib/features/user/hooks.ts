// src/lib/features/user/hooks.ts
//to do this need to properly reflect the changes from google auth
import { useQuery } from "@tanstack/react-query";
import { userApi } from "./api";
import { userKeys } from "./keys";

export const useMe = () => {
  return useQuery({
    queryKey: userKeys.me(),
    queryFn: userApi.getMe,
    staleTime: 5 * 60 * 1000,
  });
};
