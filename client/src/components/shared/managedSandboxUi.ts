const SSH_SANDBOX_RELATIVE_ERROR_TEXT = 'Only sandbox-relative paths are allowed; remote filesystem browsing is disabled.';

export const SANDBOX_BROWSER_BANNER_TEXT = 'This browser shows only the managed transfer sandbox for this connection.';
export const REMOTE_BROWSING_DISABLED_COPY = 'Remote filesystem browsing is disabled. Use sandbox-relative paths only.';

const drivePattern = /^[a-zA-Z]:([/\\]|$)/;

export function formatManagedFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

export function formatManagedTimestamp(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  if (diffMs < 86_400_000) {
    return date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
  }
  return date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
  });
}

export function joinSandboxPath(currentPath: string, name: string): string {
  const trimmed = name.trim().replace(/^\.\//, '');
  return currentPath ? `${currentPath}/${trimmed}` : trimmed;
}

export function isDisallowedSandboxPath(value: string): boolean {
  const raw = value.trim();
  if (!raw) return false;
  if (raw === '/' || raw.startsWith('/') || raw.startsWith('\\') || raw.startsWith('~')) {
    return true;
  }
  if (drivePattern.test(raw)) {
    return true;
  }
  const lower = raw.toLowerCase();
  if (raw.includes('://') || lower.startsWith('file:')) {
    return true;
  }
  return raw.replaceAll('\\', '/').split('/').some((segment) => segment === '..');
}

export function normalizeSandboxRelativePath(value: string): string {
  return value.trim().replaceAll('\\', '/').replace(/^\.\//, '').replace(/\/+/g, '/');
}

export function mapSandboxBrowserMessage(message: string): string {
  return message.includes(SSH_SANDBOX_RELATIVE_ERROR_TEXT)
    ? REMOTE_BROWSING_DISABLED_COPY
    : message;
}

export function triggerBlobDownload(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = name;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}
