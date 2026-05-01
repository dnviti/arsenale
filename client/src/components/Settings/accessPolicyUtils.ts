import type { AccessPolicyData, AccessPolicyTargetType } from '../../api/accessPolicy.api';
import type { TeamData } from '../../api/team.api';
import type { FolderData } from '../../api/folders.api';

// eslint-disable-next-line security/detect-unsafe-regex
const TIME_WINDOW_RE = /^(\d{2}:\d{2}-\d{2}:\d{2})(,\s*\d{2}:\d{2}-\d{2}:\d{2})*$/;

export const ACCESS_POLICY_TARGET_LABELS: Record<AccessPolicyTargetType, string> = {
  TENANT: 'Organization',
  TEAM: 'Team',
  FOLDER: 'Folder',
};

export interface AccessPolicyFormState {
  targetType: AccessPolicyTargetType;
  targetId: string;
  allowedTimeWindows: string;
  requireTrustedDevice: boolean;
  requireMfaStepUp: boolean;
}

export const EMPTY_ACCESS_POLICY_FORM: AccessPolicyFormState = {
  targetType: 'TENANT',
  targetId: '',
  allowedTimeWindows: '',
  requireTrustedDevice: false,
  requireMfaStepUp: false,
};

export function validateTimeWindows(value: string): string | null {
  if (!value.trim()) return null;
  if (!TIME_WINDOW_RE.test(value.trim())) {
    return 'Format must be HH:MM-HH:MM (comma-separated for multiple)';
  }

  const windows = value.split(',').map((windowValue) => windowValue.trim());
  for (const windowValue of windows) {
    const [startValue, endValue] = windowValue.split('-');
    if (!startValue || !endValue) return 'Invalid time window format';

    const [startHour, startMinute] = startValue.split(':').map(Number);
    const [endHour, endMinute] = endValue.split(':').map(Number);

    if (
      startHour < 0
      || startHour > 23
      || startMinute < 0
      || startMinute > 59
      || endHour < 0
      || endHour > 23
      || endMinute < 0
      || endMinute > 59
    ) {
      return 'Hours must be 0-23, minutes 0-59';
    }
  }

  return null;
}

export function buildAccessPolicyNameMap({
  tenantId,
  teams,
  folders,
}: {
  tenantId?: string;
  teams: TeamData[];
  folders: FolderData[];
}) {
  const map: Record<string, string> = {};

  if (tenantId) {
    map[tenantId] = 'Current organization';
  }

  for (const team of teams) {
    map[team.id] = team.name;
  }

  for (const folder of folders) {
    map[folder.id] = folder.name;
  }

  return map;
}

export function buildTargetOptions({
  targetType,
  tenantId,
  teams,
  folders,
}: {
  targetType: AccessPolicyTargetType;
  tenantId?: string;
  teams: TeamData[];
  folders: FolderData[];
}) {
  switch (targetType) {
    case 'TENANT':
      return tenantId ? [{ id: tenantId, name: 'Current organization' }] : [];
    case 'TEAM':
      return teams.map((team) => ({ id: team.id, name: team.name }));
    case 'FOLDER':
      return folders.map((folder) => ({ id: folder.id, name: folder.name }));
    default:
      return [];
  }
}

export function timeWindowBadges(policy: AccessPolicyData) {
  return policy.allowedTimeWindows
    ? policy.allowedTimeWindows.split(',').map((windowValue) => windowValue.trim())
    : ['Any time'];
}
