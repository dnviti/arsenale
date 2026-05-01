import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import SamlConfigSection from './SamlConfigSection';

const { getAuthProviderDetails } = vi.hoisted(() => ({
  getAuthProviderDetails: vi.fn(),
}));

vi.mock('../../api/admin.api', () => ({
  getAuthProviderDetails,
}));

describe('SamlConfigSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    getAuthProviderDetails.mockResolvedValue([
      {
        key: 'saml',
        label: 'SAML',
        enabled: true,
        providerName: 'Okta',
      },
    ]);
  });

  it('shows configured SAML provider state and metadata link', async () => {
    render(<SamlConfigSection />);

    expect(await screen.findByText('Okta')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'View SP Metadata' })).toHaveAttribute(
      'href',
      '/api/saml/metadata',
    );
  });
});
