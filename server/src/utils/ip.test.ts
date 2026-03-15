import { getClientIp, getSocketClientIp } from './ip';

function mockReq(overrides: { headers?: Record<string, string | string[]>; ip?: string; remoteAddress?: string }) {
  return {
    headers: overrides.headers ?? {},
    ip: overrides.ip,
    socket: { remoteAddress: overrides.remoteAddress ?? '127.0.0.1' },
  } as any;
}

function mockSocket(overrides: { headers?: Record<string, string>; address?: string }) {
  return {
    handshake: {
      headers: overrides.headers ?? {},
      address: overrides.address ?? '127.0.0.1',
    },
  } as any;
}

describe('getClientIp', () => {
  it('returns public IPv4 from X-Forwarded-For', () => {
    const req = mockReq({ headers: { 'x-forwarded-for': '203.0.113.50' } });
    expect(getClientIp(req)).toBe('203.0.113.50');
  });

  it('picks the public IP when X-Forwarded-For contains private + public', () => {
    const req = mockReq({ headers: { 'x-forwarded-for': '10.0.0.1, 203.0.113.50, 192.168.1.1' } });
    expect(getClientIp(req)).toBe('203.0.113.50');
  });

  it('returns first IP when X-Forwarded-For contains only private IPs', () => {
    const req = mockReq({ headers: { 'x-forwarded-for': '10.0.0.1, 192.168.1.1' } });
    expect(getClientIp(req)).toBe('10.0.0.1');
  });

  it('strips ::ffff: prefix from IPv4-mapped IPv6 in X-Forwarded-For', () => {
    const req = mockReq({ headers: { 'x-forwarded-for': '::ffff:203.0.113.50' } });
    expect(getClientIp(req)).toBe('203.0.113.50');
  });

  it('falls back to req.ip when no X-Forwarded-For header', () => {
    const req = mockReq({ ip: '198.51.100.10' });
    expect(getClientIp(req)).toBe('198.51.100.10');
  });

  it('falls back to req.socket.remoteAddress when req.ip is undefined', () => {
    const req = mockReq({ ip: undefined, remoteAddress: '198.51.100.20' });
    expect(getClientIp(req)).toBe('198.51.100.20');
  });

  it('strips ::ffff: prefix from fallback ip', () => {
    const req = mockReq({ ip: '::ffff:198.51.100.10' });
    expect(getClientIp(req)).toBe('198.51.100.10');
  });
});

describe('private IP detection', () => {
  it('treats 172.16-31.x.x as private', () => {
    const req16 = mockReq({ headers: { 'x-forwarded-for': '172.16.0.1, 203.0.113.1' } });
    expect(getClientIp(req16)).toBe('203.0.113.1');

    const req31 = mockReq({ headers: { 'x-forwarded-for': '172.31.255.255, 203.0.113.1' } });
    expect(getClientIp(req31)).toBe('203.0.113.1');
  });

  it('treats 172.15.x.x as public (not in private range)', () => {
    const req = mockReq({ headers: { 'x-forwarded-for': '172.15.0.1, 10.0.0.1' } });
    expect(getClientIp(req)).toBe('172.15.0.1');
  });

  it('treats fe80: link-local IPv6 as private', () => {
    const req = mockReq({ headers: { 'x-forwarded-for': 'fe80::1, 203.0.113.1' } });
    expect(getClientIp(req)).toBe('203.0.113.1');
  });

  it('treats fd and fc (ULA) IPv6 addresses as private', () => {
    const reqFd = mockReq({ headers: { 'x-forwarded-for': 'fd12::1, 203.0.113.1' } });
    expect(getClientIp(reqFd)).toBe('203.0.113.1');

    const reqFc = mockReq({ headers: { 'x-forwarded-for': 'fc00::1, 203.0.113.1' } });
    expect(getClientIp(reqFc)).toBe('203.0.113.1');
  });
});

describe('getSocketClientIp', () => {
  it('extracts public IP from handshake x-forwarded-for header', () => {
    const socket = mockSocket({ headers: { 'x-forwarded-for': '10.0.0.1, 203.0.113.99' } });
    expect(getSocketClientIp(socket)).toBe('203.0.113.99');
  });

  it('falls back to handshake address when no x-forwarded-for', () => {
    const socket = mockSocket({ address: '198.51.100.5' });
    expect(getSocketClientIp(socket)).toBe('198.51.100.5');
  });

  it('strips ::ffff: prefix from handshake address', () => {
    const socket = mockSocket({ address: '::ffff:198.51.100.5' });
    expect(getSocketClientIp(socket)).toBe('198.51.100.5');
  });
});
