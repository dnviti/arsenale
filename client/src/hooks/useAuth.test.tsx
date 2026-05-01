import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore } from '../store/authStore';
import { emptyPermissionFlags } from '../utils/permissionFlags';
import { useAuth } from './useAuth';

const { restoreSessionApi } = vi.hoisted(() => ({
  restoreSessionApi: vi.fn(),
}));

vi.mock('../api/auth.api', () => ({
  restoreSessionApi,
}));

function Probe() {
  const location = useLocation();
  const { isAuthenticated, loading } = useAuth();

  return (
    <div>
      <div data-testid="location">{location.pathname}{location.search}</div>
      <div data-testid="authenticated">{String(isAuthenticated)}</div>
      <div data-testid="loading">{String(loading)}</div>
    </div>
  );
}

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();

    useAuthStore.setState({
      accessToken: null,
      csrfToken: 'csrf-token',
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        username: 'admin',
        avatarData: null,
        tenantId: 'tenant-1',
        tenantRole: 'OWNER',
      },
      isAuthenticated: true,
      permissions: emptyPermissionFlags(),
      permissionsLoaded: false,
      permissionsLoading: false,
      permissionsSubject: null,
    });
  });

  it('restores the access token and keeps the current route in place', async () => {
    restoreSessionApi.mockResolvedValue({
      accessToken: 'restored-token',
      csrfToken: 'restored-csrf',
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        username: 'admin',
        avatarData: null,
        tenantId: 'tenant-1',
        tenantRole: 'OWNER',
      },
    });

    render(
      <MemoryRouter initialEntries={['/probe?action=open-settings']}>
        <Routes>
          <Route path="/probe" element={<Probe />} />
          <Route path="/login" element={<div data-testid="login-page">login</div>} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false');
    });

    expect(screen.getByTestId('location')).toHaveTextContent('/probe?action=open-settings');
    expect(screen.getByTestId('authenticated')).toHaveTextContent('true');
    expect(screen.queryByTestId('login-page')).not.toBeInTheDocument();
    expect(useAuthStore.getState().accessToken).toBe('restored-token');
  });

  it('logs out after an expired restore attempt without forcing a route change', async () => {
    restoreSessionApi.mockRejectedValue({
      response: { status: 401 },
      isAxiosError: true,
    });

    render(
      <MemoryRouter initialEntries={['/probe?action=open-settings']}>
        <Routes>
          <Route path="/probe" element={<Probe />} />
          <Route path="/login" element={<div data-testid="login-page">login</div>} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false');
    });

    expect(screen.getByTestId('location')).toHaveTextContent('/probe?action=open-settings');
    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(screen.queryByTestId('login-page')).not.toBeInTheDocument();
    expect(useAuthStore.getState().accessToken).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });
});
