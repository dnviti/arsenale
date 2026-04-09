import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import LinkedAccountsSection from './LinkedAccountsSection';

const {
  getOAuthProviders,
  getLinkedAccounts,
  unlinkOAuthAccount,
  initiateOAuthLink,
  initiateSamlLink,
  notify,
} = vi.hoisted(() => ({
  getOAuthProviders: vi.fn(),
  getLinkedAccounts: vi.fn(),
  unlinkOAuthAccount: vi.fn(),
  initiateOAuthLink: vi.fn(),
  initiateSamlLink: vi.fn(),
  notify: vi.fn(),
}));

vi.mock('../../api/oauth.api', () => ({
  getOAuthProviders,
  getLinkedAccounts,
  unlinkOAuthAccount,
  initiateOAuthLink,
  initiateSamlLink,
}));

vi.mock('../../store/notificationStore', () => ({
  useNotificationStore: (selector: (state: { notify: typeof notify }) => unknown) =>
    selector({ notify }),
}));

describe('LinkedAccountsSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    getOAuthProviders.mockResolvedValue({
      google: true,
      github: true,
      saml: true,
    });
    getLinkedAccounts.mockResolvedValue([
      {
        id: 'account-1',
        provider: 'GITHUB',
        providerEmail: 'octo@example.com',
        createdAt: '2026-04-07T00:00:00Z',
      },
    ]);
    unlinkOAuthAccount.mockResolvedValue(undefined);
    initiateOAuthLink.mockResolvedValue(undefined);
    initiateSamlLink.mockResolvedValue(undefined);
  });

  it('renders linked accounts and can unlink when another auth method exists', async () => {
    render(<LinkedAccountsSection hasPassword />);

    expect(await screen.findByText('GitHub')).toBeInTheDocument();
    expect(screen.getByText('octo@example.com')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Link Google' })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Unlink' }));

    await waitFor(() => {
      expect(unlinkOAuthAccount).toHaveBeenCalledWith('GITHUB');
    });
    expect(notify).toHaveBeenCalledWith('GitHub account unlinked', 'success');
  });

  it('prevents unlinking the last remaining sign-in method', async () => {
    render(<LinkedAccountsSection hasPassword={false} />);

    const unlinkButton = await screen.findByRole('button', { name: 'Unlink' });
    expect(unlinkButton).toBeDisabled();
    expect(
      screen.getByText(/Add another sign-in method before unlinking/),
    ).toBeInTheDocument();
  });
});
