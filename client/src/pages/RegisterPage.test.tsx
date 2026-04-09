import { fireEvent, render } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import RegisterPage from './RegisterPage';

const { registerApi, getPublicConfig } = vi.hoisted(() => ({
  registerApi: vi.fn(),
  getPublicConfig: vi.fn(),
}));

const { resendVerificationEmail } = vi.hoisted(() => ({
  resendVerificationEmail: vi.fn(),
}));

vi.mock('../api/auth.api', () => ({
  registerApi,
  getPublicConfig,
}));

vi.mock('../api/email.api', () => ({
  resendVerificationEmail,
}));

vi.mock('../components/OAuthButtons', () => ({
  default: () => <div data-testid="oauth-buttons" />,
}));

function renderRegisterPage() {
  return render(
    <MemoryRouter initialEntries={['/register']}>
      <Routes>
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/login" element={<div>login</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    getPublicConfig.mockResolvedValue({ selfSignupEnabled: true });
    registerApi.mockResolvedValue({
      emailVerifyRequired: true,
      message: 'Check your email to verify your account.',
      recoveryKey: 'recovery-key',
    });
    resendVerificationEmail.mockResolvedValue(undefined);
  });

  it('shows an informational message when public registration is disabled', async () => {
    getPublicConfig.mockResolvedValue({ selfSignupEnabled: false });

    const view = renderRegisterPage();

    expect(
      await view.findByText(/Public registration is currently disabled/i),
    ).toBeInTheDocument();
    expect(view.getByText('Sign in')).toBeInTheDocument();
  });

  it('shows the verification state after a successful registration', async () => {
    const view = renderRegisterPage();

    fireEvent.change(await view.findByLabelText('Email'), {
      target: { value: 'new.user@example.com' },
    });
    fireEvent.change(view.getByLabelText('Password'), {
      target: { value: 'ArsenaleTemp91Qx' },
    });
    fireEvent.change(view.getByLabelText('Confirm Password'), {
      target: { value: 'ArsenaleTemp91Qx' },
    });

    fireEvent.click(view.getByRole('button', { name: 'Create Account' }));

    expect(
      await view.findByText('Check your email to verify your account.'),
    ).toBeInTheDocument();
    expect(view.getByText('Save your vault recovery key')).toBeInTheDocument();
    expect(view.getByText('recovery-key')).toBeInTheDocument();
    expect(
      view.getByRole('button', { name: /Resend verification email/i }),
    ).toBeInTheDocument();
  });
});
