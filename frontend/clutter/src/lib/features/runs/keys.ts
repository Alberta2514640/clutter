export const runKeys = {
  base: ["runs"] as const,
  recent: (tenantId: string) => [...runKeys.base, "recent", tenantId] as const,
  playbookUploadUrl: () => [...runKeys.base, "playbook-upload-url"] as const,
  playbookS3Upload: () => [...runKeys.base, "playbook-s3-upload"] as const,
  ansibleJob: () => [...runKeys.base, "ansible-job"] as const,
};
