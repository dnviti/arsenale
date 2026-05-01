import { describe, expect, it } from 'vitest';
import { validateTimeWindows } from './accessPolicyUtils';

describe('validateTimeWindows', () => {
  it('accepts empty values and valid comma-separated ranges', () => {
    expect(validateTimeWindows('')).toBeNull();
    expect(validateTimeWindows('09:00-18:00')).toBeNull();
    expect(validateTimeWindows('09:00-12:00, 13:00-17:00')).toBeNull();
  });

  it('rejects malformed values', () => {
    expect(validateTimeWindows('9-5')).toContain('Format must be');
    expect(validateTimeWindows('25:00-18:00')).toContain('Hours must be');
    expect(validateTimeWindows('09:61-18:00')).toContain('Hours must be');
  });
});
