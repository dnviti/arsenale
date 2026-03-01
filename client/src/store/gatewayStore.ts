import { create } from 'zustand';
import {
  GatewayData, GatewayInput, GatewayUpdate,
  listGateways, createGateway as createGatewayApi,
  updateGateway as updateGatewayApi, deleteGateway as deleteGatewayApi,
} from '../api/gateway.api';

interface GatewayState {
  gateways: GatewayData[];
  loading: boolean;

  fetchGateways: () => Promise<void>;
  createGateway: (data: GatewayInput) => Promise<GatewayData>;
  updateGateway: (id: string, data: GatewayUpdate) => Promise<void>;
  deleteGateway: (id: string) => Promise<void>;
  reset: () => void;
}

export const useGatewayStore = create<GatewayState>((set, get) => ({
  gateways: [],
  loading: false,

  fetchGateways: async () => {
    set({ loading: true });
    try {
      const gateways = await listGateways();
      set({ gateways, loading: false });
    } catch {
      set({ loading: false });
    }
  },

  createGateway: async (data) => {
    const gateway = await createGatewayApi(data);
    await get().fetchGateways();
    return gateway;
  },

  updateGateway: async (id, data) => {
    const updated = await updateGatewayApi(id, data);
    set((state) => ({
      gateways: state.gateways.map((g) => (g.id === id ? { ...g, ...updated } : g)),
    }));
  },

  deleteGateway: async (id) => {
    await deleteGatewayApi(id);
    set((state) => ({
      gateways: state.gateways.filter((g) => g.id !== id),
    }));
  },

  reset: () => set({ gateways: [], loading: false }),
}));
