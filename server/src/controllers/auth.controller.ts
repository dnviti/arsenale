import { Request, Response, NextFunction } from 'express';
import { z } from 'zod';
import * as authService from '../services/auth.service';
import { AppError } from '../middleware/error.middleware';
import { config } from '../config';

const registerSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
});

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string(),
});

const refreshSchema = z.object({
  refreshToken: z.string(),
});

export async function register(req: Request, res: Response, next: NextFunction) {
  try {
    const { email, password } = registerSchema.parse(req.body);
    const user = await authService.register(email, password);
    res.status(201).json(user);
  } catch (err) {
    if (err instanceof z.ZodError) {
      return next(new AppError(err.issues[0].message, 400));
    }
    if (err instanceof Error && err.message === 'Email already registered') {
      return next(new AppError(err.message, 409));
    }
    next(err);
  }
}

const verifyTotpSchema = z.object({
  tempToken: z.string(),
  code: z.string().length(6).regex(/^\d{6}$/),
});

export async function login(req: Request, res: Response, next: NextFunction) {
  try {
    const { email, password } = loginSchema.parse(req.body);
    const result = await authService.login(email, password);

    if (result.requiresTOTP) {
      res.json({ requiresTOTP: true, tempToken: result.tempToken });
    } else {
      res.json({
        accessToken: result.accessToken,
        refreshToken: result.refreshToken,
        user: result.user,
      });
    }
  } catch (err) {
    if (err instanceof z.ZodError) {
      return next(new AppError(err.issues[0].message, 400));
    }
    if (err instanceof AppError) {
      return next(err);
    }
    if (err instanceof Error && err.message === 'Invalid email or password') {
      return next(new AppError(err.message, 401));
    }
    next(err);
  }
}

export async function verifyTotp(req: Request, res: Response, next: NextFunction) {
  try {
    const { tempToken, code } = verifyTotpSchema.parse(req.body);
    const result = await authService.verifyTotp(tempToken, code);
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) {
      return next(new AppError('Invalid code format', 400));
    }
    if (err instanceof Error) {
      if (err.message === 'Invalid TOTP code') return next(new AppError(err.message, 401));
      if (err.message.includes('token')) return next(new AppError(err.message, 401));
      if (err.message === '2FA verification failed') return next(new AppError(err.message, 401));
    }
    next(err);
  }
}

export async function refresh(req: Request, res: Response, next: NextFunction) {
  try {
    const { refreshToken } = refreshSchema.parse(req.body);
    const result = await authService.refreshAccessToken(refreshToken);
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) {
      return next(new AppError(err.issues[0].message, 400));
    }
    if (err instanceof Error && err.message.includes('refresh token')) {
      return next(new AppError(err.message, 401));
    }
    next(err);
  }
}

export async function logout(req: Request, res: Response, next: NextFunction) {
  try {
    const { refreshToken } = refreshSchema.parse(req.body);
    await authService.logout(refreshToken);
    res.json({ success: true });
  } catch (err) {
    next(err);
  }
}

const verifyEmailSchema = z.object({
  token: z.string().length(64),
});

const resendVerificationSchema = z.object({
  email: z.string().email(),
});

export async function verifyEmail(req: Request, res: Response, next: NextFunction) {
  try {
    const { token } = verifyEmailSchema.parse(req.query);
    await authService.verifyEmail(token);
    res.redirect(`${config.clientUrl}/login?verified=true`);
  } catch (err) {
    if (err instanceof z.ZodError) {
      return res.redirect(`${config.clientUrl}/login?verifyError=${encodeURIComponent('Invalid verification link.')}`);
    }
    if (err instanceof AppError) {
      return res.redirect(`${config.clientUrl}/login?verifyError=${encodeURIComponent(err.message)}`);
    }
    next(err);
  }
}

export async function resendVerification(req: Request, res: Response, next: NextFunction) {
  try {
    const { email } = resendVerificationSchema.parse(req.body);
    await authService.resendVerification(email);
    res.json({ message: 'If an account exists with this email, a verification link has been sent.' });
  } catch (err) {
    if (err instanceof z.ZodError) {
      return next(new AppError('Invalid email format', 400));
    }
    next(err);
  }
}
