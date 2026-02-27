import { Response, NextFunction } from 'express';
import { z } from 'zod';
import { AuthRequest } from '../types';
import * as userService from '../services/user.service';
import * as auditService from '../services/audit.service';
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
  customColors: z.record(z.string(), z.string()).optional(),
  scrollback: z.number().int().min(100).max(10000).optional(),
  bellStyle: z.enum(['none', 'sound', 'visual']).optional(),
});

const rdpDefaultsSchema = z.object({
  colorDepth: z.union([z.literal(8), z.literal(16), z.literal(24)]).optional(),
  width: z.number().int().min(640).max(7680).optional(),
  height: z.number().int().min(480).max(4320).optional(),
  dpi: z.number().int().min(48).max(384).optional(),
  resizeMethod: z.enum(['display-update', 'reconnect']).optional(),
  qualityPreset: z.enum(['performance', 'balanced', 'quality', 'custom']).optional(),
  enableWallpaper: z.boolean().optional(),
  enableTheming: z.boolean().optional(),
  enableFontSmoothing: z.boolean().optional(),
  enableFullWindowDrag: z.boolean().optional(),
  enableDesktopComposition: z.boolean().optional(),
  enableMenuAnimations: z.boolean().optional(),
  forceLossless: z.boolean().optional(),
  disableAudio: z.boolean().optional(),
  enableAudioInput: z.boolean().optional(),
  security: z.enum(['any', 'nla', 'nla-ext', 'tls', 'rdp']).optional(),
  ignoreCert: z.boolean().optional(),
  serverLayout: z.string().optional(),
  console: z.boolean().optional(),
  timezone: z.string().optional(),
});

const uploadAvatarSchema = z.object({
  avatarData: z.string(),
});

const searchSchema = z.object({
  q: z.string().min(1).max(100),
  scope: z.enum(['tenant', 'team']).optional().default('tenant'),
  teamId: z.string().optional(),
}).refine(
  (data) => !(data.scope === 'team' && !data.teamId),
  { message: 'teamId is required when scope is team', path: ['teamId'] }
);

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
    auditService.log({
      userId: req.user!.userId, action: 'PROFILE_UPDATE',
      details: { fields: Object.keys(data) },
      ipAddress: req.ip,
    });
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.issues[0].message, 400));
    next(err);
  }
}

export async function changePassword(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const { oldPassword, newPassword } = changePasswordSchema.parse(req.body);
    const result = await userService.changePassword(req.user!.userId, oldPassword, newPassword);
    auditService.log({ userId: req.user!.userId, action: 'PASSWORD_CHANGE', ipAddress: req.ip });
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.issues[0].message, 400));
    next(err);
  }
}

export async function updateSshDefaults(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const data = sshDefaultsSchema.parse(req.body);
    const result = await userService.updateSshDefaults(req.user!.userId, data);
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.issues[0].message, 400));
    next(err);
  }
}

export async function updateRdpDefaults(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const data = rdpDefaultsSchema.parse(req.body);
    const result = await userService.updateRdpDefaults(req.user!.userId, data);
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.issues[0].message, 400));
    next(err);
  }
}

export async function uploadAvatar(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const { avatarData } = uploadAvatarSchema.parse(req.body);
    const result = await userService.uploadAvatar(req.user!.userId, avatarData);
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.issues[0].message, 400));
    next(err);
  }
}

export async function search(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const { q, scope, teamId } = searchSchema.parse(req.query);
    const results = await userService.searchUsers(
      req.user!.userId,
      req.user!.tenantId!,
      q,
      scope,
      teamId
    );
    res.json(results);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.issues[0].message, 400));
    next(err);
  }
}
