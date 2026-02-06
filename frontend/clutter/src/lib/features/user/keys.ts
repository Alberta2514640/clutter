//to do this need to properly reflect the changes from google auth
export const userKeys = {
  base: ["user"] as const,
  me: () => [...userKeys.base, "me"] as const,
};
