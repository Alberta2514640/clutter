// src/lib/stores/organizationStore.ts
import { create } from "zustand";

// Organization types
export interface OrgMember {
  id: string;
  name: string;
  email: string;
  role: string;
  userId?: string;
}

export interface Organization {
  tenantId: string;
  name: string;
  slug: string;
  timeZone: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface OrganizationDataState {
  organization: Organization | null;
  members: OrgMember[];
  availableUsers: Array<{ id: string; name: string; email: string }>;
  isLoading: boolean;
  isSaving: boolean;
  error: string | null;
}

export interface OrganizationActions {
  loadOrganization: () => Promise<void>;
  updateOrganization: (data: Partial<Organization>) => Promise<void>;
  deleteOrganization: () => Promise<void>;
  loadMembers: () => Promise<void>;
  addMember: (userId: string, role: string) => Promise<void>;
  removeMember: (memberId: string) => Promise<void>;
  updateMemberRole: (memberId: string, role: string) => Promise<void>;
  loadAvailableUsers: () => Promise<void>;
  reset: () => void;
}

export interface OrganizationStore {
  state: OrganizationDataState;
  actions: OrganizationActions;
}

// Mock data
const MOCK_ORGANIZATION: Organization = {
  tenantId: "t_demo_001",
  name: "Demo Organization",
  slug: "demo-org",
  timeZone: "America/Edmonton",
  createdAt: "2025-01-01T00:00:00Z",
  updatedAt: "2025-01-20T10:00:00Z",
};

const MOCK_MEMBERS: OrgMember[] = [
  {
    id: "m1",
    userId: "demo_user_001",
    name: "Hamza Amar",
    email: "hamza.amar@ucalgary.ca",
    role: "Project Admin",
  },
];

const MOCK_AVAILABLE_USERS = [
  { id: "2", name: "Santiago Fuentes", email: "santiago@ucalgary.ca" },
  { id: "3", name: "Alice Johnson", email: "alice@ucalgary.ca" },
  { id: "4", name: "Bob Smith", email: "bob@ucalgary.ca" },
];

export const useOrganizationStore = create<OrganizationStore>((set, get) => ({
  state: {
    organization: null,
    members: [],
    availableUsers: [],
    isLoading: false,
    isSaving: false,
    error: null,
  },

  actions: {
    loadOrganization: async () => {
      set((s) => ({
        state: { ...s.state, isLoading: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 300));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/tenant');
        // const organization = await response.json();

        set((s) => ({
          state: {
            ...s.state,
            organization: MOCK_ORGANIZATION,
            isLoading: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isLoading: false,
            error: e instanceof Error ? e.message : "Failed to load organization",
          },
        }));
      }
    },

    updateOrganization: async (data: Partial<Organization>) => {
      set((s) => ({
        state: { ...s.state, isSaving: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 500));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/tenant', {
        //   method: 'PUT',
        //   body: JSON.stringify(data)
        // });

        set((s) => ({
          state: {
            ...s.state,
            organization: s.state.organization
              ? {
                  ...s.state.organization,
                  ...data,
                  updatedAt: new Date().toISOString(),
                }
              : null,
            isSaving: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isSaving: false,
            error: e instanceof Error ? e.message : "Failed to update organization",
          },
        }));
      }
    },

    deleteOrganization: async () => {
      set((s) => ({
        state: { ...s.state, isSaving: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 500));

        // TODO: Replace with actual API call
        // await fetch('/api/tenant', { method: 'DELETE' });

        set((s) => ({
          state: {
            ...s.state,
            organization: null,
            members: [],
            isSaving: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isSaving: false,
            error: e instanceof Error ? e.message : "Failed to delete organization",
          },
        }));
      }
    },

    loadMembers: async () => {
      set((s) => ({
        state: { ...s.state, isLoading: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 300));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/tenant/members');
        // const data = await response.json();

        set((s) => ({
          state: {
            ...s.state,
            members: MOCK_MEMBERS,
            isLoading: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isLoading: false,
            error: e instanceof Error ? e.message : "Failed to load members",
          },
        }));
      }
    },

    addMember: async (userId: string, role: string) => {
      set((s) => ({
        state: { ...s.state, isSaving: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 500));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/tenant/members', {
        //   method: 'POST',
        //   body: JSON.stringify({ userId, role })
        // });

        // Find user from available users
        const user = get().state.availableUsers.find((u) => u.id === userId);
        if (!user) throw new Error("User not found");

        const newMember: OrgMember = {
          id: `m_${Date.now()}`,
          userId: user.id,
          name: user.name,
          email: user.email,
          role,
        };

        set((s) => ({
          state: {
            ...s.state,
            members: [...s.state.members, newMember],
            isSaving: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isSaving: false,
            error: e instanceof Error ? e.message : "Failed to add member",
          },
        }));
      }
    },

    removeMember: async (memberId: string) => {
      set((s) => ({
        state: { ...s.state, isSaving: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 500));

        // TODO: Replace with actual API call
        // await fetch(`/api/tenant/members/${memberId}`, { method: 'DELETE' });

        set((s) => ({
          state: {
            ...s.state,
            members: s.state.members.filter((m) => m.id !== memberId),
            isSaving: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isSaving: false,
            error: e instanceof Error ? e.message : "Failed to remove member",
          },
        }));
      }
    },

    updateMemberRole: async (memberId: string, role: string) => {
      set((s) => ({
        state: { ...s.state, isSaving: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 500));

        // TODO: Replace with actual API call
        // await fetch(`/api/tenant/members/${memberId}`, {
        //   method: 'PUT',
        //   body: JSON.stringify({ role })
        // });

        set((s) => ({
          state: {
            ...s.state,
            members: s.state.members.map((m) => (m.id === memberId ? { ...m, role } : m)),
            isSaving: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isSaving: false,
            error: e instanceof Error ? e.message : "Failed to update member role",
          },
        }));
      }
    },

    loadAvailableUsers: async () => {
      set((s) => ({
        state: { ...s.state, isLoading: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 300));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/users/available');
        // const users = await response.json();

        set((s) => ({
          state: {
            ...s.state,
            availableUsers: MOCK_AVAILABLE_USERS,
            isLoading: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isLoading: false,
            error: e instanceof Error ? e.message : "Failed to load available users",
          },
        }));
      }
    },

    reset: () =>
      set(() => ({
        state: {
          organization: null,
          members: [],
          availableUsers: [],
          isLoading: false,
          isSaving: false,
          error: null,
        },
      })),
  },
}));

export const useOrganizationState = () => useOrganizationStore((s) => s.state);
export const useOrganizationActions = () => useOrganizationStore((s) => s.actions);
