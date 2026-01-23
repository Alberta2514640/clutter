// src/lib/stores/userStore.ts
import { create } from "zustand";

// User types (moved from API client)
export interface UserTenant {
  tenantId: string;
  name: string;
}

export interface UserData {
  userId: string;
  tenantId: string | null;
  email: string;
  displayName: string;
  tenant?: UserTenant;
}

export interface UserDataState {
  user: UserData | null;
  isLoading: boolean;
  error: string | null;
}

export interface UserActions {
  loadUser: () => Promise<void>;
  createTenant: (name: string, tags?: string[]) => Promise<void>;
  reset: () => void;
}

export interface UserStore {
  state: UserDataState;
  actions: UserActions;
}

// Mock data
const MOCK_USER: UserData = {
  userId: "demo_user_001",
  tenantId: "t_demo_001",
  email: "demo@example.com",
  displayName: "Hamza Amar",
  tenant: {
    tenantId: "t_demo_001",
    name: "Demo Organization",
  },
};

export const useUserStore = create<UserStore>((set, get) => ({
  state: {
    user: null,
    isLoading: false,
    error: null,
  },

  actions: {
    loadUser: async () => {
      set((s) => ({
        state: { ...s.state, isLoading: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 300));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/user/profile');
        // const user = await response.json();

        set((s) => ({
          state: {
            ...s.state,
            user: MOCK_USER,
            isLoading: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isLoading: false,
            error: e instanceof Error ? e.message : "Failed to load user",
          },
        }));
      }
    },

    createTenant: async (name: string, tags?: string[]) => {
      set((s) => ({
        state: { ...s.state, isLoading: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 500));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/tenant', {
        //   method: 'POST',
        //   body: JSON.stringify({ name, tags })
        // });
        // const tenant = await response.json();

        // Reload user after creating tenant
        await get().actions.loadUser();
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isLoading: false,
            error: e instanceof Error ? e.message : "Failed to create tenant",
          },
        }));
      }
    },

    reset: () =>
      set(() => ({
        state: {
          user: null,
          isLoading: false,
          error: null,
        },
      })),
  },
}));

export const useUserState = () => useUserStore((s) => s.state);
export const useUserActions = () => useUserStore((s) => s.actions);
