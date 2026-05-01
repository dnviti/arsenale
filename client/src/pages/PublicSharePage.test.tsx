import { fireEvent, render } from '@testing-library/react';
import type { ReactNode } from 'react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import PublicSharePage from './PublicSharePage';

vi.mock('@/components/auth/AuthLayout', () => ({
  default: ({ children, description, title }: { children: ReactNode; description: string; title: string }) => (
    <div>
      <h1>{title}</h1>
      <p>{description}</p>
      {children}
    </div>
  ),
}));

vi.mock('@/components/auth/AuthCodeInput', () => ({
  default: ({ label, onChange, value }: { label: string; onChange: (value: string) => void; value: string }) => (
    <label>
      {label}
      <input aria-label={label} value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  ),
}));

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

function formatExpiry(iso: string) {
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(iso));
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
        url: 'https://corp.example.com/login',
        domain: 'corp.example.com',
      },
    });

    const view = renderPublicSharePage();

    expect(await view.findByText('DB Password')).toBeInTheDocument();
    expect(view.getByText('Username')).toBeInTheDocument();
    expect(view.getByText('URL')).toBeInTheDocument();
    expect(view.getByText('URL').compareDocumentPosition(view.getByText('Domain')) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(view.getByText('Domain')).toBeInTheDocument();
    expect(view.getByText('corp.example.com')).toBeInTheDocument();
    expect(view.getByText('https://corp.example.com/login')).toBeInTheDocument();
    expect(accessExternalShare).toHaveBeenCalledWith('token-1', undefined);
  });

  it('renders certificate expiry as a formatted date', async () => {
    const expiresAt = '2026-01-15T12:00:00.000Z';

    getExternalShareInfo.mockResolvedValue({
      hasPin: false,
      isExpired: false,
      isExhausted: false,
      isRevoked: false,
      secretName: 'TLS Cert',
    });
    accessExternalShare.mockResolvedValue({
      secretName: 'TLS Cert',
      data: {
        type: 'CERTIFICATE',
        certificate: '-----BEGIN CERTIFICATE-----',
        privateKey: '-----BEGIN PRIVATE KEY-----',
        expiresAt,
      },
    });

    const view = renderPublicSharePage();

    expect(await view.findByText('TLS Cert')).toBeInTheDocument();
    expect(view.getByText('Certificate Expires')).toBeInTheDocument();
    expect(view.getByText(formatExpiry(expiresAt))).toBeInTheDocument();
    expect(view.queryByText(expiresAt)).not.toBeInTheDocument();
  });

  it('renders API key headers as pretty-printed shared payload data', async () => {
    getExternalShareInfo.mockResolvedValue({
      hasPin: false,
      isExpired: false,
      isExhausted: false,
      isRevoked: false,
      secretName: 'API Token',
    });
    accessExternalShare.mockResolvedValue({
      secretName: 'API Token',
      data: {
        type: 'API_KEY',
        apiKey: 'token-123',
        headers: {
          Authorization: 'Bearer token-123',
          'X-Trace-Id': 'abc-123',
        },
      },
    });

    const view = renderPublicSharePage();

    expect(await view.findByText('API Token')).toBeInTheDocument();
    expect(view.getByText('Headers')).toBeInTheDocument();
    expect(view.getByText('Headers').parentElement).toHaveTextContent('"Authorization": "Bearer token-123"');
    expect(view.getByText('Headers').parentElement).toHaveTextContent('"X-Trace-Id": "abc-123"');
  });

  it('keeps secure note content rendered as plain text', async () => {
    getExternalShareInfo.mockResolvedValue({
      hasPin: false,
      isExpired: false,
      isExhausted: false,
      isRevoked: false,
      secretName: 'Shared Note',
    });
    accessExternalShare.mockResolvedValue({
      secretName: 'Shared Note',
      data: {
        type: 'SECURE_NOTE',
        content: 'Plain note content',
      },
    });

    const view = renderPublicSharePage();

    expect(await view.findByText('Shared Note')).toBeInTheDocument();
    expect(view.getByText('Content')).toBeInTheDocument();
    expect(view.getByText('Plain note content')).toBeInTheDocument();
    expect(view.queryByRole('button', { name: 'Reveal' })).not.toBeInTheDocument();
    expect(view.queryByRole('button', { name: 'Hide' })).not.toBeInTheDocument();
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
