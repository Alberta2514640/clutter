// src/lib/features/user/api.ts
//to do this need to change to reflect the new google auth situation
import type { UserData } from "./types";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;

let MOCK_USER: UserData = {
  userId: "u_demo_001",
  displayName: "Hamza Amar",
  email: "hamza.amar@ucalgary.ca",
  tenantId: "t_demo_001", // set to null to simulate onboarding state
  tenant: {
    tenantId: "t_demo_001",
    name: "Demo Organization",
  },
};

export const userApi = {
  getMe: async (): Promise<UserData> => {
    await sleep(250);
    return clone(MOCK_USER);
  },

  setTenantId: async (tenantId: string | null): Promise<UserData> => {
    await sleep(200);
    MOCK_USER = {
      ...MOCK_USER,
      tenantId,
      tenant: tenantId ? (MOCK_USER.tenant ?? { tenantId, name: "Demo Organization" }) : undefined,
    };
    return clone(MOCK_USER);
  },
};
