export const exampleKeys = {
  base: ["examples"] as const,
  list: () => [...exampleKeys.base, "list"] as const,
};
