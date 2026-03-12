import crypto from 'crypto';
import { Response } from 'express';
import { config } from '../config';
import { parseExpiry } from './format';

export function setRefreshTokenCookie(res: Response, refreshToken: string): void {
  res.cookie(config.cookie.refreshTokenName, refreshToken, {
    httpOnly: true,
    secure: config.cookie.secure,
    sameSite: config.cookie.sameSite,
    path: config.cookie.path,
    maxAge: parseExpiry(config.jwtRefreshExpiresIn),
  });
}

export function setCsrfCookie(res: Response): string {
  const csrfToken = crypto.randomBytes(32).toString('hex');
  res.cookie(config.cookie.csrfTokenName, csrfToken, {
    httpOnly: false,
    secure: config.cookie.secure,
    sameSite: config.cookie.sameSite,
    path: '/',
    maxAge: parseExpiry(config.jwtRefreshExpiresIn),
  });
  return csrfToken;
}

export function clearAuthCookies(res: Response): void {
  res.clearCookie(config.cookie.refreshTokenName, {
    httpOnly: true,
    secure: config.cookie.secure,
    sameSite: config.cookie.sameSite,
    path: config.cookie.path,
  });
  res.clearCookie(config.cookie.csrfTokenName, {
    httpOnly: false,
    secure: config.cookie.secure,
    sameSite: config.cookie.sameSite,
    path: '/',
  });
}
