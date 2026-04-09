import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ProfileSection from './ProfileSection';
import { useAuthStore } from '../../store/authStore';
import { useNotificationStore } from '../../store/notificationStore';

const {
  getProfile,
  updateProfile,
  initiateEmailChange,
  confirmEmailChange,
  uploadAvatar,
} = vi.hoisted(() => ({
  getProfile: vi.fn(),
  updateProfile: vi.fn(),
  initiateEmailChange: vi.fn(),
  confirmEmailChange: vi.fn(),
  uploadAvatar: vi.fn(),
}));

vi.mock('../../api/user.api', () => ({
  getProfile,
  updateProfile,
  initiateEmailChange,
  confirmEmailChange,
  uploadAvatar,
}));

vi.mock('../common/IdentityVerification', () => ({
  default: () => <div>Identity Verification</div>,
}));

describe('ProfileSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    localStorage.clear();

    useAuthStore.setState({
      accessToken: 'access-token',
      csrfToken: 'csrf-token',
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        username: 'admin',
        avatarData: null,
      },
      isAuthenticated: true,
    });
    useNotificationStore.setState({ notification: null });

    getProfile.mockResolvedValue({
      id: 'user-1',
      email: 'admin@example.com',
      username: 'admin',
      avatarData: null,
      sshDefaults: null,
      rdpDefaults: null,
      hasPassword: false,
      vaultSetupComplete: true,
      oauthAccounts: [],
      createdAt: '2026-04-07T00:00:00Z',
    });
    updateProfile.mockResolvedValue({
      id: 'user-1',
      email: 'admin@example.com',
      username: 'admin-renamed',
      avatarData: null,
      sshDefaults: null,
      rdpDefaults: null,
      hasPassword: false,
      vaultSetupComplete: true,
      oauthAccounts: [],
      createdAt: '2026-04-07T00:00:00Z',
    });
    initiateEmailChange.mockResolvedValue({ flow: 'dual-otp' });
    confirmEmailChange.mockResolvedValue({ email: 'admin@example.com' });
    uploadAvatar.mockResolvedValue({ id: 'user-1', avatarData: 'avatar-data' });
  });

  it('loads the profile and reports password availability to the parent', async () => {
    const onHasPasswordResolved = vi.fn();

    render(
      <ProfileSection
        onHasPasswordResolved={onHasPasswordResolved}
        linkedProvider={null}
      />,
    );

    expect(await screen.findByDisplayValue('admin')).toBeInTheDocument();
    expect(screen.getByDisplayValue('admin@example.com')).toBeInTheDocument();
    expect(onHasPasswordResolved).toHaveBeenCalledWith(false);
  });

  it('saves username changes and updates the auth store', async () => {
    render(
      <ProfileSection
        onHasPasswordResolved={() => {}}
        linkedProvider={null}
      />,
    );

    const usernameInput = await screen.findByLabelText('Username');
    fireEvent.change(usernameInput, { target: { value: 'admin-renamed' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save Profile' }));

    await waitFor(() => {
      expect(updateProfile).toHaveBeenCalledWith({ username: 'admin-renamed' });
    });
    expect(useAuthStore.getState().user?.username).toBe('admin-renamed');
    expect(useNotificationStore.getState().notification).toMatchObject({
      message: 'Profile updated successfully',
      severity: 'success',
    });
  });
});
