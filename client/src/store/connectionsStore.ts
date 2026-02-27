import { create } from 'zustand';
import { ConnectionData, listConnections, updateConnection, toggleFavorite as toggleFavoriteApi } from '../api/connections.api';
import { FolderData, listFolders } from '../api/folders.api';

export type Folder = FolderData;

interface ConnectionsState {
  ownConnections: ConnectionData[];
  sharedConnections: ConnectionData[];
  folders: Folder[];
  loading: boolean;
  fetchConnections: () => Promise<void>;
  fetchFolders: () => Promise<void>;
  toggleFavorite: (connectionId: string) => Promise<void>;
  moveConnection: (connectionId: string, targetFolderId: string | null) => Promise<void>;
}

export const useConnectionsStore = create<ConnectionsState>((set, get) => ({
  ownConnections: [],
  sharedConnections: [],
  folders: [],
  loading: false,

  fetchConnections: async () => {
    set({ loading: true });
    try {
      const [connData, foldersData] = await Promise.all([
        listConnections(),
        listFolders(),
      ]);
      set({
        ownConnections: connData.own,
        sharedConnections: connData.shared,
        folders: foldersData,
        loading: false,
      });
    } catch {
      set({ loading: false });
    }
  },

  fetchFolders: async () => {
    try {
      const folders = await listFolders();
      set({ folders });
    } catch {}
  },

  toggleFavorite: async (connectionId) => {
    try {
      const result = await toggleFavoriteApi(connectionId);
      set((state) => ({
        ownConnections: state.ownConnections.map((c) =>
          c.id === result.id ? { ...c, isFavorite: result.isFavorite } : c
        ),
      }));
    } catch {
      // Silently fail; the star just does not toggle
    }
  },

  moveConnection: async (connectionId, targetFolderId) => {
    const prev = get().ownConnections;
    // Optimistic update
    set({
      ownConnections: prev.map((c) =>
        c.id === connectionId ? { ...c, folderId: targetFolderId } : c
      ),
    });
    try {
      await updateConnection(connectionId, { folderId: targetFolderId });
      await get().fetchConnections();
    } catch (err) {
      set({ ownConnections: prev });
      throw err;
    }
  },
}));
