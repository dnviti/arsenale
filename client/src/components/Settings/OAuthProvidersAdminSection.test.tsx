import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import OAuthProvidersAdminSection from './OAuthProvidersAdminSection';

const { getAuthProviderDetails } = vi.hoisted(() => ({
  getAuthProviderDetails: vi.fn(),
}));

vi.mock('../../api/admin.api', () => ({
  getAuthProviderDetails,
}));

describe('OAuthProvidersAdminSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    getAuthProviderDetails.mockResolvedValue([
      {
        key: 'GOOGLE',
        label: 'Google',
        enabled: true,
        providerName: 'Google Workspace',
      },
      {
        key: 'SAML',
        label: 'SAML',
        enabled: false,
      },
    ]);
  });

  it('renders provider state rows', async () => {
    render(<OAuthProvidersAdminSection />);

    expect(await screen.findByText('Google')).toBeInTheDocument();
    expect(screen.getByText('Enabled as Google Workspace')).toBeInTheDocument();
    expect(screen.getByText('Disabled')).toBeInTheDocument();
  });
});
