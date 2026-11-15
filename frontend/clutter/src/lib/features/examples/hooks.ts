import { useQuery } from "@tanstack/react-query";
import { examplesApi } from "./api";
import { exampleKeys } from "./keys";

// The examples are built into B, so they change only when B is redeployed.
export const useExamples = (token?: string | null, enabled = true) => {
  return useQuery({
    queryKey: exampleKeys.list(),
    queryFn: () => examplesApi.list(token as string),
    enabled: !!token && enabled,
    staleTime: 10 * 60 * 1000,
  });
};
