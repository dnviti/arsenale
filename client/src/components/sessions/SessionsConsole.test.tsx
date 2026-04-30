import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import SessionsConsole from './SessionsConsole';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useAuthStore } from '@/store/authStore';
import { useGatewayStore } from '@/store/gatewayStore';
import { emptyPermissionFlags } from '@/utils/permissionFlags';

const sessionsApiMocks = vi.hoisted(() => ({
  getSessionConsole: vi.fn(),
  pauseSession: vi.fn(),
  resumeSession: vi.fn(),
  terminateSession: vi.fn(),
}));

const recordingsApiMocks = vi.hoisted(() => ({
  deleteRecording: vi.fn(),
  downloadRecordingRaw: vi.fn(),
  exportRecordingVideo: vi.fn(),
}));

vi.mock('@/api/sessions.api', async () => {
  const actual = await vi.importActual<typeof import('@/api/sessions.api')>('@/api/sessions.api');
  return {
    ...actual,
    getSessionConsole: sessionsApiMocks.getSessionConsole,
    pauseSession: sessionsApiMocks.pauseSession,
    resumeSession: sessionsApiMocks.resumeSession,
    terminateSession: sessionsApiMocks.terminateSession,
  };
});

vi.mock('@/api/recordings.api', async () => {
  const actual = await vi.importActual<typeof import('@/api/recordings.api')>('@/api/recordings.api');
  return {
    ...actual,
    deleteRecording: recordingsApiMocks.deleteRecording,
    downloadRecordingRaw: recordingsApiMocks.downloadRecordingRaw,
    exportRecordingVideo: recordingsApiMocks.exportRecordingVideo,
  };
});

vi.mock('./RecordingPlayerLauncher', () => ({
  default: ({ request }: { request: { recordingId: string } | null }) => request ? <div>Launcher {request.recordingId}</div> : null,
}));

describe('SessionsConsole', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    sessionsApiMocks.getSessionConsole.mockResolvedValue({
      scope: 'tenant',
      total: 2,
      sessions: [
        {
          id: 'session-live-1',
          userId: 'user-1',
          username: 'alice',
          email: 'alice@example.com',
          connectionId: 'conn-1',
          connectionName: 'Finance Desktop',
          connectionHost: '10.10.10.10',
          connectionPort: 3389,
          gatewayId: 'gw-1',
          gatewayName: 'Gateway Alpha',
          instanceId: null,
          instanceName: null,
          protocol: 'RDP',
          status: 'ACTIVE',
          startedAt: '2026-04-17T10:00:00.000Z',
          lastActivityAt: '2026-04-17T10:05:00.000Z',
          endedAt: null,
          durationFormatted: '5m',
          recording: { exists: false },
        },
        {
          id: 'session-closed-1',
          userId: 'user-2',
          username: 'auditor',
          email: 'auditor@example.com',
          connectionId: 'conn-2',
          connectionName: 'Ops SSH',
          connectionHost: '10.10.10.20',
          connectionPort: 22,
          gatewayId: 'gw-2',
          gatewayName: 'Gateway Beta',
          instanceId: null,
          instanceName: null,
          protocol: 'SSH',
          status: 'CLOSED',
          startedAt: '2026-04-17T09:00:00.000Z',
          lastActivityAt: '2026-04-17T09:04:00.000Z',
          endedAt: '2026-04-17T09:05:00.000Z',
          durationFormatted: '5m',
          recording: {
            exists: true,
            id: 'recording-1',
            status: 'COMPLETE',
            format: 'asciicast',
            completedAt: '2026-04-17T09:05:00.000Z',
            fileSize: 2048,
            duration: 300,
          },
        },
      ],
    });

    useAuthStore.setState({
      permissions: {
        ...emptyPermissionFlags(),
        canViewSessions: true,
        canObserveSessions: true,
        canControlSessions: false,
      },
    });

    useGatewayStore.setState({
      gateways: [],
      fetchSessionCount: vi.fn().mockResolvedValue(undefined),
    });
  });

  it('loads combined status filters from the route and shows read-only recording actions', async () => {
    render(
      <TooltipProvider>
        <MemoryRouter initialEntries={['/sessions?status=ACTIVE,PAUSED,CLOSED']}>
          <Routes>
            <Route path="/sessions" element={<SessionsConsole />} />
          </Routes>
        </MemoryRouter>
      </TooltipProvider>,
    );

    await waitFor(() => {
      expect(screen.queryByText('Loading sessions')).not.toBeInTheDocument();
    });

    expect(sessionsApiMocks.getSessionConsole).toHaveBeenCalledWith(expect.objectContaining({ status: ['ACTIVE', 'PAUSED', 'CLOSED'] }));
    expect(screen.getByText('Finance Desktop')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Pause session' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Stop session' })).not.toBeInTheDocument();

    expect(screen.getByRole('button', { name: 'Observe session', hidden: true })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Playback recording', hidden: true })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Download raw recording', hidden: true })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Export MP4', hidden: true })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Analyze recording', hidden: true })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Open recording audit trail', hidden: true })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Delete recording' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Analyze recording', hidden: true }));

    expect(screen.getByText('Launcher recording-1')).toBeInTheDocument();
  });

  it('shows live control actions for control-capable users', async () => {
    useAuthStore.setState({
      permissions: {
        ...emptyPermissionFlags(),
        canViewSessions: true,
        canObserveSessions: true,
        canControlSessions: true,
      },
    });

    render(
      <TooltipProvider>
        <MemoryRouter initialEntries={['/sessions?status=ACTIVE,PAUSED,CLOSED']}>
          <Routes>
            <Route path="/sessions" element={<SessionsConsole />} />
          </Routes>
        </MemoryRouter>
      </TooltipProvider>,
    );

    await waitFor(() => {
      expect(screen.queryByText('Loading sessions')).not.toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Pause session', hidden: true }));

    await waitFor(() => {
      expect(sessionsApiMocks.pauseSession).toHaveBeenCalledWith('session-live-1');
    });

    expect(screen.getByRole('button', { name: 'Stop session', hidden: true })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Delete recording', hidden: true })).toBeInTheDocument();
  });

  it('defaults the status filter to active and paused', async () => {
    render(
      <TooltipProvider>
        <MemoryRouter initialEntries={['/sessions']}>
          <Routes>
            <Route path="/sessions" element={<SessionsConsole />} />
          </Routes>
        </MemoryRouter>
      </TooltipProvider>,
    );

    await waitFor(() => {
      expect(sessionsApiMocks.getSessionConsole).toHaveBeenCalledWith(expect.objectContaining({ status: ['ACTIVE', 'PAUSED'] }));
    });

    expect(screen.getByRole('button', { name: /active, paused/i })).toBeInTheDocument();
  });
});
