import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import EmailProviderSection from './EmailProviderSection';
import { useNotificationStore } from '../../store/notificationStore';

const { getEmailStatus, sendTestEmail } = vi.hoisted(() => ({
  getEmailStatus: vi.fn(),
  sendTestEmail: vi.fn(),
}));

vi.mock('../../api/admin.api', () => ({
  getEmailStatus,
  sendTestEmail,
}));

describe('EmailProviderSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useNotificationStore.setState({ notification: null });

    getEmailStatus.mockResolvedValue({
      provider: 'smtp',
      configured: true,
      from: 'noreply@example.com',
      host: 'smtp.example.com',
      port: 587,
      secure: true,
    });
    sendTestEmail.mockResolvedValue({
      success: true,
      message: 'Test email sent',
    });
  });

  it('loads provider status and sends a test email', async () => {
    render(<EmailProviderSection />);

    expect(await screen.findByText('SMTP')).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('Recipient email'), {
      target: { value: 'admin@example.com' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Send Test' }));

    await waitFor(() => {
      expect(sendTestEmail).toHaveBeenCalledWith('admin@example.com');
    });
    expect(useNotificationStore.getState().notification).toMatchObject({
      message: 'Test email sent',
      severity: 'success',
    });
  });
});
