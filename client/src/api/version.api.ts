export interface VersionInfo {
  current: string;
  latest?: string;
  latestUrl?: string;
  updateAvailable: boolean;
}

const SESSION_KEY = 'arsenale-version-check';

export async function checkVersion(): Promise<VersionInfo> {
  const healthRes = await fetch('/api/health');
  const health = await healthRes.json() as { version?: unknown };
  const current = typeof health.version === 'string' && health.version.trim()
    ? health.version
    : 'dev';

  const cached = sessionStorage.getItem(SESSION_KEY);
  if (cached) {
    try {
      const data = JSON.parse(cached) as VersionInfo;
      if (data.current === current && data.updateAvailable === false) {
        return data;
      }
    } catch {
      // Ignore corrupt cache.
    }
  }

  const result: VersionInfo = { current, updateAvailable: false };
  sessionStorage.setItem(SESSION_KEY, JSON.stringify(result));
  return result;
}
