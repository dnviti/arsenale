import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import WebAuthnSection from './WebAuthnSection';

const {
  browserSupportsWebAuthn,
  startRegistration,
  getWebAuthnStatus,
  getWebAuthnCredentials,
  getWebAuthnRegistrationOptions,
  registerWebAuthnCredential,
  removeWebAuthnCredential,
  renameWebAuthnCredential,
  notify,
} = vi.hoisted(() => ({
  browserSupportsWebAuthn: vi.fn(),
  startRegistration: vi.fn(),
  getWebAuthnStatus: vi.fn(),
  getWebAuthnCredentials: vi.fn(),
  getWebAuthnRegistrationOptions: vi.fn(),
  registerWebAuthnCredential: vi.fn(),
  removeWebAuthnCredential: vi.fn(),
  renameWebAuthnCredential: vi.fn(),
  notify: vi.fn(),
}));

vi.mock('@simplewebauthn/browser', () => ({
  browserSupportsWebAuthn,
  startRegistration,
}));

vi.mock('../../api/webauthn.api', () => ({
  getWebAuthnStatus,
  getWebAuthnCredentials,
  getWebAuthnRegistrationOptions,
  registerWebAuthnCredential,
  removeWebAuthnCredential,
  renameWebAuthnCredential,
}));

vi.mock('../../store/notificationStore', () => ({
  useNotificationStore: (selector: (state: { notify: typeof notify }) => unknown) =>
    selector({ notify }),
}));

describe('WebAuthnSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    browserSupportsWebAuthn.mockReturnValue(true);
    getWebAuthnStatus.mockResolvedValue({ enabled: true, credentialCount: 1 });
    getWebAuthnCredentials.mockResolvedValue([
      {
        id: 'cred-1',
        credentialId: 'credential-id',
        friendlyName: 'Office Key',
        deviceType: 'singleDevice',
        backedUp: true,
        lastUsedAt: '2026-04-06T00:00:00Z',
        createdAt: '2026-04-01T00:00:00Z',
      },
    ]);
    getWebAuthnRegistrationOptions.mockResolvedValue({ challenge: 'challenge-123' });
    startRegistration.mockResolvedValue({ id: 'credential-json' });
    registerWebAuthnCredential.mockResolvedValue({});
    removeWebAuthnCredential.mockResolvedValue({ removed: true });
    renameWebAuthnCredential.mockResolvedValue({ renamed: true });
  });

  it('registers a new security key after naming it', async () => {
    render(<WebAuthnSection />);

    expect(await screen.findByText('Office Key')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Add Security Key' }));

    const dialog = await screen.findByRole('dialog');
    fireEvent.change(within(dialog).getByLabelText('Key name'), {
      target: { value: 'Laptop Passkey' },
    });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(registerWebAuthnCredential).toHaveBeenCalledWith(
        { id: 'credential-json' },
        'Laptop Passkey',
        'challenge-123',
      );
    });
    expect(notify).toHaveBeenCalledWith(
      'Security key registered successfully.',
      'success',
    );
  });

  it('removes an existing security key after confirmation', async () => {
    render(<WebAuthnSection />);

    fireEvent.click(await screen.findByRole('button', { name: 'Remove' }));

    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Remove' }));

    await waitFor(() => {
      expect(removeWebAuthnCredential).toHaveBeenCalledWith('cred-1');
    });
    expect(notify).toHaveBeenCalledWith('Security key removed.', 'success');
  });
});
