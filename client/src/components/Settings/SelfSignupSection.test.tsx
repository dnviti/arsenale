import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import SelfSignupSection from './SelfSignupSection';

const { getAppConfig, setSelfSignup } = vi.hoisted(() => ({
  getAppConfig: vi.fn(),
  setSelfSignup: vi.fn(),
}));

vi.mock('../../api/admin.api', () => ({
  getAppConfig,
  setSelfSignup,
}));

describe('SelfSignupSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    getAppConfig.mockResolvedValue({
      selfSignupEnabled: false,
      selfSignupEnvLocked: false,
    });
    setSelfSignup.mockResolvedValue({
      selfSignupEnabled: true,
      selfSignupEnvLocked: false,
    });
  });

  it('loads self-signup policy and saves the updated state', async () => {
    render(<SelfSignupSection />);

    const toggle = await screen.findByRole('switch', {
      name: 'Allow new users to register themselves',
    });

    fireEvent.click(toggle);

    await waitFor(() => {
      expect(setSelfSignup).toHaveBeenCalledWith(true);
    });
  });
});
