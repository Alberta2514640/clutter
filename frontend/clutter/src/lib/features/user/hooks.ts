// src/lib/features/user/hooks.ts
import { useQuery } from "@tanstack/react-query";
import { userApi } from "./api";
import { userKeys } from "./keys";

export const useMe = () =>
  useQuery({
    queryKey: userKeys.me(),
    queryFn: userApi.getMe,
    staleTime: 5 * 60 * 1000,
  });
