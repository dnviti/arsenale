import { fireEvent, render, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DeviceAuthorizationPage from './DeviceAuthorizationPage';

vi.mock('@/components/auth/AuthLayout', () => ({
  default: ({ children, description, title }: { children: ReactNode; description: string; title: string }) => (
    <div>
      <h1>{title}</h1>
      <p>{description}</p>
      {children}
    </div>
  ),
}));

const { authorizeCliDevice, useAuth } = vi.hoisted(() => ({
  authorizeCliDevice: vi.fn(),
  useAuth: vi.fn(),
}));

vi.mock('@/api/cliAuth.api', () => ({
  authorizeCliDevice,
}));

vi.mock('@/hooks/useAuth', () => ({
  useAuth,
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location-probe">{location.pathname}{location.search}</div>;
}

function renderDevicePage(path = '/device?code=ZPFU-C9Q4') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/device" element={<DeviceAuthorizationPage />} />
        <Route path="/login" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('DeviceAuthorizationPage', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useAuth.mockReturnValue({ isAuthenticated: true, loading: false });
    authorizeCliDevice.mockResolvedValue({ message: 'Device authorized successfully' });
  });

  it('prefills the CLI code from the query string and authorizes it', async () => {
    const view = renderDevicePage();

    expect(view.getByRole('heading', { name: 'Authorize Device' })).toBeInTheDocument();
    expect(view.getByLabelText('Device code')).toHaveValue('ZPFU-C9Q4');

    fireEvent.click(view.getByRole('button', { name: 'Authorize CLI' }));

    await waitFor(() => {
      expect(authorizeCliDevice).toHaveBeenCalledWith('ZPFU-C9Q4');
    });
    expect(await view.findByText('Device authorized successfully')).toBeInTheDocument();
  });

  it('redirects unauthenticated users to login with the device URL as return path', () => {
    useAuth.mockReturnValue({ isAuthenticated: false, loading: false });

    const view = renderDevicePage();

    expect(view.getByTestId('location-probe')).toHaveTextContent(
      '/login?redirect=%2Fdevice%3Fcode%3DZPFU-C9Q4',
    );
    expect(authorizeCliDevice).not.toHaveBeenCalled();
  });
});
