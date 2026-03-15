import jwt from 'jsonwebtoken';

const TEST_SECRET = 'test-secret-key-for-jwt-tests';

vi.mock('../config', () => ({
  config: {
    jwtSecret: 'test-secret-key-for-jwt-tests',
  },
}));

import { verifyJwt } from './jwt';

describe('verifyJwt', () => {
  it('returns payload for a valid HS256 token', () => {
    const payload = { userId: '123', role: 'admin' };
    const token = jwt.sign(payload, TEST_SECRET, { algorithm: 'HS256' });

    const result = verifyJwt<typeof payload>(token);

    expect(result.userId).toBe('123');
    expect(result.role).toBe('admin');
  });

  it('throws for an expired token', () => {
    const token = jwt.sign({ userId: '1' }, TEST_SECRET, {
      algorithm: 'HS256',
      expiresIn: -10,
    });

    expect(() => verifyJwt(token)).toThrow();
  });

  it('throws for a tampered signature', () => {
    const token = jwt.sign({ userId: '1' }, TEST_SECRET, { algorithm: 'HS256' });
    // Flip the last character of the signature
    const parts = token.split('.');
    const lastChar = parts[2].slice(-1);
    parts[2] = parts[2].slice(0, -1) + (lastChar === 'A' ? 'B' : 'A');
    const tampered = parts.join('.');

    expect(() => verifyJwt(tampered)).toThrow();
  });

  it('throws for a token signed with a different secret', () => {
    const token = jwt.sign({ userId: '1' }, 'wrong-secret', { algorithm: 'HS256' });

    expect(() => verifyJwt(token)).toThrow();
  });

  it('rejects tokens not using HS256 (algorithm pinning)', () => {
    // Create a token with "none" algorithm by crafting the header manually
    const header = Buffer.from(JSON.stringify({ alg: 'none', typ: 'JWT' })).toString('base64url');
    const payload = Buffer.from(JSON.stringify({ userId: '1' })).toString('base64url');
    const forgedToken = `${header}.${payload}.`;

    expect(() => verifyJwt(forgedToken)).toThrow();
  });

  it('throws for an invalid/malformed token string', () => {
    expect(() => verifyJwt('not-a-jwt')).toThrow();
    expect(() => verifyJwt('')).toThrow();
    expect(() => verifyJwt('a.b.c')).toThrow();
  });
});
