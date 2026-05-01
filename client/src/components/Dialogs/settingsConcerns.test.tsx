import type { ReactElement } from 'react';
import { describe, expect, it, vi } from 'vitest';
import {
  buildSettingsConcerns,
  type SettingsConcernContext,
} from './settingsConcerns';

function createContext(
  overrides: Partial<SettingsConcernContext> = {},
): SettingsConcernContext {
  return {
    hasPassword: true,
    hasTenant: true,
    isAdmin: true,
    isOwner: true,
    anyConnectionFeature: true,
    connectionsEnabled: true,
    databaseProxyEnabled: true,
    keychainEnabled: true,
    zeroTrustEnabled: true,
    agenticAIEnabled: true,
    enterpriseAuthEnabled: true,
    linkedProvider: null,
    tenantId: 'tenant-1',
    onHasPasswordResolved: vi.fn(),
    onViewUserProfile: vi.fn(),
    onImport: vi.fn(),
    onExport: vi.fn(),
    deleteOrgTrigger: vi.fn(),
    setDeleteOrgTrigger: vi.fn(),
    navigateToConcern: vi.fn(),
    ...overrides,
  };
}

describe('buildSettingsConcerns', () => {
  it('groups settings into concern-based clusters and preserves profile callback wiring', () => {
    const onHasPasswordResolved = vi.fn();
    const concerns = buildSettingsConcerns(
      createContext({ onHasPasswordResolved }),
    );

    expect(concerns.map((concern) => concern.id)).toEqual([
      'personal',
      'security',
      'organization',
      'infrastructure',
      'integrations',
      'governance',
    ]);

    const personal = concerns.find((concern) => concern.id === 'personal');
    expect(personal?.sections.map((section) => section.id)).toEqual([
      'profile',
      'password',
      'appearance',
      'notifications',
      'data-movement',
      'connection-defaults',
    ]);

    const profileSection = personal?.sections.find(
      (section) => section.id === 'profile',
    );
    const profileElement = profileSection?.content as ReactElement<{
      onHasPasswordResolved: (hasPassword: boolean) => void;
    }>;
    expect(profileElement.props.onHasPasswordResolved).toBe(
      onHasPasswordResolved,
    );
  });

  it('drops gated concerns and sections when the related capabilities are disabled', () => {
    const concerns = buildSettingsConcerns(
      createContext({
        hasTenant: false,
        isAdmin: false,
        isOwner: false,
        anyConnectionFeature: false,
        connectionsEnabled: false,
        databaseProxyEnabled: false,
        keychainEnabled: false,
        zeroTrustEnabled: false,
        agenticAIEnabled: false,
        enterpriseAuthEnabled: false,
        onImport: undefined,
        onExport: undefined,
        deleteOrgTrigger: null,
      }),
    );

    expect(concerns.map((concern) => concern.id)).toEqual([
      'personal',
      'security',
      'integrations',
    ]);

    const personal = concerns.find((concern) => concern.id === 'personal');
    expect(personal?.sections.map((section) => section.id)).toEqual([
      'profile',
      'password',
      'appearance',
      'notifications',
    ]);

    const security = concerns.find((concern) => concern.id === 'security');
    expect(security?.sections.map((section) => section.id)).toEqual([
      'two-factor',
      'sms-mfa',
      'webauthn',
      'trusted-domains',
    ]);

    const integrations = concerns.find((concern) => concern.id === 'integrations');
    expect(integrations?.sections.map((section) => section.id)).toEqual([
      'sync',
    ]);
  });

  it('keeps external vault providers under integrations and out of organization concerns', () => {
    const concerns = buildSettingsConcerns(createContext());

    const organization = concerns.find((concern) => concern.id === 'organization');
    const integrations = concerns.find((concern) => concern.id === 'integrations');

    expect(organization?.sections.some((section) => section.id === 'vault-providers')).toBe(false);
    expect(integrations?.sections.some((section) => section.id === 'vault-providers')).toBe(true);
    expect(integrations?.sections.map((section) => section.id)).toContain('vault-providers');
  });

  it('keeps SQL governance controls in general settings', () => {
    const concerns = buildSettingsConcerns(createContext());

    const governance = concerns.find((concern) => concern.id === 'governance');
    expect(governance?.sections.map((section) => section.id)).toEqual([
      'system',
      'ip-allowlist',
      'access-policy',
      'db-firewall',
      'db-masking',
      'db-rate-limit',
    ]);
  });

  it('drops tenant-scoped integrations when there is no active tenant', () => {
    const concerns = buildSettingsConcerns(
      createContext({
        hasTenant: false,
        tenantId: null,
      }),
    );

    const integrations = concerns.find((concern) => concern.id === 'integrations');
    expect(integrations?.sections.map((section) => section.id)).not.toContain('vault-providers');
  });
});
