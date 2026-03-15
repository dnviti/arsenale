import { create } from 'zustand';
import {
  AccessPolicyData, CreateAccessPolicyInput, UpdateAccessPolicyInput,
  listAccessPolicies, createAccessPolicy as createAccessPolicyApi,
  updateAccessPolicy as updateAccessPolicyApi, deleteAccessPolicy as deleteAccessPolicyApi,
} from '../api/accessPolicy.api';

interface AccessPolicyState {
  policies: AccessPolicyData[];
  loading: boolean;
  error: string | null;

  fetchPolicies: () => Promise<void>;
  createPolicy: (data: CreateAccessPolicyInput) => Promise<AccessPolicyData>;
  updatePolicy: (id: string, data: UpdateAccessPolicyInput) => Promise<void>;
  deletePolicy: (id: string) => Promise<void>;
  reset: () => void;
}

export const useAccessPolicyStore = create<AccessPolicyState>((set) => ({
  policies: [],
  loading: false,
  error: null,

  fetchPolicies: async () => {
    set({ loading: true, error: null });
    try {
      const policies = await listAccessPolicies();
      set({ policies, loading: false });
    } catch {
      set({ loading: false, error: 'Failed to load access policies' });
    }
  },

  createPolicy: async (data) => {
    const policy = await createAccessPolicyApi(data);
    set((state) => ({ policies: [policy, ...state.policies] }));
    return policy;
  },

  updatePolicy: async (id, data) => {
    const updated = await updateAccessPolicyApi(id, data);
    set((state) => ({
      policies: state.policies.map((p) => (p.id === id ? updated : p)),
    }));
  },

  deletePolicy: async (id) => {
    await deleteAccessPolicyApi(id);
    set((state) => ({
      policies: state.policies.filter((p) => p.id !== id),
    }));
  },

  reset: () => set({ policies: [], loading: false, error: null }),
}));
