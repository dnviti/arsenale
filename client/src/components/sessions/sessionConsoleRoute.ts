import type { SessionConsoleStatus, SessionProtocol } from '@/api/sessions.api';

export const DEFAULT_SESSION_STATUS_FILTERS: SessionConsoleStatus[] = ['ACTIVE', 'PAUSED'];
const ALLOWED_SESSION_STATUS_FILTERS = new Set<SessionConsoleStatus>(['ACTIVE', 'IDLE', 'PAUSED', 'CLOSED']);

export interface SessionsRouteState {
  q: string;
  protocol: SessionProtocol | 'all';
  status: SessionConsoleStatus[];
  gatewayId: string;
  page: number;
  recorded: boolean;
}

export const DEFAULT_SESSIONS_ROUTE_STATE: SessionsRouteState = {
  q: '',
  protocol: 'all',
  status: DEFAULT_SESSION_STATUS_FILTERS,
  gatewayId: 'all',
  page: 0,
  recorded: false,
};

export function resolveSessionsRouteState(state: Partial<SessionsRouteState> = {}): SessionsRouteState {
  return {
    ...DEFAULT_SESSIONS_ROUTE_STATE,
    ...state,
    status: normalizeSessionStatusFilters(state.status ?? DEFAULT_SESSIONS_ROUTE_STATE.status),
  };
}

export function normalizeSessionStatusFilters(statuses: readonly string[]): SessionConsoleStatus[] {
  const normalized = statuses
    .map((status) => status.trim().toUpperCase())
    .filter((status): status is SessionConsoleStatus => ALLOWED_SESSION_STATUS_FILTERS.has(status as SessionConsoleStatus));

  const unique = Array.from(new Set(normalized));
  return unique.length > 0 ? unique : [...DEFAULT_SESSION_STATUS_FILTERS];
}

export function readSessionsRouteState(searchParams: URLSearchParams): SessionsRouteState {
  const protocol = searchParams.get('protocol');
  const statuses = normalizeSessionStatusFilters((searchParams.get('status') ?? '').split(','));
  const gatewayId = searchParams.get('gatewayId');
  const pageValue = Number.parseInt(searchParams.get('page') ?? '', 10);

  return {
    q: searchParams.get('q') ?? DEFAULT_SESSIONS_ROUTE_STATE.q,
    protocol: protocol === 'SSH' || protocol === 'RDP' || protocol === 'VNC'
      ? protocol
      : DEFAULT_SESSIONS_ROUTE_STATE.protocol,
    status: statuses,
    gatewayId: gatewayId?.trim() ? gatewayId : DEFAULT_SESSIONS_ROUTE_STATE.gatewayId,
    page: Number.isFinite(pageValue) && pageValue > 0 ? pageValue : DEFAULT_SESSIONS_ROUTE_STATE.page,
    recorded: searchParams.get('recorded') === '1',
  };
}

export function buildSessionsRoute(state: Partial<SessionsRouteState> = {}): string {
  const nextState = resolveSessionsRouteState(state);
  const searchParams = new URLSearchParams();

  if (nextState.q.trim()) {
    searchParams.set('q', nextState.q.trim());
  }
  if (nextState.protocol !== 'all') {
    searchParams.set('protocol', nextState.protocol);
  }
  const normalizedStatuses = normalizeSessionStatusFilters(nextState.status);
  const defaultStatuses = DEFAULT_SESSION_STATUS_FILTERS.join(',');
  if (normalizedStatuses.join(',') !== defaultStatuses) {
    searchParams.set('status', normalizedStatuses.join(','));
  }
  if (nextState.gatewayId !== DEFAULT_SESSIONS_ROUTE_STATE.gatewayId) {
    searchParams.set('gatewayId', nextState.gatewayId);
  }
  if (nextState.page > 0) {
    searchParams.set('page', String(nextState.page));
  }
  if (nextState.recorded) {
    searchParams.set('recorded', '1');
  }

  const query = searchParams.toString();
  return query ? `/sessions?${query}` : '/sessions';
}
