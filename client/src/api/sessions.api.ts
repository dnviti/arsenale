import api from './client';
import type { ResolvedDlpPolicy } from './connections.api';
import type { SshTerminalConfig } from '../constants/terminalThemes';

export type SessionProtocol = 'SSH' | 'RDP' | 'VNC';
export type SessionConsoleStatus = 'ACTIVE' | 'IDLE' | 'PAUSED' | 'CLOSED';

export interface GatewaySessionCount {
  gatewayId: string;
  gatewayName: string;
  count: number;
}

export interface SessionControlResponse {
  ok: boolean;
  sessionId: string;
  protocol: SessionProtocol;
  status: 'ACTIVE' | 'IDLE' | 'PAUSED' | 'CLOSED';
  paused: boolean;
}

export interface SessionTerminateResponse {
  ok: boolean;
  sessionId: string;
  protocol: SessionProtocol;
  terminated: boolean;
}

export interface SessionConsoleRecording {
  exists: boolean;
  id?: string;
  status?: 'RECORDING' | 'COMPLETE' | 'ERROR';
  format?: string;
  completedAt?: string | null;
  fileSize?: number | null;
  duration?: number | null;
}

export interface SessionConsoleSession {
  id: string;
  userId: string;
  username: string | null;
  email: string;
  connectionId: string;
  connectionName: string;
  connectionHost: string;
  connectionPort: number;
  gatewayId: string | null;
  gatewayName: string | null;
  instanceId: string | null;
  instanceName: string | null;
  protocol: SessionProtocol;
  status: SessionConsoleStatus;
  startedAt: string;
  lastActivityAt: string;
  endedAt: string | null;
  durationFormatted: string;
  recording: SessionConsoleRecording;
}

export interface SessionConsoleResponse {
  scope: 'tenant' | 'own';
  total: number;
  sessions: SessionConsoleSession[];
}

export async function getSessionCount(): Promise<number> {
  const { data } = await api.get('/sessions/count');
  return data.count;
}

export async function getSessionCountByGateway(): Promise<GatewaySessionCount[]> {
  const { data } = await api.get('/sessions/count/gateway');
  return data;
}

export async function pauseSession(sessionId: string): Promise<SessionControlResponse> {
  const { data } = await api.post(`/sessions/${sessionId}/pause`);
  return data;
}

export async function resumeSession(sessionId: string): Promise<SessionControlResponse> {
  const { data } = await api.post(`/sessions/${sessionId}/resume`);
  return data;
}

export async function terminateSession(sessionId: string): Promise<SessionTerminateResponse> {
  const { data } = await api.post(`/sessions/${sessionId}/terminate`);
  return data;
}

export async function getSessionConsole(params?: {
  protocol?: SessionProtocol;
  status?: SessionConsoleStatus[];
  gatewayId?: string;
  limit?: number;
  offset?: number;
}): Promise<SessionConsoleResponse> {
  const { data } = await api.get('/sessions/console', {
    params: {
      protocol: params?.protocol,
      status: params?.status?.length ? params.status.join(',') : undefined,
      gatewayId: params?.gatewayId,
      limit: params?.limit,
      offset: params?.offset,
    },
  });
  return data;
}

// ---------------------------------------------------------------------------
// SSH Proxy
// ---------------------------------------------------------------------------

export interface SshProxyStatus {
  enabled: boolean;
  port: number;
  listening: boolean;
  activeSessions: number;
  allowedAuthMethods: string[];
}

export async function getSshProxyStatus(): Promise<SshProxyStatus> {
  const { data } = await api.get('/sessions/ssh-proxy/status');
  return data;
}

export interface StartSshSessionInput {
  connectionId: string;
  username?: string;
  password?: string;
  domain?: string;
  credentialMode?: 'saved' | 'domain' | 'manual';
}

export interface TerminalBrokerSshSessionResponse {
  transport: 'terminal-broker';
  sessionId: string;
  token: string;
  expiresAt: string;
  webSocketPath: string;
  webSocketUrl: string;
  dlpPolicy: ResolvedDlpPolicy;
  enforcedSshSettings: Partial<SshTerminalConfig> | null;
  sftpSupported: boolean;
  fileBrowserSupported: boolean;
}

export type StartSshSessionResponse = TerminalBrokerSshSessionResponse;

export interface ObserveSshSessionResponse {
  sessionId: string;
  token: string;
  expiresAt: string;
  webSocketPath: string;
  webSocketUrl?: string;
  mode: 'observe';
  readOnly: true;
}

export interface ObserveDesktopSessionResponse {
  sessionId: string;
  protocol: Extract<SessionProtocol, 'RDP' | 'VNC'>;
  token: string;
  expiresAt: string;
  webSocketPath: string;
  webSocketUrl?: string;
  readOnly: true;
}

export async function startSshSession(payload: StartSshSessionInput): Promise<StartSshSessionResponse> {
  const { data } = await api.post('/sessions/ssh', payload);
  return data;
}

export async function observeSshSession(sessionId: string): Promise<ObserveSshSessionResponse> {
  const { data } = await api.post(`/sessions/ssh/${sessionId}/observe`);
  return data;
}

export async function observeRdpSession(sessionId: string): Promise<ObserveDesktopSessionResponse> {
  const { data } = await api.post(`/sessions/rdp/${sessionId}/observe`);
  return data;
}

export async function observeVncSession(sessionId: string): Promise<ObserveDesktopSessionResponse> {
  const { data } = await api.post(`/sessions/vnc/${sessionId}/observe`);
  return data;
}

export async function endSshSession(sessionId: string): Promise<void> {
  await api.post(`/sessions/ssh/${sessionId}/end`, {});
}
