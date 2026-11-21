// lib/features/organization/api.ts
import type {
  ApiEnvelope,
  CloudFormationStackUrlResponse,
  CreateCloudFormationStackUrlInput,
  CreateOrganizationInput,
  OrganizationAwsAccount,
  OrganizationAwsAccountsResponse,
  Organization,
  UpdateOrganizationAwsAccountInput,
  UpdateOrganizationInput,
} from "./types";
import { apiFetch } from "../user/session";
import { localDevApi } from "./localDev";

const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

// The local dev gateway's own routes exist only here, never in a deployment.
export const isLocalDevApi = localDevApi(process.env.NODE_ENV, API_BASE);

// The region the AWS setup opens in: the region Clutter's cloud runs in
// (ADR-015). The role it creates is not tied to a region.
export const DEFAULT_AWS_REGION = "ca-central-1";

export const organizationApi = {
  // GET /organization
  // supports either {data: {...}} or {data: [{...}, ...]}
  list: async (token: string): Promise<Organization[]> => {
    const json = await apiFetch<ApiEnvelope<Organization | Organization[]>>("/organization", token, { method: "GET" });

    return Array.isArray(json.data) ? json.data : [json.data];
  },

  // POST /organization
  create: async (token: string, input: CreateOrganizationInput): Promise<Organization> => {
    const json = await apiFetch<ApiEnvelope<Organization>>("/organization", token, {
      method: "POST",
      body: JSON.stringify(input),
    });

    return json.data;
  },

  // PUT /organization/:organizationId
  update: async (token: string, organizationId: string, input: UpdateOrganizationInput): Promise<Organization> => {
    const json = await apiFetch<ApiEnvelope<Organization>>(`/organization/${organizationId}`, token, {
      method: "PUT",
      body: JSON.stringify(input),
    });

    return json.data;
  },

  // DELETE /organization/:organizationId
  delete: async (token: string, organizationId: string): Promise<void> => {
    await apiFetch<ApiEnvelope<unknown>>(`/organization/${organizationId}`, token, {
      method: "DELETE",
    });
  },

  // GET /cloudformation-stack-url/:organizationId?accountName=...&region=...
  createCloudFormationStackUrl: async (
    token: string,
    input: CreateCloudFormationStackUrlInput
  ): Promise<CloudFormationStackUrlResponse> => {
    const params = new URLSearchParams({
      accountName: input.accountName,
      region: input.region ?? DEFAULT_AWS_REGION,
    });

    const json = await apiFetch<ApiEnvelope<CloudFormationStackUrlResponse>>(
      `/cloudformation-stack-url/${input.organizationId}?${params.toString()}`,
      token,
      { method: "GET" }
    );

    return json.data;
  },

  // GET /organization/:organizationId/accounts
  listAccounts: async (token: string, organizationId: string): Promise<OrganizationAwsAccount[]> => {
    const json = await apiFetch<ApiEnvelope<OrganizationAwsAccountsResponse>>(
      `/organization/${organizationId}/accounts`,
      token,
      { method: "GET" }
    );

    return [...(json.data?.complete ?? []), ...(json.data?.incomplete ?? [])].filter(
      (account): account is OrganizationAwsAccount => !!account && typeof account === "object"
    );
  },

  // POST /dev/link-fake-account (local dev gateway only). Answers with the
  // linked account's name: a new fake one, or the one already linked.
  linkFakeAccount: async (token: string, organizationId: string): Promise<string> => {
    const json = await apiFetch<{ message: string; accountName?: string }>("/dev/link-fake-account", token, {
      method: "POST",
      body: JSON.stringify({ organizationId }),
    });

    return json.accountName ?? "";
  },

  // POST /dev/link-devstack (local dev gateway only). Makes a credential in
  // this machine's DevStack and a connection in B from it, named name, and
  // answers with the connection's name. The secret never reaches the browser.
  linkDevstack: async (token: string, organizationId: string, connector: string, name: string): Promise<string> => {
    const json = await apiFetch<{ message: string; connectionName?: string }>("/dev/link-devstack", token, {
      method: "POST",
      body: JSON.stringify({ organizationId, connector, name }),
    });

    return json.connectionName ?? "";
  },

  // DELETE /organization/:organizationId/accounts/:accountId
  deleteAccount: async (token: string, organizationId: string, accountId: string): Promise<void> => {
    await apiFetch<ApiEnvelope<unknown>>(`/organization/${organizationId}/accounts/${accountId}`, token, {
      method: "DELETE",
    });
  },

  // POST /organization/:organizationId/accounts/:accountId
  updateAccount: async (
    token: string,
    organizationId: string,
    accountId: string,
    input: UpdateOrganizationAwsAccountInput
  ): Promise<void> => {
    await apiFetch<ApiEnvelope<unknown>>(`/organization/${organizationId}/accounts/${accountId}`, token, {
      method: "POST",
      body: JSON.stringify(input),
    });
  },
};
