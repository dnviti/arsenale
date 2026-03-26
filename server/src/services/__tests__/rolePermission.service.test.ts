import {
  resolvePermissions,
  ROLE_DEFAULTS,
  ALL_PERMISSION_FLAGS,
} from '../rolePermission.service';

// ---------------------------------------------------------------------------
// ALL_PERMISSION_FLAGS
// ---------------------------------------------------------------------------
describe('ALL_PERMISSION_FLAGS', () => {
  it('contains exactly 11 entries', () => {
    expect(ALL_PERMISSION_FLAGS).toHaveLength(11);
  });

  it('all entries are strings', () => {
    for (const flag of ALL_PERMISSION_FLAGS) {
      expect(typeof flag).toBe('string');
    }
  });
});

// ---------------------------------------------------------------------------
// ROLE_DEFAULTS
// ---------------------------------------------------------------------------
describe('ROLE_DEFAULTS', () => {
  const allRoles = ['OWNER', 'ADMIN', 'OPERATOR', 'MEMBER', 'CONSULTANT', 'AUDITOR', 'GUEST'] as const;

  it('OWNER has all permissions true', () => {
    for (const flag of ALL_PERMISSION_FLAGS) {
      expect(ROLE_DEFAULTS.OWNER[flag]).toBe(true);
    }
  });

  it('GUEST has all permissions false', () => {
    for (const flag of ALL_PERMISSION_FLAGS) {
      expect(ROLE_DEFAULTS.GUEST[flag]).toBe(false);
    }
  });

  it('AUDITOR has canViewAuditLog and canManageSessions true, rest false', () => {
    for (const flag of ALL_PERMISSION_FLAGS) {
      if (flag === 'canViewAuditLog' || flag === 'canManageSessions') {
        expect(ROLE_DEFAULTS.AUDITOR[flag]).toBe(true);
      } else {
        expect(ROLE_DEFAULTS.AUDITOR[flag]).toBe(false);
      }
    }
  });

  it.each(allRoles)('%s has exactly 11 flags', (role) => {
    const keys = Object.keys(ROLE_DEFAULTS[role]);
    expect(keys).toHaveLength(11);
  });
});

// ---------------------------------------------------------------------------
// resolvePermissions
// ---------------------------------------------------------------------------
describe('resolvePermissions', () => {
  it('returns full defaults for a role when no overrides provided', () => {
    const result = resolvePermissions('GUEST');
    expect(result).toEqual(ROLE_DEFAULTS.GUEST);
  });

  it('returns full defaults when overrides is null', () => {
    const result = resolvePermissions('ADMIN', null);
    expect(result).toEqual(ROLE_DEFAULTS.ADMIN);
  });

  it('returns full defaults when overrides is undefined', () => {
    const result = resolvePermissions('MEMBER', undefined);
    expect(result).toEqual(ROLE_DEFAULTS.MEMBER);
  });

  it('correctly merges an override that flips a flag from default', () => {
    // GUEST defaults to all false; override canConnect to true
    const result = resolvePermissions('GUEST', { canConnect: true });
    expect(result.canConnect).toBe(true);
    // All other flags remain false
    for (const flag of ALL_PERMISSION_FLAGS) {
      if (flag !== 'canConnect') {
        expect(result[flag]).toBe(false);
      }
    }
  });

  it('can flip a true default to false via override', () => {
    // OWNER defaults to all true; override canManageUsers to false
    const result = resolvePermissions('OWNER', { canManageUsers: false });
    expect(result.canManageUsers).toBe(false);
    // All other flags remain true
    for (const flag of ALL_PERMISSION_FLAGS) {
      if (flag !== 'canManageUsers') {
        expect(result[flag]).toBe(true);
      }
    }
  });

  it('applies multiple overrides at once', () => {
    const result = resolvePermissions('GUEST', {
      canConnect: true,
      canViewAuditLog: true,
    });
    expect(result.canConnect).toBe(true);
    expect(result.canViewAuditLog).toBe(true);
    expect(result.canCreateConnections).toBe(false);
  });

  it('ignores unknown/invalid flag keys in overrides', () => {
    const result = resolvePermissions('GUEST', {
      notARealFlag: true,
      alsoFake: false,
    } as Record<string, boolean>);
    // Result should be identical to unmodified GUEST defaults
    expect(result).toEqual(ROLE_DEFAULTS.GUEST);
  });

  it('ignores non-boolean values in overrides', () => {
    const result = resolvePermissions('GUEST', {
      canConnect: 'yes' as unknown as boolean,
      canViewAuditLog: 1 as unknown as boolean,
      canManageSessions: null as unknown as boolean,
    });
    // None of the non-boolean values should be applied
    expect(result).toEqual(ROLE_DEFAULTS.GUEST);
  });

  it('does not mutate ROLE_DEFAULTS when overrides are applied', () => {
    const originalGuest = { ...ROLE_DEFAULTS.GUEST };
    resolvePermissions('GUEST', { canConnect: true });
    expect(ROLE_DEFAULTS.GUEST).toEqual(originalGuest);
  });

  it('returns a new object each call (no shared reference)', () => {
    const a = resolvePermissions('OWNER');
    const b = resolvePermissions('OWNER');
    expect(a).toEqual(b);
    expect(a).not.toBe(b);
  });
});
