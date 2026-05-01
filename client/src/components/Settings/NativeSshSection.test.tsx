import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import NativeSshSection from './NativeSshSection';

const { getSshProxyStatus } = vi.hoisted(() => ({
  getSshProxyStatus: vi.fn(),
}));

vi.mock('../../api/sessions.api', () => ({
  getSshProxyStatus,
}));

describe('NativeSshSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    getSshProxyStatus.mockResolvedValue({
      enabled: true,
      port: 2222,
      listening: true,
      activeSessions: 3,
      allowedAuthMethods: ['password', 'token'],
    });
  });

  it('renders the SSH proxy status and config snippet', async () => {
    render(<NativeSshSection />);

    expect(await screen.findByText('Running')).toBeInTheDocument();
    expect(screen.getByText('2222')).toBeInTheDocument();
    expect(screen.getByText('password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Copy Snippet' })).toBeInTheDocument();
    expect(screen.getByDisplayValue(/Host arsenale-proxy/)).toBeInTheDocument();
  });
});
