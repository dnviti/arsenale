import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import SessionDashboard from './SessionDashboard';
import { useAuthStore } from '@/store/authStore';
import { useGatewayStore } from '@/store/gatewayStore';
import { emptyPermissionFlags } from '@/utils/permissionFlags';

describe('SessionDashboard', () => {
  beforeEach(() => {
    useAuthStore.setState({
      permissions: {
        ...emptyPermissionFlags(),
        canObserveSessions: true,
        canControlSessions: true,
      },
    });

    useGatewayStore.setState({
      sessionCount: 7,
    });
  });

  it('navigates to the unified sessions console', () => {
    function LocationProbe() {
      const location = useLocation();
      return <div data-testid="location-probe">{location.pathname}</div>;
    }

    render(
      <MemoryRouter initialEntries={['/settings']}>
        <Routes>
          <Route path="/settings" element={<><SessionDashboard /><LocationProbe /></>} />
          <Route path="/sessions" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText('7')).toBeInTheDocument();
    expect(screen.getByText(/Pause, resume, stop/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /open sessions console/i }));

    expect(screen.getByTestId('location-probe')).toHaveTextContent('/sessions');
  });

  it('uses the in-app opener when provided', () => {
    const onOpenSessions = vi.fn();

    render(
      <MemoryRouter>
        <SessionDashboard onOpenSessions={onOpenSessions} />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole('button', { name: /open sessions console/i }));

    expect(onOpenSessions).toHaveBeenCalledTimes(1);
  });
});
