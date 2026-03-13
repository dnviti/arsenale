import type { DlpPolicy, ResolvedDlpPolicy } from '../types';

interface TenantDlpFields {
  dlpDisableCopy: boolean;
  dlpDisablePaste: boolean;
  dlpDisableDownload: boolean;
  dlpDisableUpload: boolean;
}

/**
 * Resolve the effective DLP policy by merging tenant-level floor
 * with connection-level overrides. Uses logical OR (most restrictive wins).
 */
export function resolveDlpPolicy(
  tenantDlp: TenantDlpFields,
  connectionDlp?: DlpPolicy | null,
): ResolvedDlpPolicy {
  const conn = connectionDlp ?? {};
  return {
    disableCopy: tenantDlp.dlpDisableCopy || conn.disableCopy || false,
    disablePaste: tenantDlp.dlpDisablePaste || conn.disablePaste || false,
    disableDownload: tenantDlp.dlpDisableDownload || conn.disableDownload || false,
    disableUpload: tenantDlp.dlpDisableUpload || conn.disableUpload || false,
  };
}
