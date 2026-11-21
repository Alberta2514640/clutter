export const catalogKeys = {
  base: ["catalog"] as const,
  list: () => [...catalogKeys.base, "list"] as const,
  // Every linked-account list of one organization, one per connector.
  linkedForOrg: (organizationId: string) => [...catalogKeys.base, "linked", organizationId] as const,
  linked: (organizationId: string, connector: string) => [...catalogKeys.linkedForOrg(organizationId), connector] as const,
};
