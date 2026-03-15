import { describe, it, expect } from 'vitest';
import {
  hasMinRole,
  hasAnyRole,
  isAdminOrAbove,
  isOperatorOrAbove,
  ALL_ROLES,
  ASSIGNABLE_ROLES,
  type TenantRole,
} from './roles';

describe('hasMinRole', () => {
  it('OWNER meets any role', () => {
    for (const role of ALL_ROLES) {
      expect(hasMinRole('OWNER', role)).toBe(true);
    }
  });

  it('ADMIN meets ADMIN', () => {
    expect(hasMinRole('ADMIN', 'ADMIN')).toBe(true);
  });

  it('ADMIN does NOT meet OWNER', () => {
    expect(hasMinRole('ADMIN', 'OWNER')).toBe(false);
  });

  it('MEMBER meets GUEST', () => {
    expect(hasMinRole('MEMBER', 'GUEST')).toBe(true);
  });

  it('GUEST does not meet MEMBER', () => {
    expect(hasMinRole('GUEST', 'MEMBER')).toBe(false);
  });

  it('returns false for undefined role', () => {
    expect(hasMinRole(undefined, 'GUEST')).toBe(false);
  });

  it('returns false for unknown string role (defaults to 0)', () => {
    expect(hasMinRole('UNKNOWN', 'GUEST')).toBe(false);
  });

  it('each role meets itself', () => {
    for (const role of ALL_ROLES) {
      expect(hasMinRole(role, role)).toBe(true);
    }
  });

  it('respects hierarchy order: OWNER > ADMIN > OPERATOR > MEMBER > CONSULTANT > AUDITOR > GUEST', () => {
    const ordered: TenantRole[] = ['OWNER', 'ADMIN', 'OPERATOR', 'MEMBER', 'CONSULTANT', 'AUDITOR', 'GUEST'];
    for (let i = 0; i < ordered.length; i++) {
      for (let j = i; j < ordered.length; j++) {
        expect(hasMinRole(ordered[i], ordered[j])).toBe(true);
      }
      for (let j = 0; j < i; j++) {
        expect(hasMinRole(ordered[i], ordered[j])).toBe(false);
      }
    }
  });
});

describe('hasAnyRole', () => {
  it('ADMIN in [ADMIN, OWNER] returns true', () => {
    expect(hasAnyRole('ADMIN', 'ADMIN', 'OWNER')).toBe(true);
  });

  it('MEMBER in [ADMIN, OWNER] returns false', () => {
    expect(hasAnyRole('MEMBER', 'ADMIN', 'OWNER')).toBe(false);
  });

  it('returns false for undefined', () => {
    expect(hasAnyRole(undefined, 'ADMIN', 'OWNER')).toBe(false);
  });

  it('GUEST in [GUEST] returns true', () => {
    expect(hasAnyRole('GUEST', 'GUEST')).toBe(true);
  });
});

describe('isAdminOrAbove', () => {
  it('OWNER returns true', () => {
    expect(isAdminOrAbove('OWNER')).toBe(true);
  });

  it('ADMIN returns true', () => {
    expect(isAdminOrAbove('ADMIN')).toBe(true);
  });

  it('OPERATOR returns false', () => {
    expect(isAdminOrAbove('OPERATOR')).toBe(false);
  });

  it('MEMBER returns false', () => {
    expect(isAdminOrAbove('MEMBER')).toBe(false);
  });

  it('undefined returns false', () => {
    expect(isAdminOrAbove(undefined)).toBe(false);
  });
});

describe('isOperatorOrAbove', () => {
  it('OWNER returns true', () => {
    expect(isOperatorOrAbove('OWNER')).toBe(true);
  });

  it('ADMIN returns true', () => {
    expect(isOperatorOrAbove('ADMIN')).toBe(true);
  });

  it('OPERATOR returns true', () => {
    expect(isOperatorOrAbove('OPERATOR')).toBe(true);
  });

  it('MEMBER returns false', () => {
    expect(isOperatorOrAbove('MEMBER')).toBe(false);
  });

  it('undefined returns false', () => {
    expect(isOperatorOrAbove(undefined)).toBe(false);
  });
});

describe('constants', () => {
  it('ALL_ROLES has 7 entries', () => {
    expect(ALL_ROLES).toHaveLength(7);
  });

  it('ASSIGNABLE_ROLES excludes OWNER', () => {
    expect(ASSIGNABLE_ROLES).not.toContain('OWNER');
    expect(ASSIGNABLE_ROLES).toEqual(['ADMIN', 'OPERATOR', 'MEMBER', 'CONSULTANT', 'AUDITOR', 'GUEST']);
  });
});
