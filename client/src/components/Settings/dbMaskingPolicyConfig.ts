import type { MaskingPolicyInput, MaskingStrategy } from '../../api/dbAudit.api';
import { ALL_ROLES } from '../../utils/roles';

interface MaskingTemplate {
  category: string;
  name: string;
  columnPattern: string;
  strategy: MaskingStrategy;
  description: string;
  summary?: string;
  badge?: string;
  badgeTone?: 'neutral' | 'success' | 'warning' | 'destructive';
}

export const MASKING_EXEMPT_ROLES = ALL_ROLES;

export const MASKING_STRATEGY_LABELS: Record<MaskingStrategy, string> = {
  REDACT: 'Full redaction',
  HASH: 'SHA-256 hash',
  PARTIAL: 'Partial mask',
};

export const MASKING_STRATEGY_VARIANTS: Record<MaskingStrategy, 'destructive' | 'secondary' | 'outline'> = {
  REDACT: 'destructive',
  HASH: 'secondary',
  PARTIAL: 'outline',
};

export const MASKING_STRATEGY_OPTIONS: Array<{
  value: MaskingStrategy;
  label: string;
  description: string;
}> = [
  {
    value: 'REDACT',
    label: 'Redact',
    description: 'Replace the full value with a redacted placeholder.',
  },
  {
    value: 'HASH',
    label: 'Hash',
    description: 'Return a truncated SHA-256 hash for stable comparisons.',
  },
  {
    value: 'PARTIAL',
    label: 'Partial mask',
    description: 'Expose only a small prefix while masking the rest of the value.',
  },
];

export const EMPTY_MASKING_POLICY_FORM: MaskingPolicyInput = {
  name: '',
  columnPattern: '',
  strategy: 'REDACT',
  exemptRoles: [],
  scope: '',
  description: '',
  enabled: true,
};

export const MASKING_POLICY_TEMPLATES: MaskingTemplate[] = [
  {
    category: 'PII / Identity',
    name: 'Mask SSN / National ID',
    columnPattern: '(ssn|social_security|national_id|tax_id|identity_number)',
    strategy: 'PARTIAL',
    description: 'Partially masks social security numbers and national identifiers.',
    summary: 'Good default when operators need the last digits for support work.',
    badge: 'Partial',
    badgeTone: 'warning',
  },
  {
    category: 'PII / Identity',
    name: 'Redact Full Names',
    columnPattern: '(full_name|first_name|last_name|surname|given_name)',
    strategy: 'REDACT',
    description: 'Fully redacts personal name columns.',
    badge: 'Redact',
    badgeTone: 'destructive',
  },
  {
    category: 'PII / Identity',
    name: 'Hash Personal Identifiers',
    columnPattern: '(passport|driver_license|license_number)',
    strategy: 'HASH',
    description: 'Hashes government-issued ID numbers for pseudonymized analytics.',
    badge: 'Hash',
  },
  {
    category: 'Financial',
    name: 'Mask Credit Cards',
    columnPattern: '(credit_card|card_number|pan|cc_number)',
    strategy: 'PARTIAL',
    description: 'Shows only the last digits of payment card numbers.',
    badge: 'Partial',
    badgeTone: 'warning',
  },
  {
    category: 'Financial',
    name: 'Redact Bank Accounts',
    columnPattern: '(bank_account|iban|routing_number|sort_code|account_number)',
    strategy: 'REDACT',
    description: 'Fully redacts banking and financial account numbers.',
    badge: 'Redact',
    badgeTone: 'destructive',
  },
  {
    category: 'Financial',
    name: 'Redact Salary / Compensation',
    columnPattern: '(salary|wage|compensation|income|bonus)',
    strategy: 'REDACT',
    description: 'Hides salary and compensation data from non-privileged users.',
    badge: 'Redact',
    badgeTone: 'destructive',
  },
  {
    category: 'Authentication',
    name: 'Redact Passwords',
    columnPattern: '(password|passwd|pwd|secret|pin)',
    strategy: 'REDACT',
    description: 'Fully redacts password and secret columns.',
    badge: 'Redact',
    badgeTone: 'destructive',
  },
  {
    category: 'Authentication',
    name: 'Hash API Keys / Tokens',
    columnPattern: '(api_key|access_key|secret_key|auth_token|refresh_token)',
    strategy: 'HASH',
    description: 'Hashes API keys and tokens for reference without exposing raw values.',
    badge: 'Hash',
  },
  {
    category: 'Contact Information',
    name: 'Mask Email Addresses',
    columnPattern: '(email|e_mail|email_address)',
    strategy: 'PARTIAL',
    description: 'Partially masks email addresses while retaining the domain.',
    badge: 'Partial',
    badgeTone: 'warning',
  },
  {
    category: 'Contact Information',
    name: 'Mask Phone Numbers',
    columnPattern: '(phone|telephone|mobile|cell|fax)',
    strategy: 'PARTIAL',
    description: 'Partially masks phone numbers while keeping the trailing digits.',
    badge: 'Partial',
    badgeTone: 'warning',
  },
  {
    category: 'Contact Information',
    name: 'Redact Physical Addresses',
    columnPattern: '(address|street|city|zip_code|postal_code)',
    strategy: 'REDACT',
    description: 'Fully redacts physical address components.',
    badge: 'Redact',
    badgeTone: 'destructive',
  },
  {
    category: 'Healthcare',
    name: 'Redact Medical Records',
    columnPattern: '(diagnosis|medical_record|patient_id|health_id)',
    strategy: 'REDACT',
    description: 'Fully redacts protected health information.',
    badge: 'Redact',
    badgeTone: 'destructive',
  },
  {
    category: 'Healthcare',
    name: 'Hash Prescription Data',
    columnPattern: '(prescription|medication|drug_name)',
    strategy: 'HASH',
    description: 'Hashes prescription data for pseudonymized research workflows.',
    badge: 'Hash',
  },
];

export function validateMaskingColumnPattern(pattern: string): string | null {
  if (!pattern.trim()) {
    return 'A column pattern is required.';
  }

  try {
    // Compile locally so the dialog catches invalid regular expressions before save.
    // eslint-disable-next-line security/detect-non-literal-regexp -- this path only validates user input locally.
    new RegExp(pattern);
    return null;
  } catch (error) {
    if (error instanceof Error && error.message.trim()) {
      return error.message;
    }
    return 'Invalid regular expression.';
  }
}
