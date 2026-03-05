import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface User {
  id: string;
  email: string;
  username: string | null;
  avatarData: string | null;
  vaultSetupComplete?: boolean;
  tenantId?: string;
  tenantRole?: string;
}

interface AuthState {
  accessToken: string | null;
  csrfToken: string | null;
  user: User | null;
  isAuthenticated: boolean;
  setAuth: (accessToken: string, csrfToken: string, user: User) => void;
  setAccessToken: (token: string) => void;
  setCsrfToken: (token: string) => void;
  updateUser: (data: Partial<User>) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      csrfToken: null,
      user: null,
      isAuthenticated: false,
      setAuth: (accessToken, csrfToken, user) =>
        set({ accessToken, csrfToken, user, isAuthenticated: true }),
      setAccessToken: (accessToken) => set({ accessToken }),
      setCsrfToken: (csrfToken) => set({ csrfToken }),
      updateUser: (data) => {
        const current = get().user;
        if (current) set({ user: { ...current, ...data } });
      },
      logout: () =>
        set({
          accessToken: null,
          csrfToken: null,
          user: null,
          isAuthenticated: false,
        }),
    }),
    {
      name: 'rdm-auth',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        csrfToken: state.csrfToken,
      }),
    }
  )
);
