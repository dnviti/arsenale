import api from './client';
import type { ResolvedDlpPolicy } from './connections.api';

export type CliDesktopProtocol = 'RDP' | 'VNC';

export interface CliDesktopLaunchSession {
  protocol: CliDesktopProtocol;
  connectionId: string;
  sessionId: string;
  token: string;
  webSocketPath: string;
  controlToken: string;
  controlTokenExpiresAt: string;
  enableDrive?: boolean;
  recordingId?: string;
  dlpPolicy: ResolvedDlpPolicy;
  resolvedUsername?: string;
  resolvedDomain?: string;
}

export async function redeemCliDesktopLaunch(grant: string): Promise<CliDesktopLaunchSession> {
  const { data } = await api.post<CliDesktopLaunchSession>('/cli/connect/desktop/redeem', { grant });
  return data;
}

export async function heartbeatCliDesktopSession(sessionId: string, controlToken: string): Promise<void> {
  await api.post(`/cli/connect/desktop/${sessionId}/heartbeat`, { controlToken });
}

export async function endCliDesktopSession(sessionId: string, controlToken: string): Promise<void> {
  await api.post(`/cli/connect/desktop/${sessionId}/end`, { controlToken });
}
