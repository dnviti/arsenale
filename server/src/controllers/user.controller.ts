import { Response, NextFunction } from 'express';
import { z } from 'zod';
import { AuthRequest } from '../types';
import * as userService from '../services/user.service';
import { AppError } from '../middleware/error.middleware';

const updateProfileSchema = z.object({
  username: z.string().min(1).max(50).optional(),
  email: z.string().email().optional(),
});

const changePasswordSchema = z.object({
  oldPassword: z.string(),
  newPassword: z.string().min(8),
});

const sshDefaultsSchema = z.object({
  fontFamily: z.string().optional(),
  fontSize: z.number().int().min(10).max(24).optional(),
  lineHeight: z.number().min(1.0).max(2.0).optional(),
  letterSpacing: z.number().min(0).max(5).optional(),
  cursorStyle: z.enum(['block', 'underline', 'bar']).optional(),
  cursorBlink: z.boolean().optional(),
  theme: z.string().optional(),
  customColors: z.record(z.string()).optional(),
  scrollback: z.number().int().min(100).max(10000).optional(),
  bellStyle: z.enum(['none', 'sound', 'visual']).optional(),
});

const uploadAvatarSchema = z.object({
  avatarData: z.string(),
});

export async function getProfile(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const result = await userService.getProfile(req.user!.userId);
    res.json(result);
  } catch (err) {
    next(err);
  }
}

export async function updateProfile(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const data = updateProfileSchema.parse(req.body);
    const result = await userService.updateProfile(req.user!.userId, data);
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.errors[0].message, 400));
    next(err);
  }
}

export async function changePassword(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const { oldPassword, newPassword } = changePasswordSchema.parse(req.body);
    const result = await userService.changePassword(req.user!.userId, oldPassword, newPassword);
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.errors[0].message, 400));
    next(err);
  }
}

export async function updateSshDefaults(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const data = sshDefaultsSchema.parse(req.body);
    const result = await userService.updateSshDefaults(req.user!.userId, data);
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.errors[0].message, 400));
    next(err);
  }
}

export async function uploadAvatar(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const { avatarData } = uploadAvatarSchema.parse(req.body);
    const result = await userService.uploadAvatar(req.user!.userId, avatarData);
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.errors[0].message, 400));
    next(err);
  }
}
