import { fireEvent, render } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import PublicSharePage from './PublicSharePage';

const { getExternalShareInfo, accessExternalShare } = vi.hoisted(() => ({
  getExternalShareInfo: vi.fn(),
  accessExternalShare: vi.fn(),
}));

vi.mock('../api/secrets.api', () => ({
  getExternalShareInfo,
  accessExternalShare,
}));

function renderPublicSharePage(path = '/share/token-1') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/share/:token" element={<PublicSharePage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('PublicSharePage', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('auto-accesses a valid share that does not require a PIN', async () => {
    getExternalShareInfo.mockResolvedValue({
      hasPin: false,
      isExpired: false,
      isExhausted: false,
      isRevoked: false,
      secretName: 'DB Password',
    });
    accessExternalShare.mockResolvedValue({
      secretName: 'DB Password',
      data: {
        type: 'LOGIN',
        username: 'admin',
        password: 'secret-password',
      },
    });

    const view = renderPublicSharePage();

    expect(await view.findByText('DB Password')).toBeInTheDocument();
    expect(view.getByText('Username')).toBeInTheDocument();
    expect(accessExternalShare).toHaveBeenCalledWith('token-1', undefined);
  });

  it('validates PIN format before submitting access', async () => {
    getExternalShareInfo.mockResolvedValue({
      hasPin: true,
      isExpired: false,
      isExhausted: false,
      isRevoked: false,
      secretName: 'Shared API Key',
    });

    const view = renderPublicSharePage();

    await view.findByText('Shared API Key');

    fireEvent.change(view.getByLabelText('PIN'), {
      target: { value: '123' },
    });
    fireEvent.click(view.getByRole('button', { name: 'Decrypt' }));

    expect(await view.findByText('PIN must be 4-8 digits')).toBeInTheDocument();
    expect(accessExternalShare).not.toHaveBeenCalled();
  });
});
