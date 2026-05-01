export function createTabInstanceId(prefix: string, connectionId: string): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${connectionId}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${connectionId}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export function buildConnectionViewerUrl(connectionId: string, tabId: string): string {
  const params = new URLSearchParams({ tabId });
  return `/connection/${connectionId}?${params.toString()}`;
}

export function resolveConnectionViewerTabId(search: string, connectionId: string): string {
  const requestedTabId = new URLSearchParams(search).get('tabId')?.trim();
  return requestedTabId || createTabInstanceId('popup', connectionId);
}
