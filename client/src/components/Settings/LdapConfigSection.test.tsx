import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import LdapConfigSection from './LdapConfigSection';

const {
  getLdapStatus,
  testLdapConnection,
  triggerLdapSync,
} = vi.hoisted(() => ({
  getLdapStatus: vi.fn(),
  testLdapConnection: vi.fn(),
  triggerLdapSync: vi.fn(),
}));

vi.mock('../../api/ldap.api', () => ({
  getLdapStatus,
  testLdapConnection,
  triggerLdapSync,
}));

describe('LdapConfigSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    getLdapStatus.mockResolvedValue({
      enabled: true,
      providerName: 'OpenLDAP',
      serverUrl: 'ldaps://ldap.example.com',
      baseDn: 'dc=example,dc=com',
      syncEnabled: true,
      syncCron: '0 * * * *',
      autoProvision: true,
    });
    testLdapConnection.mockResolvedValue({
      ok: true,
      message: 'Connection successful',
      userCount: 12,
      groupCount: 3,
    });
    triggerLdapSync.mockResolvedValue({
      created: 1,
      updated: 2,
      disabled: 0,
      errors: [],
    });
  });

  it('shows directory status and runs connection test and sync', async () => {
    render(<LdapConfigSection />);

    expect(await screen.findByText('OpenLDAP')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Test Connection' }));

    await waitFor(() => {
      expect(testLdapConnection).toHaveBeenCalledTimes(1);
    });
    expect(await screen.findByText('Connection successful')).toBeInTheDocument();
    expect(screen.getByText('12 users, 3 groups')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Sync Now' }));

    await waitFor(() => {
      expect(triggerLdapSync).toHaveBeenCalledTimes(1);
    });
    expect(await screen.findByText('Sync complete: 1 created, 2 updated, 0 disabled.')).toBeInTheDocument();
  });
});
