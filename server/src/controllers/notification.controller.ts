import { Response } from 'express';
import { AuthRequest, assertAuthenticated } from '../types';
import * as notificationService from '../services/notification.service';
import * as prefService from '../services/notificationPreference.service';
import { validatedQuery } from '../middleware/validate.middleware';
import type { NotificationQueryInput } from '../schemas/notification.schemas';
import { NotificationType } from '../lib/prisma';

export async function list(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const query = validatedQuery<NotificationQueryInput>(req);
  const result = await notificationService.listNotifications(
    req.user.userId,
    query.limit,
    query.offset
  );
  res.json(result);
}

export async function markRead(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  await notificationService.markAsRead(req.params.id as string, req.user.userId);
  res.json({ success: true });
}

export async function markAllRead(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  await notificationService.markAllAsRead(req.user.userId);
  res.json({ success: true });
}

export async function remove(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  await notificationService.deleteNotification(req.params.id as string, req.user.userId);
  res.json({ success: true });
}

export async function getPreferences(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const prefs = await prefService.getPreferences(req.user.userId);
  res.json(prefs);
}

export async function updatePreference(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const type = req.params.type as NotificationType;
  const { inApp, email } = req.body as { inApp?: boolean; email?: boolean };
  const pref = await prefService.upsertPreference(req.user.userId, type, { inApp, email });
  prefService.invalidateDeliveryCache(req.user.userId);
  res.json(pref);
}

export async function bulkUpdatePreferences(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { preferences } = req.body as {
    preferences: Array<{ type: NotificationType; inApp?: boolean; email?: boolean }>;
  };
  const prefs = await prefService.bulkUpsertPreferences(req.user.userId, preferences);
  prefService.invalidateDeliveryCache(req.user.userId);
  res.json(prefs);
}
