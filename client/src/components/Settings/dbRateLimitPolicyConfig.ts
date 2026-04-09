import type { DbQueryType, RateLimitAction, RateLimitPolicyInput } from '../../api/dbAudit.api';
import { ALL_ROLES } from '../../utils/roles';

interface RateLimitTemplate {
  category: string;
  name: string;
  queryType: DbQueryType | null;
  windowMs: number;
  maxQueries: number;
  burstMax: number;
  action: RateLimitAction;
  description: string;
  summary?: string;
  badge?: string;
  badgeTone?: 'neutral' | 'success' | 'warning' | 'destructive';
}

export const ALL_QUERY_TYPES = 'ALL_TYPES';

export const RATE_LIMIT_QUERY_TYPE_OPTIONS: Array<{
  value: DbQueryType | typeof ALL_QUERY_TYPES;
  label: string;
}> = [
  { value: ALL_QUERY_TYPES, label: 'All query types' },
  { value: 'SELECT', label: 'SELECT' },
  { value: 'INSERT', label: 'INSERT' },
  { value: 'UPDATE', label: 'UPDATE' },
  { value: 'DELETE', label: 'DELETE' },
  { value: 'DDL', label: 'DDL' },
  { value: 'OTHER', label: 'OTHER' },
];

export const RATE_LIMIT_WINDOW_OPTIONS = [
  { value: 10000, label: '10 seconds' },
  { value: 30000, label: '30 seconds' },
  { value: 60000, label: '1 minute' },
  { value: 300000, label: '5 minutes' },
  { value: 3600000, label: '1 hour' },
];

export const RATE_LIMIT_EXEMPT_ROLES = ALL_ROLES;

export const RATE_LIMIT_ACTION_VARIANTS: Record<RateLimitAction, 'destructive' | 'secondary'> = {
  REJECT: 'destructive',
  LOG_ONLY: 'secondary',
};

export const EMPTY_RATE_LIMIT_POLICY_FORM: RateLimitPolicyInput = {
  name: '',
  queryType: null,
  windowMs: 60000,
  maxQueries: 100,
  burstMax: 10,
  exemptRoles: [],
  scope: '',
  action: 'REJECT',
  enabled: true,
  priority: 0,
};

export const RATE_LIMIT_POLICY_TEMPLATES: RateLimitTemplate[] = [
  {
    category: 'General Protection',
    name: 'Standard Query Limit',
    queryType: null,
    windowMs: 60000,
    maxQueries: 100,
    burstMax: 10,
    action: 'REJECT',
    description: 'Balanced default limit for mixed workloads.',
    summary: 'All queries · 100/min · burst 10',
    badge: 'REJECT',
    badgeTone: 'destructive',
  },
  {
    category: 'General Protection',
    name: 'Strict Query Limit',
    queryType: null,
    windowMs: 60000,
    maxQueries: 30,
    burstMax: 5,
    action: 'REJECT',
    description: 'Tighter ceiling for shared or high-risk tenants.',
    summary: 'All queries · 30/min · burst 5',
    badge: 'REJECT',
    badgeTone: 'destructive',
  },
  {
    category: 'General Protection',
    name: 'Relaxed Query Limit',
    queryType: null,
    windowMs: 60000,
    maxQueries: 500,
    burstMax: 50,
    action: 'LOG_ONLY',
    description: 'Observe rate pressure without blocking requests.',
    summary: 'All queries · 500/min · burst 50',
    badge: 'LOG',
    badgeTone: 'warning',
  },
  {
    category: 'Write Protection',
    name: 'INSERT Rate Limit',
    queryType: 'INSERT',
    windowMs: 60000,
    maxQueries: 50,
    burstMax: 10,
    action: 'REJECT',
    description: 'Cap ingestion spikes on INSERT-heavy workloads.',
    summary: 'INSERT · 50/min · burst 10',
    badge: 'REJECT',
    badgeTone: 'destructive',
  },
  {
    category: 'Write Protection',
    name: 'UPDATE Rate Limit',
    queryType: 'UPDATE',
    windowMs: 60000,
    maxQueries: 30,
    burstMax: 5,
    action: 'REJECT',
    description: 'Slow down large mutation bursts before they fan out.',
    summary: 'UPDATE · 30/min · burst 5',
    badge: 'REJECT',
    badgeTone: 'destructive',
  },
  {
    category: 'Write Protection',
    name: 'DELETE Rate Limit',
    queryType: 'DELETE',
    windowMs: 60000,
    maxQueries: 20,
    burstMax: 3,
    action: 'REJECT',
    description: 'Protect against mass deletion events.',
    summary: 'DELETE · 20/min · burst 3',
    badge: 'REJECT',
    badgeTone: 'destructive',
  },
  {
    category: 'DDL Protection',
    name: 'DDL Rate Limit',
    queryType: 'DDL',
    windowMs: 300000,
    maxQueries: 5,
    burstMax: 2,
    action: 'REJECT',
    description: 'Throttle schema change operations aggressively.',
    summary: 'DDL · 5/5 min · burst 2',
    badge: 'REJECT',
    badgeTone: 'destructive',
  },
  {
    category: 'Performance',
    name: 'SELECT Throttle',
    queryType: 'SELECT',
    windowMs: 10000,
    maxQueries: 50,
    burstMax: 20,
    action: 'REJECT',
    description: 'Reduce read storms before they consume the database.',
    summary: 'SELECT · 50/10 sec · burst 20',
    badge: 'REJECT',
    badgeTone: 'destructive',
  },
  {
    category: 'Performance',
    name: 'Heavy Read Alert',
    queryType: 'SELECT',
    windowMs: 60000,
    maxQueries: 200,
    burstMax: 30,
    action: 'LOG_ONLY',
    description: 'Track oversized read patterns without rejecting them.',
    summary: 'SELECT · 200/min · burst 30',
    badge: 'LOG',
    badgeTone: 'warning',
  },
];

export function formatRateLimitWindow(ms: number): string {
  if (ms < 60000) return `${ms / 1000}s`;
  if (ms < 3600000) return `${ms / 60000}m`;
  return `${ms / 3600000}h`;
}
