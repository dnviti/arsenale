import type { TeamMember } from '../../api/team.api';

export const TEAM_ROLES = ['TEAM_ADMIN', 'TEAM_EDITOR', 'TEAM_VIEWER'] as const;

export type TeamRole = (typeof TEAM_ROLES)[number];

const dateFormatter = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' });
const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short',
});

export function roleLabel(role: string) {
  return role.replace(/^TEAM_/, '').replace(/_/g, ' ');
}

export function getTeamMemberName(member: Pick<TeamMember, 'username' | 'email'>) {
  return member.username || member.email;
}

export function getInitials(label: string) {
  return label.charAt(0).toUpperCase();
}

export function formatTeamDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return dateFormatter.format(date);
}

export function formatTeamDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return dateTimeFormatter.format(date);
}

export function formatMemberExpiry(expiresAt: string | null, expired: boolean) {
  if (!expiresAt) {
    return 'No expiration';
  }

  if (expired) {
    return 'Expired';
  }

  return formatTeamDateTime(expiresAt);
}

export function toDateTimeLocalValue(value: string | null) {
  if (!value) {
    return '';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }

  const local = new Date(date.getTime() - (date.getTimezoneOffset() * 60000));
  return local.toISOString().slice(0, 16);
}

export function fromDateTimeLocalValue(value: string) {
  if (!value) {
    return null;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return date.toISOString();
}
