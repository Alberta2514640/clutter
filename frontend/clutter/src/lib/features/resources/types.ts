export type ResourceVariable = {
  name: string;
  type: "string" | "number" | "boolean" | "map";
  required: boolean;
  default: string | number | boolean | Record<string, string> | null;
  description: string;
};

export type SupportedResource = {
  id: string;
  label: string;
  displayName: string;
  description?: string;
  variables: ResourceVariable[];
  createdAt: string;
};

export type ApiEnvelope<T> = { data: T; message?: string };
