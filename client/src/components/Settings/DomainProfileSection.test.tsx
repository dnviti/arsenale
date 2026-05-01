import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DomainProfileSection from './DomainProfileSection';
import { useNotificationStore } from '../../store/notificationStore';
import { useVaultStore } from '../../store/vaultStore';

const {
  getDomainProfile,
  updateDomainProfile,
  clearDomainProfile,
} = vi.hoisted(() => ({
  getDomainProfile: vi.fn(),
  updateDomainProfile: vi.fn(),
  clearDomainProfile: vi.fn(),
}));

vi.mock('../../api/user.api', () => ({
  getDomainProfile,
  updateDomainProfile,
  clearDomainProfile,
}));

describe('DomainProfileSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useNotificationStore.setState({ notification: null });
    useVaultStore.setState({
      unlocked: true,
      initialized: true,
      mfaUnlockAvailable: false,
      mfaUnlockMethods: [],
    });

    getDomainProfile.mockResolvedValue({
      domainName: 'CONTOSO',
      domainUsername: 'john.doe',
      hasDomainPassword: true,
    });
    updateDomainProfile.mockResolvedValue({
      domainName: 'CONTOSO',
      domainUsername: 'jane.doe',
      hasDomainPassword: true,
    });
    clearDomainProfile.mockResolvedValue({ success: true });
  });

  it('loads and displays the existing domain identity', async () => {
    render(<DomainProfileSection />);

    expect(await screen.findByText('Active domain identity')).toBeInTheDocument();
    expect(screen.getByText('CONTOSO')).toBeInTheDocument();
    expect(screen.getByText('john.doe')).toBeInTheDocument();
    expect(screen.getByText('Stored')).toBeInTheDocument();
  });

  it('saves edited domain profile values', async () => {
    render(<DomainProfileSection />);

    fireEvent.click(await screen.findByRole('button', { name: 'Edit' }));
    fireEvent.change(screen.getByLabelText('Domain username'), {
      target: { value: 'jane.doe' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(updateDomainProfile).toHaveBeenCalledWith({
        domainUsername: 'jane.doe',
      });
    });
    expect(useNotificationStore.getState().notification).toMatchObject({
      message: 'Domain profile updated',
      severity: 'success',
    });
  });
});
