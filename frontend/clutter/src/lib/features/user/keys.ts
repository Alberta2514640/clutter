export const userKeys = {
  base: ["user"] as const,
  me: () => [...userKeys.base, "me"] as const,
};
