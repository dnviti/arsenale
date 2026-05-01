import type { FirewallAction, FirewallRuleInput } from '../../api/dbAudit.api';

interface FirewallTemplate {
  category: string;
  name: string;
  pattern: string;
  action: FirewallAction;
  description: string;
  summary?: string;
  badge?: string;
  badgeTone?: 'neutral' | 'success' | 'warning' | 'destructive';
}

export const FIREWALL_ACTION_VARIANTS: Record<FirewallAction, 'destructive' | 'secondary' | 'outline'> = {
  BLOCK: 'destructive',
  ALERT: 'secondary',
  LOG: 'outline',
};

export const EMPTY_FIREWALL_RULE_FORM: FirewallRuleInput = {
  name: '',
  pattern: '',
  action: 'BLOCK',
  scope: '',
  description: '',
  enabled: true,
  priority: 0,
};

export const FIREWALL_RULE_TEMPLATES: FirewallTemplate[] = [
  {
    category: 'Destructive Operations',
    name: 'Block DROP TABLE',
    pattern: '\\bDROP\\s+TABLE\\b',
    action: 'BLOCK',
    description: 'Prevent accidental or hostile table deletion.',
    badge: 'BLOCK',
    badgeTone: 'destructive',
  },
  {
    category: 'Destructive Operations',
    name: 'Block TRUNCATE',
    pattern: '\\bTRUNCATE\\b',
    action: 'BLOCK',
    description: 'Stop full-table wipes that bypass row-by-row safeguards.',
    badge: 'BLOCK',
    badgeTone: 'destructive',
  },
  {
    category: 'Destructive Operations',
    name: 'Block DROP DATABASE',
    pattern: '\\bDROP\\s+DATABASE\\b',
    action: 'BLOCK',
    description: 'Protect entire databases from destructive removal.',
    badge: 'BLOCK',
    badgeTone: 'destructive',
  },
  {
    category: 'Destructive Operations',
    name: 'Block DROP SCHEMA',
    pattern: '\\bDROP\\s+SCHEMA\\b',
    action: 'BLOCK',
    description: 'Prevent schema deletion across shared environments.',
    badge: 'BLOCK',
    badgeTone: 'destructive',
  },
  {
    category: 'Data Modification',
    name: 'Alert DELETE without WHERE',
    pattern: '^\\s*DELETE\\s+FROM\\s+\\S+\\s*;?\\s*$',
    action: 'ALERT',
    description: 'Flag deletes that would remove every row.',
    badge: 'ALERT',
    badgeTone: 'warning',
  },
  {
    category: 'Data Modification',
    name: 'Alert UPDATE without WHERE',
    pattern: '^\\s*UPDATE\\s+\\S+\\s+SET\\s+.*(?<!WHERE\\s+.*)\\s*;?\\s*$',
    action: 'ALERT',
    description: 'Catch mass updates before they land.',
    badge: 'ALERT',
    badgeTone: 'warning',
  },
  {
    category: 'Data Modification',
    name: 'Log all INSERT statements',
    pattern: '\\bINSERT\\s+INTO\\b',
    action: 'LOG',
    description: 'Record write-heavy ingestion activity without blocking it.',
    badge: 'LOG',
  },
  {
    category: 'Schema Changes',
    name: 'Alert ALTER TABLE',
    pattern: '\\bALTER\\s+TABLE\\b',
    action: 'ALERT',
    description: 'Surface table shape changes for review.',
    badge: 'ALERT',
    badgeTone: 'warning',
  },
  {
    category: 'Schema Changes',
    name: 'Block CREATE/DROP INDEX',
    pattern: '\\b(CREATE|DROP)\\s+(UNIQUE\\s+)?INDEX\\b',
    action: 'BLOCK',
    description: 'Prevent index churn that can destabilize performance.',
    badge: 'BLOCK',
    badgeTone: 'destructive',
  },
  {
    category: 'Schema Changes',
    name: 'Alert GRANT/REVOKE',
    pattern: '\\b(GRANT|REVOKE)\\b',
    action: 'ALERT',
    description: 'Highlight changes to database privileges.',
    badge: 'ALERT',
    badgeTone: 'warning',
  },
  {
    category: 'Security',
    name: 'Block SQL comment injection',
    pattern: '(--|/\\*|\\*/)',
    action: 'BLOCK',
    description: 'Reject classic inline-comment injection patterns.',
    badge: 'BLOCK',
    badgeTone: 'destructive',
  },
  {
    category: 'Security',
    name: 'Block UNION-based injection',
    pattern: '\\bUNION\\s+(ALL\\s+)?SELECT\\b',
    action: 'BLOCK',
    description: 'Block common UNION SELECT exfiltration patterns.',
    badge: 'BLOCK',
    badgeTone: 'destructive',
  },
  {
    category: 'Security',
    name: 'Block system table access',
    pattern: '\\b(pg_catalog|information_schema|sys\\.objects|sysobjects|mysql\\.user)\\b',
    action: 'BLOCK',
    description: 'Restrict direct reads from system catalogs.',
    badge: 'BLOCK',
    badgeTone: 'destructive',
  },
  {
    category: 'Performance',
    name: 'Alert SELECT *',
    pattern: '^\\s*SELECT\\s+\\*\\s+FROM\\s+\\S+\\s*;?\\s*$',
    action: 'ALERT',
    description: 'Flag unbounded reads that tend to grow over time.',
    badge: 'ALERT',
    badgeTone: 'warning',
  },
  {
    category: 'Performance',
    name: 'Alert CROSS JOIN',
    pattern: '\\bCROSS\\s+JOIN\\b',
    action: 'ALERT',
    description: 'Spot cartesian products before they impact latency.',
    badge: 'ALERT',
    badgeTone: 'warning',
  },
];
