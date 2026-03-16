import crypto from 'crypto';
import { Request, Response, NextFunction } from 'express';
import { config } from '../config';

/**
 * Detect extension client context: the request carries a Bearer token but has
 * no cookies at all.  Browser extensions operate from a different origin, so
 * httpOnly cookies are never attached.  Because extensions are immune to CSRF
 * (cross-site requests cannot reach chrome-extension:// origins), CSRF
 * validation is safely skipped.
 */
function isExtensionContext(req: Request): boolean {
  const authHeader = req.headers['authorization'] as string | undefined;
  const hasBearerToken = !!authHeader && authHeader.startsWith('Bearer ');
  const hasCookies = !!req.cookies && Object.keys(req.cookies).length > 0;
  return hasBearerToken && !hasCookies;
}

export function validateCsrf(req: Request, res: Response, next: NextFunction): void {
  // Skip CSRF validation for extension clients (Bearer-only, no cookies)
  if (isExtensionContext(req)) {
    return next();
  }

  const headerToken = req.headers['x-csrf-token'] as string | undefined;
  const cookieToken = req.cookies?.[config.cookie.csrfTokenName] as string | undefined;

  if (!headerToken || !cookieToken) {
    res.status(403).json({ error: 'CSRF token missing' });
    return;
  }

  const headerBuf = Buffer.from(headerToken);
  const cookieBuf = Buffer.from(cookieToken);

  if (headerBuf.length !== cookieBuf.length || !crypto.timingSafeEqual(headerBuf, cookieBuf)) {
    res.status(403).json({ error: 'CSRF token mismatch' });
    return;
  }

  next();
}
