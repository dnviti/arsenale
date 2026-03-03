import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface UiPreferences {
  rdpFileBrowserOpen: boolean;
  sshSftpBrowserOpen: boolean;
  sshSftpTransferQueueOpen: boolean;
  sidebarFavoritesOpen: boolean;
  sidebarRecentsOpen: boolean;
  sidebarSharedOpen: boolean;
  sidebarCompact: boolean;
  sidebarTeamSections: Record<string, boolean>;
  settingsActiveTab: string;
  keychainScopeFilter: string;
  keychainTypeFilter: string;
  keychainSortBy: string;
  orchestrationDashboardTab: string;
  orchestrationAutoRefresh: boolean;
  orchestrationRefreshInterval: number;
}

interface UiPreferencesState extends UiPreferences {
  set: <K extends keyof UiPreferences>(key: K, value: UiPreferences[K]) => void;
  toggle: (key: keyof Omit<UiPreferences, 'sidebarTeamSections' | 'settingsActiveTab' | 'keychainScopeFilter' | 'keychainTypeFilter' | 'keychainSortBy' | 'orchestrationDashboardTab' | 'orchestrationRefreshInterval'>) => void;
  toggleTeamSection: (teamId: string) => void;
}

const defaults: UiPreferences = {
  rdpFileBrowserOpen: false,
  sshSftpBrowserOpen: false,
  sshSftpTransferQueueOpen: true,
  sidebarFavoritesOpen: true,
  sidebarRecentsOpen: true,
  sidebarSharedOpen: true,
  sidebarCompact: false,
  sidebarTeamSections: {},
  settingsActiveTab: 'profile',
  keychainScopeFilter: 'ALL',
  keychainTypeFilter: 'ALL',
  keychainSortBy: 'name',
  orchestrationDashboardTab: 'sessions',
  orchestrationAutoRefresh: true,
  orchestrationRefreshInterval: 10000,
};

export const useUiPreferencesStore = create<UiPreferencesState>()(
  persist(
    (set) => ({
      ...defaults,
      set: (key, value) => set({ [key]: value }),
      toggle: (key) =>
        set((state) => ({ [key]: !state[key] })),
      toggleTeamSection: (teamId) =>
        set((state) => ({
          sidebarTeamSections: {
            ...state.sidebarTeamSections,
            [teamId]: !(state.sidebarTeamSections[teamId] ?? true),
          },
        })),
    }),
    { name: 'rdm-ui-preferences' },
  ),
);
