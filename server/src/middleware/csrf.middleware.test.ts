vi.mock('../config', () => ({
  config: {
    cookie: {
      csrfTokenName: 'csrf-token',
    },
  },
}));

import { Request, Response, NextFunction } from 'express';
import { validateCsrf } from './csrf.middleware';

function createMocks(headerToken?: string, cookieToken?: string) {
  const req = {
    headers: {} as Record<string, string>,
    cookies: {} as Record<string, string>,
  } as unknown as Request;

  if (headerToken !== undefined) {
    req.headers['x-csrf-token'] = headerToken;
  }
  if (cookieToken !== undefined) {
    (req.cookies as Record<string, string>)['csrf-token'] = cookieToken;
  }

  const res = {
    status: vi.fn().mockReturnThis(),
    json: vi.fn().mockReturnThis(),
  } as unknown as Response;

  const next: NextFunction = vi.fn();

  return { req, res, next };
}

describe('validateCsrf', () => {
  it('calls next() when header and cookie tokens match', () => {
    const token = 'valid-csrf-token-abc123';
    const { req, res, next } = createMocks(token, token);

    validateCsrf(req, res, next);

    expect(next).toHaveBeenCalled();
    expect(res.status).not.toHaveBeenCalled();
  });

  it('returns 403 when header token is missing', () => {
    const { req, res, next } = createMocks(undefined, 'some-cookie-token');

    validateCsrf(req, res, next);

    expect(res.status).toHaveBeenCalledWith(403);
    expect(res.json).toHaveBeenCalledWith({ error: 'CSRF token missing' });
    expect(next).not.toHaveBeenCalled();
  });

  it('returns 403 when cookie token is missing', () => {
    const { req, res, next } = createMocks('some-header-token', undefined);

    validateCsrf(req, res, next);

    expect(res.status).toHaveBeenCalledWith(403);
    expect(res.json).toHaveBeenCalledWith({ error: 'CSRF token missing' });
    expect(next).not.toHaveBeenCalled();
  });

  it('returns 403 when tokens are mismatched', () => {
    const { req, res, next } = createMocks('token-aaa', 'token-bbb');

    validateCsrf(req, res, next);

    expect(res.status).toHaveBeenCalledWith(403);
    expect(res.json).toHaveBeenCalledWith({ error: 'CSRF token mismatch' });
    expect(next).not.toHaveBeenCalled();
  });

  it('returns 403 when tokens have different lengths', () => {
    const { req, res, next } = createMocks('short', 'muchlongertoken');

    validateCsrf(req, res, next);

    expect(res.status).toHaveBeenCalledWith(403);
    expect(res.json).toHaveBeenCalledWith({ error: 'CSRF token mismatch' });
    expect(next).not.toHaveBeenCalled();
  });
});
