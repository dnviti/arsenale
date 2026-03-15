export interface VersionInfo {
  current: string;
  latest?: string;
  latestUrl?: string;
  updateAvailable: boolean;
}

const SESSION_KEY = 'arsenale-version-check';

function compareSemver(a: string, b: string): number {
  const pa = a.split('.').map(Number);
  const pb = b.split('.').map(Number);
  for (let i = 0; i < 3; i++) {
    const diff = (pa[i] || 0) - (pb[i] || 0);
    if (diff !== 0) return diff;
  }
  return 0;
}

export async function checkVersion(): Promise<VersionInfo> {
  // Get current version from server health endpoint
  const healthRes = await fetch('/api/health');
  const health = await healthRes.json();
  const current: string = health.version;

  // Check sessionStorage cache first
  const cached = sessionStorage.getItem(SESSION_KEY);
  if (cached) {
    try {
      const data = JSON.parse(cached) as VersionInfo;
      // Re-evaluate against current in case health version changed
      if (data.latest) {
        return {
          current,
          latest: data.latest,
          latestUrl: data.latestUrl,
          updateAvailable: compareSemver(data.latest, current) > 0,
        };
      }
      return { current, updateAvailable: false };
    } catch {
      // Ignore corrupt cache
    }
  }

  // Fetch latest release from GitHub
  try {
    const ghRes = await fetch(
      'https://api.github.com/repos/dnviti/arsenale/releases/latest',
    );
    if (!ghRes.ok) {
      const result: VersionInfo = { current, updateAvailable: false };
      sessionStorage.setItem(SESSION_KEY, JSON.stringify(result));
      return result;
    }
    const release = await ghRes.json();
    const latest: string = (release.tag_name || '').replace(/^v/, '');
    const result: VersionInfo = {
      current,
      latest,
      latestUrl: release.html_url,
      updateAvailable: compareSemver(latest, current) > 0,
    };
    sessionStorage.setItem(SESSION_KEY, JSON.stringify(result));
    return result;
  } catch {
    const result: VersionInfo = { current, updateAvailable: false };
    sessionStorage.setItem(SESSION_KEY, JSON.stringify(result));
    return result;
  }
}
