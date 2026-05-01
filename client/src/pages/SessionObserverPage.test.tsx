import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import SessionObserverPage from './SessionObserverPage';
import { useAuthStore } from '@/store/authStore';

const {
  restoreSessionApi,
  observeSshSession,
  observeRdpSession,
  observeVncSession,
} = vi.hoisted(() => ({
  restoreSessionApi: vi.fn(),
  observeSshSession: vi.fn(),
  observeRdpSession: vi.fn(),
  observeVncSession: vi.fn(),
}));

vi.mock('@/api/auth.api', () => ({
  restoreSessionApi,
}));

vi.mock('@/api/sessions.api', () => ({
  observeSshSession,
  observeRdpSession,
  observeVncSession,
}));

vi.mock('@/components/Terminal/SshObserverTerminal', () => ({
  default: ({ session }: { session: { sessionId: string } }) => <div data-testid="ssh-observer">ssh:{session.sessionId}</div>,
}));

vi.mock('@/components/SessionObserver/DesktopObserverViewer', () => ({
  default: ({ protocol, session }: { protocol: string; session: { sessionId: string } }) => (
    <div data-testid="desktop-observer">{protocol}:{session.sessionId}</div>
  ),
}));

function renderObserverPage(initialEntry: string) {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/session-observer/:protocol/:id" element={<SessionObserverPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('SessionObserverPage', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    useAuthStore.setState({
      accessToken: 'access-token',
      csrfToken: 'csrf-token',
      isAuthenticated: true,
    });

    restoreSessionApi.mockResolvedValue({
      accessToken: 'restored-access-token',
      csrfToken: 'restored-csrf-token',
    });
  });

  it('calls SSH observe endpoint and renders the SSH observer terminal', async () => {
    observeSshSession.mockResolvedValue({
      sessionId: 'session-ssh-1',
      token: 'observer-token',
      expiresAt: '2026-04-17T11:00:00.000Z',
      webSocketPath: '/ws/terminal',
      webSocketUrl: 'wss://localhost/ws/terminal?token=observer-token',
      mode: 'observe',
      readOnly: true,
    });

    renderObserverPage('/session-observer/ssh/session-ssh-1');

    await waitFor(() => {
      expect(observeSshSession).toHaveBeenCalledWith('session-ssh-1');
    });

    expect(observeRdpSession).not.toHaveBeenCalled();
    expect(observeVncSession).not.toHaveBeenCalled();
    expect(await screen.findByTestId('ssh-observer')).toHaveTextContent('ssh:session-ssh-1');
    expect(screen.getByText('Read-only observer')).toBeInTheDocument();
  });

  it('calls VNC observe endpoint and renders the desktop observer viewer', async () => {
    observeVncSession.mockResolvedValue({
      sessionId: 'session-vnc-1',
      protocol: 'VNC',
      token: 'desktop-token',
      expiresAt: '2026-04-17T11:00:00.000Z',
      webSocketPath: '/guacamole/',
      webSocketUrl: 'wss://localhost/guacamole/?token=desktop-token',
      readOnly: true,
    });

    renderObserverPage('/session-observer/vnc/session-vnc-1');

    await waitFor(() => {
      expect(observeVncSession).toHaveBeenCalledWith('session-vnc-1');
    });

    expect(observeSshSession).not.toHaveBeenCalled();
    expect(observeRdpSession).not.toHaveBeenCalled();
    expect(await screen.findByTestId('desktop-observer')).toHaveTextContent('VNC:session-vnc-1');
  });
});
